// Package deployer for grid deployer
package deployer

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	proxy "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
	proxyTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zosbase/pkg/gridtypes"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"golang.org/x/sync/errgroup"
)

// MockDeployer to be used for any deployer in mock testing
type MockDeployer interface { // TODO: Change Name && separate them
	Deploy(ctx context.Context,
		oldDeploymentIDs map[uint32]uint64,
		newDeployments map[uint32]zos.Deployment,
		newDeploymentSolutionProvider map[uint32]*uint64,
	) (map[uint32]uint64, error)

	Cancel(ctx context.Context,
		contractID uint64,
	) error

	GetDeployments(ctx context.Context, dls map[uint32]uint64) (map[uint32]zos.Deployment, error)
	BatchDeploy(ctx context.Context,
		deployments map[uint32][]zos.Deployment,
		deploymentsSolutionProvider map[uint32][]*uint64,
	) (map[uint32][]zos.Deployment, error)
}

// Deployer to be used for any deployer
type Deployer struct {
	identity        substrate.Identity
	twinID          uint32
	gridProxyClient proxy.Client
	ncPool          client.NodeClientGetter
	revertOnFailure bool
	substrateConn   subi.SubstrateExt
	tracer          trace.Tracer
}

// NewDeployer returns a new deployer
func NewDeployer(
	tfPluginClient TFPluginClient,
	revertOnFailure bool,
) Deployer {
	deployer := Deployer{
		tfPluginClient.Identity,
		tfPluginClient.TwinID,
		tfPluginClient.GridProxyClient,
		tfPluginClient.NcPool,
		revertOnFailure,
		tfPluginClient.SubstrateConn,
		noop.NewTracerProvider().Tracer("no-op"),
	}

	if tfPluginClient.traceProvider != nil {
		deployer.tracer = tfPluginClient.traceProvider.Tracer("grid-deployer")
	}

	return deployer
}

// Deploy deploys or updates a new deployment given the old deployments' IDs
func (d *Deployer) Deploy(ctx context.Context,
	oldDeploymentIDs map[uint32]uint64,
	newDeployments map[uint32]zos.Deployment,
	newDeploymentSolutionProvider map[uint32]*uint64,
) (map[uint32]uint64, error) {

	ctx, span := d.tracer.Start(ctx, "deployer.Deploy",
		trace.WithAttributes(
			attribute.Int("old_deployments_count", len(oldDeploymentIDs)),
			attribute.Int("new_deployments_count", len(newDeployments)),
			attribute.Bool("revert_on_failure", d.revertOnFailure),
		))
	defer span.End()

	span.AddEvent("fetching old deployments")
	oldDeployments, oldErr := d.GetDeployments(ctx, oldDeploymentIDs)
	if oldErr == nil {
		span.AddEvent("validating deployments")
		// check resources only when old deployments are readable
		// being readable means it's a fresh deployment or an update with good nodes
		// this is done to avoid preventing deletion of deployments on dead nodes
		if err := d.Validate(ctx, oldDeployments, newDeployments); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "validation failed")
			return oldDeploymentIDs, err
		}
	}

	// ignore oldErr until we need oldDeployments
	span.AddEvent("starting deployment process")
	currentDeployments, err := d.deploy(ctx, oldDeploymentIDs, newDeployments, newDeploymentSolutionProvider, d.revertOnFailure)

	if err != nil && d.revertOnFailure {
		span.AddEvent("deployment failed, revert on failure triggered")
		if oldErr != nil {
			err = errors.Wrapf(err, "failed to fetch deployment objects to revert deployments: %s; try again", oldErr)
			span.RecordError(err)
			span.SetStatus(codes.Error, "revert failed: old deployments unavailable")
			return currentDeployments, err
		}

		currentDls, rerr := d.deploy(ctx, currentDeployments, oldDeployments, newDeploymentSolutionProvider, false)
		if rerr != nil {
			err = errors.Wrapf(err, "failed to revert deployments: %s; try again", rerr)
			span.RecordError(err)
			span.SetStatus(codes.Error, "revert failed")
			return currentDls, err
		}
		return currentDls, err
	}

	if err == nil {
		span.AddEvent("deployment successful",
			trace.WithAttributes(attribute.Int("deployments_count", len(currentDeployments))))
	}

	return currentDeployments, err
}

func spanErrorAndEnd(span trace.Span, description string, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, description)
	span.End()
}

func (d *Deployer) deploy(
	ctx context.Context,
	oldDeployments map[uint32]uint64,
	newDeployments map[uint32]zos.Deployment,
	newDeploymentSolutionProvider map[uint32]*uint64,
	revertOnFailure bool,
) (currentDeployments map[uint32]uint64, err error) {

	ctx, span := d.tracer.Start(ctx, "Deployer.deploy")
	defer span.End()

	currentDeployments = make(map[uint32]uint64)
	for nodeID, contractID := range oldDeployments {
		currentDeployments[nodeID] = contractID
	}
	// deletions
	span.AddEvent("processing deletions")
	for node, contractID := range oldDeployments {
		if _, ok := newDeployments[node]; !ok {
			span.AddEvent("canceling_contract",
				trace.WithAttributes(
					attribute.Int("node", int(node)),
					attribute.Int("contract_id", int(contractID)),
				))

			err = d.substrateConn.EnsureContractCanceled(d.identity, contractID)
			if err != nil && !strings.Contains(err.Error(), "ContractNotExists") {
				span.RecordError(err)
				span.SetStatus(codes.Error, "failed to delete deployment")
				return currentDeployments, errors.Wrap(err, "failed to delete deployment")
			}
			delete(currentDeployments, node)
		}
	}

	// creations
	span.AddEvent("processing creations")
	for node, dl := range newDeployments {
		if _, ok := oldDeployments[node]; !ok {

			nodeCtx, nodeSpan := d.tracer.Start(ctx, "Deployer.create_deployment",
				trace.WithAttributes(attribute.Int("node", int(node))))

			nodeClient, err := d.ncPool.GetNodeClient(d.substrateConn, node)
			if err != nil {
				spanErrorAndEnd(nodeSpan, "failed to get node client", err)
				span.RecordError(err)
				return currentDeployments, errors.Wrap(err, "failed to get node client")
			}

			if err := dl.Sign(d.twinID, d.identity); err != nil {
				spanErrorAndEnd(nodeSpan, "failed to sign deployment", err)
				return currentDeployments, errors.Wrap(err, "error signing deployment")
			}

			if err := dl.Valid(); err != nil {
				spanErrorAndEnd(nodeSpan, "invalid deployment", err)
				return currentDeployments, errors.Wrap(err, "deployment is invalid")
			}

			hash, err := dl.ChallengeHash()
			log.Debug().Bytes("HASH", hash)

			if err != nil {
				spanErrorAndEnd(nodeSpan, "failed to create hash", err)
				return currentDeployments, errors.Wrap(err, "failed to create hash")
			}

			hashHex := hex.EncodeToString(hash)

			publicIPCount, err := CountDeploymentPublicIPs(dl)
			if err != nil {
				spanErrorAndEnd(nodeSpan, "failed to count public IPs", err)
				return currentDeployments, errors.Wrap(err, "failed to count deployment public IPs")
			}
			log.Debug().Uint32("Number of public ips", publicIPCount)

			nodeSpan.SetAttributes(
				attribute.String("hash", hashHex),
				attribute.Int("public_ip_count", int(publicIPCount)),
			)

			contractReused := false
			contractID, err := d.substrateConn.CreateNodeContract(d.identity, node, dl.Metadata, hashHex, publicIPCount, newDeploymentSolutionProvider[node])
			if err != nil {
				if !strings.Contains(err.Error(), "ContractIsNotUnique") {
					spanErrorAndEnd(nodeSpan, "failed to create contract", err)
					return currentDeployments, errors.Wrapf(err, "failed to create contract on node %d", node)
				}

				contractID, err = d.substrateConn.GetContractWithHash(d.identity, node, []byte(hashHex))
				if err != nil {
					spanErrorAndEnd(nodeSpan, "failed to find existing contract with hash", err)
					return currentDeployments, errors.Wrapf(err, "failed to find existing contract on node %d", node)
				}

				contract, err := d.substrateConn.GetContract(contractID)
				if err != nil {
					spanErrorAndEnd(nodeSpan, "failed to get existing contract by id", err)
					return currentDeployments, errors.Wrapf(err, "failed to get existing contract on node %d", node)
				}

				if contract.State.IsDeleted {
					err = errors.Errorf("contract %d is not active", contractID)
					spanErrorAndEnd(nodeSpan, "contract is deleted", err)
					return currentDeployments, err
				}

				log.Info().
					Uint32("node", node).
					Uint64("contractID", contractID).
					Msg("reusing existing contract")
				contractReused = true
			}

			log.Debug().Uint64("returned contract ID", contractID)
			dl.ContractID = contractID

			nodeSpan.SetAttributes(attribute.Int("contract_id", int(contractID)))

			// Update deployment with contract ID and send to node
			nodeSpan.AddEvent("sending deployment to node")
			err = nodeClient.DeploymentDeploy(nodeCtx, dl)
			if err != nil {
				// If deployment exists, continue as already deployed
				if !contractReused || !strings.Contains(err.Error(), "exists") {
					nodeSpan.AddEvent("deployment failed, canceling contract")
					// Other deployment error: cancel contract
					rerr := d.substrateConn.EnsureContractCanceled(d.identity, dl.ContractID)
					if rerr != nil {
						spanErrorAndEnd(nodeSpan, "failed to cancel contract after deployment error", rerr)
						return currentDeployments, errors.Wrapf(err, "error cancelling contract: %s; you must cancel it manually (id: %d)", rerr, dl.ContractID)
					}

					spanErrorAndEnd(nodeSpan, "failed to send deployment to node", err)
					return currentDeployments, errors.Wrapf(err, "error sending deployment to node %d", node)
				}

				log.Debug().
					Uint32("node", node).
					Uint64("contractID", dl.ContractID).
					Msg("deployment exists, continuing as already deployed")
			}

			// Deployment sent successfully, wait for it
			currentDeployments[node] = dl.ContractID
			newWorkloadVersions := make(map[string]uint32)
			for _, w := range dl.Workloads {
				newWorkloadVersions[w.Name] = 0
			}

			nodeSpan.AddEvent("waiting for deployment")
			if err = d.Wait(nodeCtx, nodeClient, dl.ContractID, newWorkloadVersions); err != nil {
				spanErrorAndEnd(nodeSpan, "waiting for deployment failed", err)
				return currentDeployments, errors.Wrap(err, "error waiting deployment")
			}
			nodeSpan.End()

		}
	}

	// updates
	span.AddEvent("processing updates")
	for node, dl := range newDeployments {
		if oldDeploymentID, ok := oldDeployments[node]; ok {

			nodeCtx, nodeSpan := d.tracer.Start(ctx, "Deployer.update_deployment",
				trace.WithAttributes(attribute.Int("node", int(node)),
					attribute.Int("old_contract_id", int(oldDeploymentID)),
				))

			client, err := d.ncPool.GetNodeClient(d.substrateConn, node)
			if err != nil {
				spanErrorAndEnd(nodeSpan, "failed to get node client", err)
				return currentDeployments, errors.Wrap(err, "failed to get node client")
			}

			nodeSpan.AddEvent("fetching old deployment")
			oldDl, err := client.DeploymentGet(nodeCtx, oldDeploymentID)
			if err != nil {
				spanErrorAndEnd(nodeSpan, "failed to get old deployment", err)
				return currentDeployments, errors.Wrap(err, "failed to get old deployment to update it")
			}

			matchOldVersions(&oldDl, &dl)

			oldDeploymentHash, err := HashDeployment(oldDl)
			if err != nil {
				spanErrorAndEnd(nodeSpan, "failed to get deployment hash", err)
				return currentDeployments, errors.Wrap(err, "could not get deployment hash")
			}

			newDeploymentHash, err := HashDeployment(dl)
			if err != nil {
				spanErrorAndEnd(nodeSpan, "failed to get new deployment hash", err)
				return currentDeployments, errors.Wrap(err, "could not get deployment hash")
			}

			nodeSpan.SetAttributes(
				attribute.String("old_hash", oldDeploymentHash),
				attribute.String("new_hash", newDeploymentHash),
			)

			if oldDeploymentHash == newDeploymentHash && SameWorkloadsNames(dl, oldDl) {
				nodeSpan.End()

				continue
			}

			nodeSpan.AddEvent("assigning versions")
			newWorkloadsVersions, err := assignVersions(&oldDl, &dl)
			if err != nil {
				spanErrorAndEnd(nodeSpan, "failed to assign versions", err)
				return currentDeployments, errors.Wrapf(err, "failed to assign new versions to deployment with contract %d", oldDeploymentID)
			}

			dl.ContractID = oldDl.ContractID

			if err := dl.Sign(d.twinID, d.identity); err != nil {
				spanErrorAndEnd(nodeSpan, "failed to sign deployments", err)
				return currentDeployments, errors.Wrap(err, "error signing deployment")
			}

			if err := dl.Valid(); err != nil {
				spanErrorAndEnd(nodeSpan, "invalid deployment", err)
				return currentDeployments, errors.Wrap(err, "deployment is invalid")
			}

			log.Debug().Interface("deployment", dl)
			hash, err := dl.ChallengeHash()
			if err != nil {
				spanErrorAndEnd(nodeSpan, "failed to create hash", err)
				return currentDeployments, errors.Wrap(err, "failed to create hash")
			}
			hashHex := hex.EncodeToString(hash)
			log.Debug().Str("HASH", hashHex)

			// TODO: Destroy and create if publicIPCount is changed
			// publicIPCount, err := countDeploymentPublicIPs(dl)
			nodeSpan.AddEvent("updating contract on chain")
			contractID, err := d.substrateConn.UpdateNodeContract(d.identity, dl.ContractID, dl.Metadata, hashHex)
			if err != nil {
				spanErrorAndEnd(nodeSpan, "failed to update contract", err)
				return currentDeployments, errors.Wrap(err, "failed to update deployment")
			}

			dl.ContractID = contractID

			nodeSpan.SetAttributes(attribute.Int("new_contract_id", int(contractID)))

			nodeSpan.AddEvent("sending update to node")
			err = client.DeploymentUpdate(nodeCtx, dl)
			if err != nil {
				spanErrorAndEnd(nodeSpan, "failed to send update to node", err)
				// cancel previous contract
				return currentDeployments, errors.Wrapf(err, "failed to send deployment update request to node %d", node)
			}

			currentDeployments[node] = dl.ContractID

			nodeSpan.AddEvent("waiting_for_update_completion")
			err = d.Wait(nodeCtx, client, dl.ContractID, newWorkloadsVersions)
			if err != nil {
				spanErrorAndEnd(nodeSpan, "waiting for update failed", err)
				return currentDeployments, errors.Wrap(err, "error waiting deployment")
			}

			nodeSpan.End()
		}
	}

	span.SetAttributes(
		attribute.Int("final_deployments_count", len(currentDeployments)),
	)

	return currentDeployments, nil
}

// Cancel cancels an old deployment not given in the new deployments
func (d *Deployer) Cancel(ctx context.Context,
	contractID uint64,
) error {

	_, span := d.tracer.Start(ctx, "Deployer.Cancel",
		trace.WithAttributes(
			attribute.Int("contract_id", int(contractID)),
		))
	defer span.End()

	span.AddEvent("canceling_contract")
	err := d.substrateConn.EnsureContractCanceled(d.identity, contractID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to cancel contract")
		return errors.Wrapf(err, "failed to delete deployment: %d", contractID)
	}

	return nil
}

// GetDeployments returns deployments from a map of nodes IDs and deployments IDs
func (d *Deployer) GetDeployments(ctx context.Context, dls map[uint32]uint64) (map[uint32]zos.Deployment, error) {

	ctx, span := d.tracer.Start(ctx, "Deployer.GetDeployments",
		trace.WithAttributes(
			attribute.Int("deployments_count", len(dls)),
		))
	defer span.End()

	res := make(map[uint32]zos.Deployment)
	span.AddEvent("starting fetch deployments")

	for nodeID, dlID := range dls {
		span.AddEvent("fetching a single deployment", trace.WithAttributes(
			attribute.Int("node_ID", int(nodeID)),
			attribute.Int("deployment_ID", int(dlID)),
		))

		nc, err := d.ncPool.GetNodeClient(d.substrateConn, nodeID)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to get node client")
			return nil, errors.Wrapf(err, "failed to get a client for node %d", nodeID)
		}

		dl, err := nc.DeploymentGet(ctx, dlID)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to get deployment")
			return nil, errors.Wrapf(err, "failed to get deployment %d of node %d", dlID, nodeID)
		}

		res[nodeID] = dl

	}

	span.SetAttributes(attribute.Int("fetched_count", len(res)))

	return res, nil
}

// Progress struct for checking progress
type Progress struct {
	time    time.Time
	stateOk int
}

func getExponentialBackoff(initialInterval time.Duration, multiplier float64, maxInterval time.Duration, maxElapsedTime time.Duration) *backoff.ExponentialBackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = initialInterval
	b.Multiplier = multiplier
	b.MaxInterval = maxInterval
	b.MaxElapsedTime = maxElapsedTime
	return b
}

// Wait waits for a deployment to be deployed on node
func (d *Deployer) Wait(
	ctx context.Context,
	nodeClient *client.NodeClient,
	deploymentID uint64,
	workloadVersions map[string]uint32,
) error {

	ctx, span := d.tracer.Start(ctx, "Deployer.Wait",
		trace.WithAttributes(
			attribute.Int("deployment_id", int(deploymentID)),
			attribute.Int("workload_count", len(workloadVersions)),
		))
	defer span.End()

	lastProgress := Progress{time.Now(), 0}
	numberOfWorkloads := len(workloadVersions)

	span.AddEvent("starting wait with backoff")

	deploymentError := backoff.Retry(func() error {
		stateOk := 0

		span.AddEvent("checking deployment changes")
		deploymentChanges, err := nodeClient.DeploymentChanges(ctx, deploymentID)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to fetch deployment changes")
			return backoff.Permanent(err)
		}

		for _, wl := range deploymentChanges {
			if _, ok := workloadVersions[wl.Name]; ok && wl.Version == workloadVersions[wl.Name] {
				var errString string
				switch wl.Result.State {
				case zos.StateOk:
					stateOk++
				case zos.StateError:
					errString = fmt.Sprintf("workload %s within deployment %d failed with error: %s", wl.Name, deploymentID, wl.Result.Error)
				case zos.StateDeleted:
					errString = fmt.Sprintf("workload %s state within deployment %d is deleted: %s", wl.Name, deploymentID, wl.Result.Error)
				case zos.StatePaused:
					errString = fmt.Sprintf("workload %s state within deployment %d is paused: %s", wl.Name, deploymentID, wl.Result.Error)
				case zos.StateUnChanged:
					errString = fmt.Sprintf("workload %s within deployment %d was not updated: %s", wl.Name, deploymentID, wl.Result.Error)
				}
				if errString != "" {
					span.AddEvent("workload failed",
						trace.WithAttributes(
							attribute.String("workload_name", wl.Name),
							attribute.String("error", errString),
						))
					err = errors.New(errString)

					span.RecordError(err)
					span.SetStatus(codes.Error, "workload failed")
					return backoff.Permanent(err)
				}
			}
		}

		if stateOk == numberOfWorkloads {
			span.AddEvent("all_workloads_completed")
			return nil
		}

		currentProgress := Progress{time.Now(), stateOk}
		if lastProgress.stateOk < currentProgress.stateOk {
			lastProgress = currentProgress
		} else if currentProgress.time.Sub(lastProgress.time) > 4*time.Minute {
			timeoutError := errors.Errorf("waiting for deployment %d timed out", deploymentID)
			span.AddEvent("wait_timeout",
				trace.WithAttributes(
					attribute.String("timeout_duration", "4 minutes"),
					attribute.Int("current_progress", stateOk),
				))
			return backoff.Permanent(timeoutError)
		}

		return errors.New("deployment in progress")
	},
		backoff.WithContext(getExponentialBackoff(3*time.Second, 1.25, 40*time.Second, 50*time.Minute), ctx))

	if deploymentError != nil {
		span.RecordError(deploymentError)
		span.SetStatus(codes.Error, "waiting for deployment failed")
	}

	return deploymentError
}

// BatchDeploy deploys a batch of deployments, successful deployments should have ContractID fields set
func (d *Deployer) BatchDeploy(
	ctx context.Context,
	deployments map[uint32][]zos.Deployment,
	deploymentsSolutionProvider map[uint32][]*uint64,
) (map[uint32][]zos.Deployment, error) {

	ctx, span := d.tracer.Start(ctx, "Deployer.BatchDeploy",
		trace.WithAttributes(
			attribute.Int("node_count", len(deployments)),
		))
	defer span.End()

	deploymentsSlice := make([]zos.Deployment, 0)
	contractsData := make([]substrate.BatchCreateContractData, 0)

	mu := sync.Mutex{}

	group, ctx2 := errgroup.WithContext(ctx)
	for node, dls := range deployments {
		// loading node clients first before creating any contract and caching the clients
		_, err := d.ncPool.GetNodeClient(d.substrateConn, node)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to get node client")
			return map[uint32][]zos.Deployment{}, errors.Wrap(err, "failed to get node client")
		}

		for i, dl := range dls {
			i := i
			dl := dl
			node := node

			group.Go(func() error {
				select {
				case <-ctx2.Done():
					return nil
				default:
				}

				_, workloadSpan := d.tracer.Start(ctx, "Deployer.prepare_deployment",
					trace.WithAttributes(
						attribute.Int("node", int(node)),
						attribute.Int("twin_id", int(dl.TwinID)),
					))

				if err := dl.Sign(d.twinID, d.identity); err != nil {
					spanErrorAndEnd(workloadSpan, "failed to sign deployment", err)
					return errors.Wrap(err, "error signing deployment")
				}

				if err := dl.Valid(); err != nil {
					spanErrorAndEnd(workloadSpan, "invalid deployment", err)
					return errors.Wrap(err, "deployment is invalid")
				}

				hash, err := dl.ChallengeHash()
				log.Debug().Bytes("HASH", hash)
				if err != nil {
					spanErrorAndEnd(workloadSpan, "failed to create hash", err)
					return errors.Wrap(err, "failed to create hash")
				}

				hashHex := hex.EncodeToString(hash)

				publicIPCount, err := CountDeploymentPublicIPs(dl)
				if err != nil {
					spanErrorAndEnd(workloadSpan, "failed to count public IPs", err)
					return errors.Wrap(err, "failed to count deployment public IPs")
				}
				log.Debug().Uint32("Number of public ips", publicIPCount)

				var solutionProviderID *uint64
				if deploymentsSolutionProvider[node] != nil && len(deploymentsSolutionProvider[node]) > i {
					solutionProviderID = deploymentsSolutionProvider[node][i]
				}

				workloadSpan.SetAttributes(
					attribute.String("hash", hashHex),
					attribute.Int("public_ip_count", int(publicIPCount)),
				)
				workloadSpan.End()

				mu.Lock()
				contractsData = append(contractsData, substrate.BatchCreateContractData{
					Node:               node,
					Body:               dl.Metadata,
					Hash:               hashHex,
					PublicIPs:          publicIPCount,
					SolutionProviderID: solutionProviderID,
				})
				deploymentsSlice = append(deploymentsSlice, dl)
				mu.Unlock()
				return nil
			})
		}
	}

	span.AddEvent("waiting for preparation")
	if err := group.Wait(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "preparation failed")
		return map[uint32][]zos.Deployment{}, err
	}

	span.AddEvent("creating batch contracts")
	contracts, index, err := d.substrateConn.BatchCreateContract(d.identity, contractsData)
	if err != nil && index == nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "batch contract creation failed")
		return map[uint32][]zos.Deployment{}, errors.Wrap(err, "failed to create contracts")
	}

	var multiErr error
	if err != nil {
		multiErr = multierror.Append(multiErr, err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "batch contract creation failed")
	}

	failedContracts := make([]uint64, 0)
	var wg sync.WaitGroup

	span.AddEvent("deploying to nodes")
	for i, dl := range deploymentsSlice {
		if index != nil && *index == i {
			span.AddEvent("stopping at failed index",
				trace.WithAttributes(attribute.Int("contract_id", int(dl.ContractID))),
				trace.WithAttributes(attribute.Int("twin_id", int(dl.TwinID))),
			)
			break
		}
		node := contractsData[i].Node
		dl := dl
		i := i

		wg.Add(1)
		go func() {
			defer wg.Done()

			deployCtx, deploySpan := d.tracer.Start(ctx, "Deployer.deploy_single_from_batch",
				trace.WithAttributes(
					attribute.Int("node", int(node)),
				))

			client, err := d.ncPool.GetNodeClient(d.substrateConn, node)
			if err != nil {
				mu.Lock()
				multiErr = multierror.Append(multiErr, errors.Wrapf(err, "failed to get node %d client", node))
				failedContracts = append(failedContracts, dl.ContractID)
				mu.Unlock()

				spanErrorAndEnd(deploySpan, "failed to get node client", err)
				return
			}

			dl.ContractID = contracts[i]

			deploySpan.SetAttributes(attribute.Int("assigned_contract_id", int(dl.ContractID)))

			deploySpan.AddEvent("sending_deployment_to_node")
			err = client.DeploymentDeploy(deployCtx, dl)
			if err != nil {
				mu.Lock()
				multiErr = multierror.Append(multiErr, errors.Wrapf(err, "error sending deployment with contract id %d to node %d", dl.ContractID, node))
				failedContracts = append(failedContracts, dl.ContractID)
				mu.Unlock()

				spanErrorAndEnd(deploySpan, "failed to deploy to node", err)
				return
			}

			newWorkloadVersions := make(map[string]uint32)
			for _, w := range dl.Workloads {
				newWorkloadVersions[w.Name] = 0
			}

			deploySpan.AddEvent("waiting_for_deployment")
			err = d.Wait(deployCtx, client, dl.ContractID, newWorkloadVersions)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				multiErr = multierror.Append(multiErr, errors.Wrapf(err, "error waiting deployment on node %d", node))
				failedContracts = append(failedContracts, dl.ContractID)

				spanErrorAndEnd(deploySpan, "waiting for deployment failed", err)
				return
			}

			deploymentsSlice[i].ContractID = contracts[i]

			deploySpan.End()

		}()
	}

	span.AddEvent("waiting for all deployments")
	wg.Wait()

	resDeployments := make(map[uint32][]zos.Deployment, len(deployments))
	for i, dl := range deploymentsSlice {
		resDeployments[contractsData[i].Node] = append(resDeployments[contractsData[i].Node], dl)
	}

	if len(failedContracts) != 0 {
		span.AddEvent("canceling failed contracts",
			trace.WithAttributes(
				attribute.Int("failed_contracts_count", len(failedContracts)),
			))
		err := d.substrateConn.BatchCancelContract(d.identity, failedContracts)
		if err != nil {
			multiErr = multierror.Append(multiErr, errors.Wrapf(err, "failed to cancel failed contracts %v", failedContracts))
			span.RecordError(err)
		}
	}

	if multiErr != nil {
		span.SetStatus(codes.Error, "batch deploy completed with errors")
		span.SetAttributes(
			attribute.Int("failed_contracts", len(failedContracts)),
			attribute.Int("successful_deployments", len(deploymentsSlice)-len(failedContracts)),
		)
	} else {
		span.SetAttributes(
			attribute.Int("total_deployments", len(deploymentsSlice)),
		)
	}

	return resDeployments, multiErr
}

// matchOldVersions assigns deployment and workloads versions of the new versionless deployment to the ones of the old deployment
func matchOldVersions(oldDl *zos.Deployment, newDl *zos.Deployment) {
	oldWlVersions := map[string]uint32{}
	for _, wl := range oldDl.Workloads {
		oldWlVersions[wl.Name] = wl.Version
	}

	newDl.Version = oldDl.Version

	for idx, wl := range newDl.Workloads {
		newDl.Workloads[idx].Version = oldWlVersions[wl.Name]
	}
}

// assignVersions determines and assigns the versions of the new deployment and its workloads
func assignVersions(oldDl *zos.Deployment, newDl *zos.Deployment) (map[string]uint32, error) {
	oldHashes, err := GetWorkloadHashes(*oldDl)
	if err != nil {
		return nil, errors.Wrap(err, "could not get old workloads hashes")
	}

	newHashes, err := GetWorkloadHashes(*newDl)
	if err != nil {
		return nil, errors.Wrap(err, "could not get new workloads hashes")
	}

	newWorkloadsVersions := make(map[string]uint32)
	newDl.Version = oldDl.Version + 1

	for idx, w := range newDl.Workloads {
		newHash := newHashes[w.Name]
		oldHash, ok := oldHashes[w.Name]
		if !ok || newHash != oldHash {
			newDl.Workloads[idx].Version = newDl.Version
		}
		newWorkloadsVersions[w.Name] = newDl.Workloads[idx].Version
	}

	return newWorkloadsVersions, nil
}

// Validate is a best effort validation. it returns an error if it's very sure there's a problem
//   - validates old deployments nodes (for update cases) and new deployments nodes
//   - validates nodes' farm
//   - checks free public ips
//   - checks free nodes capacity
//   - checks PublicConfig Ipv4 for fqdn gateway
//   - checks PublicConfig domain for name gateway
//
// errors that may arise because of dead nodes are ignored.
// if a real error dodges the validation, it'll be fail anyway in the deploying phase
func (d *Deployer) Validate(ctx context.Context, oldDeployments map[uint32]zos.Deployment, newDeployments map[uint32]zos.Deployment) error {

	ctx, span := d.tracer.Start(ctx, "Deployer.Validate",
		trace.WithAttributes(
			attribute.Int("old_deployments_count", len(oldDeployments)),
			attribute.Int("new_deployments_count", len(newDeployments)),
		))
	defer span.End()

	farmIPs := make(map[int]int)
	nodeMap := make(map[uint32]proxyTypes.NodeWithNestedCapacity)

	span.AddEvent("fetching node information")
	for node := range oldDeployments {
		nodeInfo, err := d.gridProxyClient.Node(ctx, node)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to get node data from grid proxy")
			return errors.Wrapf(err, "could not get node %d data from the grid proxy", node)
		}
		nodeMap[node] = nodeInfo
		farmIPs[nodeInfo.FarmID] = 0
	}

	for node := range newDeployments {
		if _, ok := nodeMap[node]; ok {
			continue
		}
		nodeInfo, err := d.gridProxyClient.Node(ctx, node)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to get node data from proxy")
			return errors.Wrapf(err, "could not get node %d data from the grid proxy", node)
		}
		nodeMap[node] = nodeInfo
		farmIPs[nodeInfo.FarmID] = 0
	}

	span.AddEvent("fetching farm information")
	for farm := range farmIPs {
		farmUint64 := uint64(farm)
		farmInfo, _, err := d.gridProxyClient.Farms(ctx, proxyTypes.FarmFilter{
			FarmID: &farmUint64,
		}, proxyTypes.Limit{
			Page: 1,
			Size: 1,
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to get farm data from proxy")
			return errors.Wrapf(err, "could not get farm %d data from the grid proxy", farm)
		}
		if len(farmInfo) == 0 {
			err := errors.Errorf("farm %d not returned from the proxy", farm)
			span.RecordError(err)
			span.SetStatus(codes.Error, "farm not found in proxy")
			return err
		}
		for _, ip := range farmInfo[0].PublicIps {
			if ip.ContractID == 0 {
				farmIPs[farm]++
			}
		}
	}

	span.AddEvent("validating old deployments")
	for node, dl := range oldDeployments {
		nodeData, ok := nodeMap[node]
		if !ok {
			err := errors.Errorf("node %d not returned from the grid proxy", node)
			span.RecordError(err)
			span.SetStatus(codes.Error, "node not found in proxy")
			return err
		}

		publicIPCount, err := CountDeploymentPublicIPs(dl)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to count public IPs")
			return errors.Wrap(err, "failed to count deployment public IPs")
		}

		farmIPs[nodeData.FarmID] += int(publicIPCount)
	}

	span.AddEvent("validating new deployments")
	for node, dl := range newDeployments {

		_, nodeSpan := d.tracer.Start(ctx, "Deployer.validate_node_deployment",
			trace.WithAttributes(
				attribute.Int("node", int(node)),
			))

		if err := dl.Valid(); err != nil {
			spanErrorAndEnd(nodeSpan, "invalid deployment", err)
			span.RecordError(err)
			return errors.Wrap(err, "invalid deployment")
		}

		oldDl, alreadyExists := oldDeployments[node]

		needed, err := Capacity(dl)
		if err != nil {
			spanErrorAndEnd(nodeSpan, "failed to calculate capacity", err)
			span.RecordError(err)
			return err
		}

		publicIPCount, err := CountDeploymentPublicIPs(dl)
		if err != nil {
			spanErrorAndEnd(nodeSpan, "failed to count public IPs", err)
			span.RecordError(err)
			return errors.Wrap(err, "failed to count deployment public IPs")
		}
		requiredIPs := int(publicIPCount)
		nodeInfo := nodeMap[node]

		nodeSpan.SetAttributes(
			attribute.Int("required_public_ips", requiredIPs),
			attribute.Int("farm_id", nodeInfo.FarmID),
		)

		if alreadyExists {
			oldCap, err := Capacity(oldDl)
			if err != nil {
				spanErrorAndEnd(nodeSpan, "failed to calculate old capacity", err)
				span.RecordError(err)
				return errors.Wrapf(err, "could not read old deployment %d of node %d capacity", oldDl.ContractID, node)
			}

			addCapacity(&nodeInfo.Capacity.Total, &oldCap)
			contract, err := d.substrateConn.GetContract(oldDl.ContractID)
			if err != nil {
				spanErrorAndEnd(nodeSpan, "failed to get contract", err)
				span.RecordError(err)
				return errors.Wrapf(err, "could not get node contract %d", oldDl.ContractID)
			}
			current := int(contract.PublicIPCount())
			if requiredIPs > current {
				err := errors.Errorf(
					"currently, it's not possible to increase the number of reserved public ips in a deployment, node: %d, current: %d, requested: %d",
					node,
					current,
					requiredIPs,
				)
				spanErrorAndEnd(nodeSpan, "cannot increase public IPs", err)
				span.RecordError(err)
				return err
			}
		}

		farmIPs[nodeInfo.FarmID] -= requiredIPs
		if farmIPs[nodeInfo.FarmID] < 0 {
			err := errors.Errorf("farm %d does not have enough public ips", nodeInfo.FarmID)
			spanErrorAndEnd(nodeSpan, "insufficient public IPs in farm", err)
			span.RecordError(err)
			return err
		}

		if HasWorkload(&dl, zos.GatewayFQDNProxyType) && nodeInfo.PublicConfig.Ipv4 == "" {
			err := errors.Errorf("node %d cannot deploy a fqdn workload as it does not have a public ipv4 configured", node)
			spanErrorAndEnd(nodeSpan, "IPv4 is missing from fqdn workload", err)
			span.RecordError(err)
			return err
		}

		if HasWorkload(&dl, zos.GatewayNameProxyType) && nodeInfo.PublicConfig.Domain == "" {
			err := errors.Errorf("node %d cannot deploy a gateway name workload as it does not have a domain configured", node)
			spanErrorAndEnd(nodeSpan, "domain is missing for gateway name workload", err)
			span.RecordError(err)
			return err
		}

		mru := nodeInfo.Capacity.Total.MRU - nodeInfo.Capacity.Used.MRU
		hru := nodeInfo.Capacity.Total.HRU - nodeInfo.Capacity.Used.HRU
		sru := 2*nodeInfo.Capacity.Total.SRU - nodeInfo.Capacity.Used.SRU

		nodeSpan.SetAttributes(
			attribute.Int64("available_mru", int64(mru)),
			attribute.Int64("available_sru", int64(sru)),
			attribute.Int64("available_hru", int64(hru)),
			attribute.Int64("needed_mru", int64(needed.MRU)),
			attribute.Int64("needed_sru", int64(needed.SRU)),
			attribute.Int64("needed_hru", int64(needed.HRU)),
		)

		if uint64(mru) < needed.MRU ||
			uint64(sru) < needed.SRU ||
			uint64(hru) < needed.HRU {
			free := zos.Capacity{
				HRU: uint64(hru),
				MRU: uint64(mru),
				SRU: uint64(sru),
			}
			err := errors.Errorf("node %d does not have enough resources. needed: %v, free: %v", node, capacityPrettyPrint(needed), capacityPrettyPrint(free))
			spanErrorAndEnd(nodeSpan, "insufficient resources", err)
			span.RecordError(err)
			return err
		}
		nodeSpan.End()
	}

	return nil
}

// capacityPrettyPrint prints the capacity data
func capacityPrettyPrint(cap zos.Capacity) string {
	return fmt.Sprintf("[mru: %d, sru: %d, hru: %d]", cap.MRU, cap.SRU, cap.HRU)
}

// addCapacity adds a new data for capacity
func addCapacity(cap *proxyTypes.Capacity, add *zos.Capacity) {
	cap.CRU += add.CRU
	cap.MRU += gridtypes.Unit(add.MRU)
	cap.SRU += gridtypes.Unit(add.SRU)
	cap.HRU += gridtypes.Unit(add.HRU)
}
