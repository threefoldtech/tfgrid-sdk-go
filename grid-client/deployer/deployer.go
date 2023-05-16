// Package deployer for grid deployer
package deployer

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
	proxy "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
	proxyTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"golang.org/x/sync/errgroup"
)

var DeploymentHasNoUpdatesErr = errors.New("deployment has no updates")

// MockDeployer to be used for any deployer in mock testing
type MockDeployer interface { //TODO: Change Name && separate them
	Deploy(ctx context.Context,
		oldDeploymentIDs map[uint32]uint64,
		newDeployments map[uint32]gridtypes.Deployment,
		newDeploymentSolutionProvider map[uint32]*uint64,
	) (map[uint32]uint64, error)

	Cancel(ctx context.Context,
		contractID uint64,
	) error

	GetDeployments(ctx context.Context, dls map[uint32]uint64) (map[uint32]gridtypes.Deployment, error)
	BatchDeploy(ctx context.Context,
		deployments map[uint32][]gridtypes.Deployment,
		deploymentsSolutionProvider map[uint32][]*uint64,
	) (map[uint32][]gridtypes.Deployment, error)
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
	newDeployments map[uint32]gridtypes.Deployment,
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
	currentDeployments, err := d.deploy(ctx, oldDeploymentIDs, newDeployments, newDeploymentSolutionProvider)

	if err != nil && d.revertOnFailure {
		if oldErr != nil {
			return currentDeployments, errors.Wrapf(err, "failed to fetch deployment objects to revert deployments: %s; try again", oldErr)
		}

		currentDls, rerr := d.deploy(ctx, currentDeployments, oldDeployments, newDeploymentSolutionProvider)
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
	newDeployments map[uint32]gridtypes.Deployment,
	newDeploymentSolutionProvider map[uint32]*uint64,
) (currentDeployments map[uint32]uint64, err error) {
	currentDeployments = make(map[uint32]uint64)
	for nodeID, contractID := range oldDeployments {
		currentDeployments[nodeID] = contractID
	}

	if err := d.deleteHandler(ctx, oldDeployments, newDeployments, currentDeployments); err != nil {
		return currentDeployments, errors.Wrap(err, "failed to delete deployment")
	}

	if err := d.createHandler(ctx, oldDeployments, newDeployments, newDeploymentSolutionProvider, currentDeployments); err != nil {
		return currentDeployments, errors.Wrap(err, "failed to create deployments")
	}

	if err := d.updateHandler(ctx, oldDeployments, newDeployments); err != nil {
		return currentDeployments, errors.Wrap(err, "failed to update deployments")
	}

	return currentDeployments, nil
}

func (d *Deployer) deleteHandler(ctx context.Context, oldDls map[uint32]uint64, newDls map[uint32]gridtypes.Deployment, currentDeployments map[uint32]uint64) error {
	contractsToDelete := []uint64{}
	nodeIDs := []uint32{}
	for nodeID, contractID := range oldDls {
		if _, ok := newDls[nodeID]; !ok {
			contractsToDelete = append(contractsToDelete, contractID)
			nodeIDs = append(nodeIDs, nodeID)
		}
	}

	if err := d.substrateConn.BatchCancelContract(d.identity, contractsToDelete); err != nil {
		return err
	}

	for _, nodeID := range nodeIDs {
		delete(currentDeployments, nodeID)
	}

	return nil
}

func (d *Deployer) createHandler(
	ctx context.Context,
	oldDls map[uint32]uint64,
	newDls map[uint32]gridtypes.Deployment,
	solutionProviders map[uint32]*uint64,
	currentDeployments map[uint32]uint64,
) error {
	dls := map[uint32][]gridtypes.Deployment{}
	dlsSolProviders := map[uint32][]*uint64{}
	for node, dl := range newDls {
		if _, ok := oldDls[node]; !ok {
			dls[node] = []gridtypes.Deployment{dl}
			dlsSolProviders[node] = []*uint64{solutionProviders[node]}
		}
	}

	ret, err := d.BatchDeploy(ctx, dls, dlsSolProviders)

	updateCurrentDeployments(ret, currentDeployments)

	return err
}

func (d *Deployer) updateHandler(ctx context.Context, oldDls map[uint32]uint64, newDls map[uint32]gridtypes.Deployment) error {
	errGroup := errgroup.Group{}

	for nodeID, deployment := range newDls {
		if oldDeploymentContractID, ok := oldDls[nodeID]; ok {
			// make copies for variables captured by the go routine
			node := nodeID
			dl := deployment
			contractID := oldDeploymentContractID

			errGroup.Go(func() error {
				err := d.preprocessDeploymentUpdate(ctx, &dl, node, contractID)
				if errors.Is(err, DeploymentHasNoUpdatesErr) {
					return nil
				}

				if err != nil {
					return errors.Wrap(err, "failed to validate to deployment for update")
				}

				hash, err := dl.ChallengeHash()
				if err != nil {
					return errors.Wrap(err, "failed to create hash")
				}
				hashHex := hex.EncodeToString(hash)

				if _, err = d.substrateConn.UpdateNodeContract(d.identity, dl.ContractID, "", hashHex); err != nil {
					return errors.Wrap(err, "failed to update deployment")
				}

				client, err := d.ncPool.GetNodeClient(d.substrateConn, node)
				if err != nil {
					return errors.Wrap(err, "failed to get node client")
				}

				err = client.DeploymentUpdate(ctx, dl)
				if err != nil {
					return errors.Wrapf(err, "failed to send deployment update request to node %d", node)
				}

				return d.Wait(ctx, client, &dl)
			})

		}
	}

	return errGroup.Wait()
}

// preprocessDeploymentUpdate first validates that the deployment has any updates to begin with,
// then assigns versions to the deployment and its workloads,
// then signs and validates the deployment
func (d *Deployer) preprocessDeploymentUpdate(ctx context.Context, dl *gridtypes.Deployment, node uint32, contractID uint64) error {
	client, err := d.ncPool.GetNodeClient(d.substrateConn, node)
	if err != nil {
		return errors.Wrap(err, "failed to get node client")
	}

	oldDl, err := client.DeploymentGet(ctx, contractID)
	if err != nil {
		return errors.Wrap(err, "failed to get old deployment to update it")
	}

	hasUpdates, err := doesDeploymentHaveUpdates(&oldDl, dl)
	if err != nil {
		return errors.Wrap(err, "failed to determine if deployment has an update")
	}

	if !hasUpdates {
		return DeploymentHasNoUpdatesErr
	}

	if err := d.prepareDeploymentForUpdate(&oldDl, dl); err != nil {
		return err
	}

	return nil
}

func (d *Deployer) prepareDeploymentForUpdate(oldDl *gridtypes.Deployment, dl *gridtypes.Deployment) error {
	if err := udpateDeploymentVersions(oldDl, dl); err != nil {
		return errors.Wrap(err, "failed to udpate deployment versions")
	}

	if err := dl.Sign(d.twinID, d.identity); err != nil {
		return errors.Wrap(err, "error signing deployment")
	}

	if err := dl.Valid(); err != nil {
		return errors.Wrap(err, "deployment is invalid")
	}

	dl.ContractID = oldDl.ContractID

	return nil
}

func doesDeploymentHaveUpdates(oldDl *gridtypes.Deployment, newDl *gridtypes.Deployment) (bool, error) {
	// TODO: new deployment should have the same order of old deployment to be able to compare hashes
	// if two deployments have the same workloads with no updates, but only the workloads have different order on each deployment,
	// the hashes would differ

	oldDeploymentHash, err := oldDl.ChallengeHash()
	if err != nil {
		return false, errors.Wrap(err, "could not get deployment hash")
	}

	newDeploymentHash, err := newDl.ChallengeHash()
	if err != nil {
		return false, errors.Wrap(err, "could not get deployment hash")
	}

	if string(oldDeploymentHash) != string(newDeploymentHash) {
		return true, nil
	}

	// workload names are not included in deployment hashes, so a separete check should be done
	if !SameWorkloadsNames(oldDl.Workloads, newDl.Workloads) {
		return true, nil
	}

	return true, nil
}

func udpateDeploymentVersions(oldDl *gridtypes.Deployment, newDl *gridtypes.Deployment) error {
	oldHashes, err := GetWorkloadHashes(*oldDl)
	if err != nil {
		return errors.Wrap(err, "could not get old workloads hashes")
	}

	oldVersions := ConstructWorkloadVersions(oldDl)

	newDl.Version = oldDl.Version + 1

	for idx, w := range newDl.Workloads {
		newHash, err := ChallengeWorkloadHash(w)
		if err != nil {
			return err
		}

		if oldHashes[w.Name.String()] == newHash {
			newDl.Workloads[idx].Version = oldVersions[w.Name.String()]
		}

		newDl.Workloads[idx].Version = newDl.Version
	}

	return nil
}

func updateCurrentDeployments(dls map[uint32][]gridtypes.Deployment, currentDeployments map[uint32]uint64) {
	// TODO: current deloyments should handle more than one deployment per node
	for node, dls := range dls {
		for _, dl := range dls {
			if dl.ContractID != 0 {
				currentDeployments[node] = dl.ContractID
				break
			}
		}
	}
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
func (d *Deployer) GetDeployments(ctx context.Context, dls map[uint32]uint64) (map[uint32]gridtypes.Deployment, error) {
	res := make(map[uint32]gridtypes.Deployment)

	for nodeID, dlID := range dls {
		nc, err := d.ncPool.GetNodeClient(d.substrateConn, nodeID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get a client for node %d", nodeID)
		}

		sub, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		dl, err := nc.DeploymentGet(sub, dlID)
		if err != nil {

			return nil, errors.Wrapf(err, "failed to get deployment %d of node %d", dlID, nodeID)
		}
		res[nodeID] = dl
	}

	return res, nil
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
	dl *gridtypes.Deployment,
) error {
	timestamp := time.Now()
	lastStateOkCount := 0

	workloadVersions := ConstructWorkloadVersions(dl)
	deploymentError := backoff.Retry(func() error {
		stateOk := 0
		sub, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		deploymentChanges, err := nodeClient.DeploymentChanges(sub, dl.ContractID)
		if err != nil {
			return backoff.Permanent(err)
		}

		for _, wl := range deploymentChanges {
			if _, ok := workloadVersions[wl.Name.String()]; ok && wl.Version == workloadVersions[wl.Name.String()] {
				if wl.Result.State == gridtypes.StateOk {
					stateOk++
					continue
				}

				// if state is neither ok nor init, some error has ocurred
				if wl.Result.State != gridtypes.StateInit {
					return backoff.Permanent(fmt.Errorf("workload failure: workload %s state is %s: %s", wl.Name, wl.Result.State, wl.Result.Error))
				}
			}
		}

		if stateOk == len(dl.Workloads) {
			return nil
		}

		if time.Now().Sub(timestamp) > 4*time.Minute {
			return backoff.Permanent(errors.Errorf("deployment %d has timed out", dl.ContractID))
		}

		if lastStateOkCount < stateOk {
			lastStateOkCount = stateOk
			timestamp = time.Now()
		}

		return errors.New("deployment in progress")
	},
		backoff.WithContext(getExponentialBackoff(3*time.Second, 1.25, 40*time.Second, 50*time.Minute), ctx))

	return deploymentError
}

// BatchDeploy deploys a batch of deployments, successful deployments should have ContractID fields set
func (d *Deployer) BatchDeploy(ctx context.Context, deployments map[uint32][]gridtypes.Deployment, deploymentsSolutionProvider map[uint32][]*uint64) (map[uint32][]gridtypes.Deployment, error) {
	deploymentsSlice := make([]gridtypes.Deployment, 0)
	contractsData := make([]substrate.BatchCreateContractData, 0)

	mu := sync.Mutex{}

	group, ctx2 := errgroup.WithContext(ctx)
	for node, dls := range deployments {
		// loading node clients first before creating any contract and caching the clients
		_, err := d.ncPool.GetNodeClient(d.substrateConn, node)
		if err != nil {
			return map[uint32][]gridtypes.Deployment{}, errors.Wrap(err, "failed to get node client")
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
		return map[uint32][]gridtypes.Deployment{}, err
	}

	contracts, index, err := d.substrateConn.BatchCreateContract(d.identity, contractsData)
	if err != nil && index == nil {
		return map[uint32][]gridtypes.Deployment{}, errors.Wrap(err, "failed to create contracts")
	}

	var multiErr error
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

			err = d.Wait(ctx, client, &dl)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				multiErr = multierror.Append(multiErr, errors.Wrap(err, "error waiting deployment"))
				failedContracts = append(failedContracts, dl.ContractID)
				return
			}
			deploymentsSlice[i].ContractID = contracts[i]
		}()
	}
	wg.Wait()

	resDeployments := make(map[uint32][]gridtypes.Deployment, len(deployments))
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
func matchOldVersions(oldDl *gridtypes.Deployment, newDl *gridtypes.Deployment) {
	oldWlVersions := map[string]uint32{}
	for _, wl := range oldDl.Workloads {
		oldWlVersions[wl.Name.String()] = wl.Version
	}

	newDl.Version = oldDl.Version

	for idx, wl := range newDl.Workloads {
		newDl.Workloads[idx].Version = oldWlVersions[wl.Name.String()]
	}
}

// assignVersions determines and assigns the versions of the new deployment and its workloads
func assignVersions(oldDl *gridtypes.Deployment, newDl *gridtypes.Deployment) (map[string]uint32, error) {
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
		newHash := newHashes[string(w.Name)]
		oldHash, ok := oldHashes[string(w.Name)]
		if !ok || newHash != oldHash {
			newDl.Workloads[idx].Version = newDl.Version
		}
		newWorkloadsVersions[w.Name.String()] = newDl.Workloads[idx].Version
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
func (d *Deployer) Validate(ctx context.Context, oldDeployments map[uint32]gridtypes.Deployment, newDeployments map[uint32]gridtypes.Deployment) error {
	farmIPs := make(map[int]int)
	nodeMap := make(map[uint32]proxyTypes.NodeWithNestedCapacity)

	for node := range oldDeployments {
		nodeInfo, err := d.gridProxyClient.Node(node)
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
		nodeInfo, err := d.gridProxyClient.Node(node)
		if err != nil {
			return errors.Wrapf(err, "could not get node %d data from the grid proxy", node)
		}
		nodeMap[node] = nodeInfo
		farmIPs[nodeInfo.FarmID] = 0
	}

	for farm := range farmIPs {
		farmUint64 := uint64(farm)
		farmInfo, _, err := d.gridProxyClient.Farms(proxyTypes.FarmFilter{
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
		oldDl, alreadyExists := oldDeployments[node]
		if err := dl.Valid(); err != nil {
			return errors.Wrap(err, "invalid deployment")
		}

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
		if mru < needed.MRU ||
			sru < needed.SRU ||
			hru < needed.HRU {
			free := gridtypes.Capacity{
				HRU: hru,
				MRU: mru,
				SRU: sru,
			}
			return errors.Errorf("node %d does not have enough resources. needed: %v, free: %v", node, capacityPrettyPrint(needed), capacityPrettyPrint(free))
		}
	}
	return nil
}

// capacityPrettyPrint prints the capacity data
func capacityPrettyPrint(cap gridtypes.Capacity) string {
	return fmt.Sprintf("[mru: %d, sru: %d, hru: %d]", cap.MRU, cap.SRU, cap.HRU)
}

// addCapacity adds a new data for capacity
func addCapacity(cap *proxyTypes.Capacity, add *gridtypes.Capacity) {
	cap.CRU += add.CRU
	cap.MRU += add.MRU
	cap.SRU += add.SRU
	cap.HRU += add.HRU
}
