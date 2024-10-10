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
	"github.com/threefoldtech/zos/pkg/gridtypes"
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
}

// NewDeployer returns a new deployer
func NewDeployer(
	tfPluginClient TFPluginClient,
	revertOnFailure bool,
) Deployer {
	return Deployer{
		tfPluginClient.Identity,
		tfPluginClient.TwinID,
		tfPluginClient.GridProxyClient,
		tfPluginClient.NcPool,
		revertOnFailure,
		tfPluginClient.SubstrateConn,
	}
}

// Deploy deploys or updates a new deployment given the old deployments' IDs
func (d *Deployer) Deploy(ctx context.Context,
	oldDeploymentIDs map[uint32]uint64,
	newDeployments map[uint32]zos.Deployment,
	newDeploymentSolutionProvider map[uint32]*uint64,
) (map[uint32]uint64, error) {
	oldDeployments, oldErr := d.GetDeployments(ctx, oldDeploymentIDs)
	if oldErr == nil {
		// check resources only when old deployments are readable
		// being readable means it's a fresh deployment or an update with good nodes
		// this is done to avoid preventing deletion of deployments on dead nodes
		if err := d.Validate(ctx, oldDeployments, newDeployments); err != nil {
			return oldDeploymentIDs, err
		}
	}

	// ignore oldErr until we need oldDeployments
	currentDeployments, err := d.deploy(ctx, oldDeploymentIDs, newDeployments, newDeploymentSolutionProvider, d.revertOnFailure)

	if err != nil && d.revertOnFailure {
		if oldErr != nil {
			return currentDeployments, errors.Wrapf(err, "failed to fetch deployment objects to revert deployments: %s; try again", oldErr)
		}

		currentDls, rerr := d.deploy(ctx, currentDeployments, oldDeployments, newDeploymentSolutionProvider, false)
		if rerr != nil {
			return currentDls, errors.Wrapf(err, "failed to revert deployments: %s; try again", rerr)
		}
		return currentDls, err
	}

	return currentDeployments, err
}

func (d *Deployer) deploy(
	ctx context.Context,
	oldDeployments map[uint32]uint64,
	newDeployments map[uint32]zos.Deployment,
	newDeploymentSolutionProvider map[uint32]*uint64,
	revertOnFailure bool,
) (currentDeployments map[uint32]uint64, err error) {
	currentDeployments = make(map[uint32]uint64)
	for nodeID, contractID := range oldDeployments {
		currentDeployments[nodeID] = contractID
	}
	// deletions
	for node, contractID := range oldDeployments {
		if _, ok := newDeployments[node]; !ok {
			err = d.substrateConn.EnsureContractCanceled(d.identity, contractID)
			if err != nil && !strings.Contains(err.Error(), "ContractNotExists") {
				return currentDeployments, errors.Wrap(err, "failed to delete deployment")
			}
			delete(currentDeployments, node)
		}
	}

	// creations
	for node, dl := range newDeployments {
		if _, ok := oldDeployments[node]; !ok {
			client, err := d.ncPool.GetNodeClient(d.substrateConn, node)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to get node client")
			}

			if err := dl.Sign(d.twinID, d.identity); err != nil {
				return currentDeployments, errors.Wrap(err, "error signing deployment")
			}

			if err := dl.Valid(); err != nil {
				return currentDeployments, errors.Wrap(err, "deployment is invalid")
			}

			hash, err := dl.ChallengeHash()
			log.Debug().Bytes("HASH", hash)

			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to create hash")
			}

			hashHex := hex.EncodeToString(hash)

			publicIPCount, err := CountDeploymentPublicIPs(dl)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to count deployment public IPs")
			}
			log.Debug().Uint32("Number of public ips", publicIPCount)

			contractID, err := d.substrateConn.CreateNodeContract(d.identity, node, dl.Metadata, hashHex, publicIPCount, newDeploymentSolutionProvider[node])
			log.Debug().Uint64("CreateNodeContract returned id", contractID)
			if err != nil {
				return currentDeployments, errors.Wrapf(err, "failed to create contract on node %d", node)
			}

			dl.ContractID = contractID
			err = client.DeploymentDeploy(ctx, dl)
			if err != nil {
				rerr := d.substrateConn.EnsureContractCanceled(d.identity, contractID)
				if rerr != nil {
					return currentDeployments, errors.Wrapf(err, "error cancelling contract: %s; you must cancel it manually (id: %d)", rerr, contractID)
				}
				return currentDeployments, errors.Wrapf(err, "error sending deployment to node %d", node)

			}
			currentDeployments[node] = dl.ContractID
			newWorkloadVersions := make(map[string]uint32)
			for _, w := range dl.Workloads {
				newWorkloadVersions[w.Name] = 0
			}
			err = d.Wait(ctx, client, dl.ContractID, newWorkloadVersions)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "error waiting deployment")
			}
		}
	}

	// updates
	for node, dl := range newDeployments {
		if oldDeploymentID, ok := oldDeployments[node]; ok {
			client, err := d.ncPool.GetNodeClient(d.substrateConn, node)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to get node client")
			}

			oldDl, err := client.DeploymentGet(ctx, oldDeploymentID)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to get old deployment to update it")
			}

			matchOldVersions(&oldDl, &dl)

			oldDeploymentHash, err := HashDeployment(oldDl)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "could not get deployment hash")
			}

			newDeploymentHash, err := HashDeployment(dl)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "could not get deployment hash")
			}

			if oldDeploymentHash == newDeploymentHash && SameWorkloadsNames(dl, oldDl) {
				continue
			}

			newWorkloadsVersions, err := assignVersions(&oldDl, &dl)
			if err != nil {
				return currentDeployments, errors.Wrapf(err, "failed to assign new versions to deployment with contract %d", oldDeploymentID)
			}

			dl.ContractID = oldDl.ContractID

			if err := dl.Sign(d.twinID, d.identity); err != nil {
				return currentDeployments, errors.Wrap(err, "error signing deployment")
			}

			if err := dl.Valid(); err != nil {
				return currentDeployments, errors.Wrap(err, "deployment is invalid")
			}

			log.Debug().Interface("deployment", dl)
			hash, err := dl.ChallengeHash()
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to create hash")
			}
			hashHex := hex.EncodeToString(hash)
			log.Debug().Str("HASH", hashHex)

			// TODO: Destroy and create if publicIPCount is changed
			// publicIPCount, err := countDeploymentPublicIPs(dl)
			contractID, err := d.substrateConn.UpdateNodeContract(d.identity, dl.ContractID, dl.Metadata, hashHex)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "failed to update deployment")
			}
			dl.ContractID = contractID
			err = client.DeploymentUpdate(ctx, dl)
			if err != nil {
				// cancel previous contract
				return currentDeployments, errors.Wrapf(err, "failed to send deployment update request to node %d", node)
			}
			currentDeployments[node] = dl.ContractID

			err = d.Wait(ctx, client, dl.ContractID, newWorkloadsVersions)
			if err != nil {
				return currentDeployments, errors.Wrap(err, "error waiting deployment")
			}
		}
	}

	return currentDeployments, nil
}

// Cancel cancels an old deployment not given in the new deployments
func (d *Deployer) Cancel(ctx context.Context,
	contractID uint64,
) error {
	err := d.substrateConn.EnsureContractCanceled(d.identity, contractID)
	if err != nil {
		return errors.Wrapf(err, "failed to delete deployment: %d", contractID)
	}

	return nil
}

// GetDeployments returns deployments from a map of nodes IDs and deployments IDs
func (d *Deployer) GetDeployments(ctx context.Context, dls map[uint32]uint64) (map[uint32]zos.Deployment, error) {
	res := make(map[uint32]zos.Deployment)

	for nodeID, dlID := range dls {
		nc, err := d.ncPool.GetNodeClient(d.substrateConn, nodeID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get a client for node %d", nodeID)
		}

		dl, err := nc.DeploymentGet(ctx, dlID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get deployment %d of node %d", dlID, nodeID)
		}

		res[nodeID] = dl
	}

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
	lastProgress := Progress{time.Now(), 0}
	numberOfWorkloads := len(workloadVersions)

	deploymentError := backoff.Retry(func() error {
		stateOk := 0

		deploymentChanges, err := nodeClient.DeploymentChanges(ctx, deploymentID)
		if err != nil {
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
					return backoff.Permanent(errors.New(errString))
				}
			}
		}

		if stateOk == numberOfWorkloads {
			return nil
		}

		currentProgress := Progress{time.Now(), stateOk}
		if lastProgress.stateOk < currentProgress.stateOk {
			lastProgress = currentProgress
		} else if currentProgress.time.Sub(lastProgress.time) > 4*time.Minute {
			timeoutError := errors.Errorf("waiting for deployment %d timed out", deploymentID)
			return backoff.Permanent(timeoutError)
		}

		return errors.New("deployment in progress")
	},
		backoff.WithContext(getExponentialBackoff(3*time.Second, 1.25, 40*time.Second, 50*time.Minute), ctx))

	return deploymentError
}

// BatchDeploy deploys a batch of deployments, successful deployments should have ContractID fields set
func (d *Deployer) BatchDeploy(
	ctx context.Context,
	deployments map[uint32][]zos.Deployment,
	deploymentsSolutionProvider map[uint32][]*uint64,
) (map[uint32][]zos.Deployment, error) {
	deploymentsSlice := make([]zos.Deployment, 0)
	contractsData := make([]substrate.BatchCreateContractData, 0)

	mu := sync.Mutex{}

	group, ctx2 := errgroup.WithContext(ctx)
	for node, dls := range deployments {
		// loading node clients first before creating any contract and caching the clients
		_, err := d.ncPool.GetNodeClient(d.substrateConn, node)
		if err != nil {
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

				if err := dl.Sign(d.twinID, d.identity); err != nil {
					return errors.Wrap(err, "error signing deployment")
				}

				if err := dl.Valid(); err != nil {
					return errors.Wrap(err, "deployment is invalid")
				}

				hash, err := dl.ChallengeHash()
				log.Debug().Bytes("HASH", hash)

				if err != nil {
					return errors.Wrap(err, "failed to create hash")
				}

				hashHex := hex.EncodeToString(hash)

				publicIPCount, err := CountDeploymentPublicIPs(dl)
				if err != nil {
					return errors.Wrap(err, "failed to count deployment public IPs")
				}
				log.Debug().Uint32("Number of public ips", publicIPCount)

				var solutionProviderID *uint64
				if deploymentsSolutionProvider[node] != nil && len(deploymentsSolutionProvider[node]) > i {
					solutionProviderID = deploymentsSolutionProvider[node][i]
				}
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

	if err := group.Wait(); err != nil {
		return map[uint32][]zos.Deployment{}, err
	}

	contracts, index, err := d.substrateConn.BatchCreateContract(d.identity, contractsData)
	if err != nil && index == nil {
		return map[uint32][]zos.Deployment{}, errors.Wrap(err, "failed to create contracts")
	}

	var multiErr error
	if err != nil {
		multiErr = multierror.Append(multiErr, err)
	}

	failedContracts := make([]uint64, 0)
	var wg sync.WaitGroup
	for i, dl := range deploymentsSlice {
		if index != nil && *index == i {
			break
		}
		node := contractsData[i].Node
		dl := dl
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			client, err := d.ncPool.GetNodeClient(d.substrateConn, node)
			if err != nil {
				mu.Lock()
				multiErr = multierror.Append(multiErr, errors.Wrapf(err, "failed to get node %d client", node))
				failedContracts = append(failedContracts, dl.ContractID)
				mu.Unlock()
				return
			}
			dl.ContractID = contracts[i]

			err = client.DeploymentDeploy(ctx, dl)
			if err != nil {
				mu.Lock()
				multiErr = multierror.Append(multiErr, errors.Wrapf(err, "error sending deployment with contract id %d to node %d", dl.ContractID, node))
				failedContracts = append(failedContracts, dl.ContractID)
				mu.Unlock()
				return
			}
			newWorkloadVersions := make(map[string]uint32)
			for _, w := range dl.Workloads {
				newWorkloadVersions[w.Name] = 0
			}
			err = d.Wait(ctx, client, dl.ContractID, newWorkloadVersions)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				multiErr = multierror.Append(multiErr, errors.Wrapf(err, "error waiting deployment on node %d", node))
				failedContracts = append(failedContracts, dl.ContractID)
				return
			}
			deploymentsSlice[i].ContractID = contracts[i]
		}()
	}
	wg.Wait()

	resDeployments := make(map[uint32][]zos.Deployment, len(deployments))
	for i, dl := range deploymentsSlice {
		resDeployments[contractsData[i].Node] = append(resDeployments[contractsData[i].Node], dl)
	}

	if len(failedContracts) != 0 {
		err := d.substrateConn.BatchCancelContract(d.identity, failedContracts)
		if err != nil {
			multiErr = multierror.Append(multiErr, errors.Wrapf(err, "failed to cancel failed contracts %v", failedContracts))
		}
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
	farmIPs := make(map[int]int)
	nodeMap := make(map[uint32]proxyTypes.NodeWithNestedCapacity)

	for node := range oldDeployments {
		nodeInfo, err := d.gridProxyClient.Node(ctx, node)
		if err != nil {
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
			return errors.Wrapf(err, "could not get node %d data from the grid proxy", node)
		}
		nodeMap[node] = nodeInfo
		farmIPs[nodeInfo.FarmID] = 0
	}

	for farm := range farmIPs {
		farmUint64 := uint64(farm)
		farmInfo, _, err := d.gridProxyClient.Farms(ctx, proxyTypes.FarmFilter{
			FarmID: &farmUint64,
		}, proxyTypes.Limit{
			Page: 1,
			Size: 1,
		})
		if err != nil {
			return errors.Wrapf(err, "could not get farm %d data from the grid proxy", farm)
		}
		if len(farmInfo) == 0 {
			return errors.Errorf("farm %d not returned from the proxy", farm)
		}
		for _, ip := range farmInfo[0].PublicIps {
			if ip.ContractID == 0 {
				farmIPs[farm]++
			}
		}
	}

	for node, dl := range oldDeployments {
		nodeData, ok := nodeMap[node]
		if !ok {
			return errors.Errorf("node %d not returned from the grid proxy", node)
		}

		publicIPCount, err := CountDeploymentPublicIPs(dl)
		if err != nil {
			return errors.Wrap(err, "failed to count deployment public IPs")
		}

		farmIPs[nodeData.FarmID] += int(publicIPCount)
	}

	for node, dl := range newDeployments {
		if err := dl.Valid(); err != nil {
			return errors.Wrap(err, "invalid deployment")
		}

		oldDl, alreadyExists := oldDeployments[node]

		needed, err := Capacity(dl)
		if err != nil {
			return err
		}

		publicIPCount, err := CountDeploymentPublicIPs(dl)
		if err != nil {
			return errors.Wrap(err, "failed to count deployment public IPs")
		}
		requiredIPs := int(publicIPCount)
		nodeInfo := nodeMap[node]
		if alreadyExists {
			oldCap, err := Capacity(oldDl)
			if err != nil {
				return errors.Wrapf(err, "could not read old deployment %d of node %d capacity", oldDl.ContractID, node)
			}
			addCapacity(&nodeInfo.Capacity.Total, &oldCap)
			contract, err := d.substrateConn.GetContract(oldDl.ContractID)
			if err != nil {
				return errors.Wrapf(err, "could not get node contract %d", oldDl.ContractID)
			}
			current := int(contract.PublicIPCount())
			if requiredIPs > current {
				return errors.Errorf(
					"currently, it's not possible to increase the number of reserved public ips in a deployment, node: %d, current: %d, requested: %d",
					node,
					current,
					requiredIPs,
				)
			}
		}

		farmIPs[nodeInfo.FarmID] -= requiredIPs
		if farmIPs[nodeInfo.FarmID] < 0 {
			return errors.Errorf("farm %d does not have enough public ips", nodeInfo.FarmID)
		}
		if HasWorkload(&dl, zos.GatewayFQDNProxyType) && nodeInfo.PublicConfig.Ipv4 == "" {
			return errors.Errorf("node %d cannot deploy a fqdn workload as it does not have a public ipv4 configured", node)
		}
		if HasWorkload(&dl, zos.GatewayNameProxyType) && nodeInfo.PublicConfig.Domain == "" {
			return errors.Errorf("node %d cannot deploy a gateway name workload as it does not have a domain configured", node)
		}
		mru := nodeInfo.Capacity.Total.MRU - nodeInfo.Capacity.Used.MRU
		hru := nodeInfo.Capacity.Total.HRU - nodeInfo.Capacity.Used.HRU
		sru := 2*nodeInfo.Capacity.Total.SRU - nodeInfo.Capacity.Used.SRU
		if uint64(mru) < needed.MRU ||
			uint64(sru) < needed.SRU ||
			uint64(hru) < needed.HRU {
			free := zos.Capacity{
				HRU: uint64(hru),
				MRU: uint64(mru),
				SRU: uint64(sru),
			}
			return errors.Errorf("node %d does not have enough resources. needed: %v, free: %v", node, capacityPrettyPrint(needed), capacityPrettyPrint(free))
		}
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
