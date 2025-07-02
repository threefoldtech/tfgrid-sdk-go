package deployer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/pkg/errors"
	zerolog "github.com/rs/zerolog/log"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	zosTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	"github.com/threefoldtech/zosbase/pkg/gridtypes"
	"github.com/threefoldtech/zosbase/pkg/gridtypes/zos"
)

// K8sDeployer for deploying k8s
type K8sDeployer struct {
	tfPluginClient *TFPluginClient
	deployer       MockDeployer
}

// NewK8sDeployer generates new K8s Deployer
func NewK8sDeployer(tfPluginClient *TFPluginClient) K8sDeployer {
	deployer := NewDeployer(*tfPluginClient, true)
	k8sDeployer := K8sDeployer{
		tfPluginClient: tfPluginClient,
		deployer:       &deployer,
	}

	return k8sDeployer
}

// Validate validates K8s deployer
func (d *K8sDeployer) Validate(ctx context.Context, k8sCluster *workloads.K8sCluster) error {
	sub := d.tfPluginClient.SubstrateConn

	if err := validateAccountBalanceForExtrinsics(sub, d.tfPluginClient.Identity); err != nil {
		return d.tfPluginClient.sentry.error(err)
	}

	if err := d.tfPluginClient.State.AssignNodesIPRange(k8sCluster); err != nil {
		return d.tfPluginClient.sentry.error(err)
	}

	if err := k8sCluster.Validate(); err != nil {
		return d.tfPluginClient.sentry.error(err)
	}

	// validate cluster nodes
	var nodes []uint32
	for _, master := range k8sCluster.Masters {
		if !workloads.Contains(nodes, master.NodeID) {
			nodes = append(nodes, master.NodeID)
		}
	}
	for _, worker := range k8sCluster.Workers {
		if !workloads.Contains(nodes, worker.NodeID) {
			nodes = append(nodes, worker.NodeID)
		}
	}
	return d.tfPluginClient.sentry.error(client.AreNodesUp(ctx, sub, nodes, d.tfPluginClient.NcPool))
}

// generateVersionlessDeployments generates a new deployment without a version
func (d *K8sDeployer) generateVersionlessDeployments(k8sCluster *workloads.K8sCluster) (map[uint32]zosTypes.Deployment, error) {
	err := d.assignNodesIPs(k8sCluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to assign node ips")
	}
	deployments := make(map[uint32]zosTypes.Deployment)
	nodeWorkloads := make(map[uint32][]zosTypes.Workload)

	// Generate workloads for all masters
	for idx, master := range k8sCluster.Masters {
		masterWorkloads := []gridtypes.Workload{}
		if idx == 0 {
			masterWorkloads = master.MasterZosWorkload(k8sCluster, workloads.Leader)
		} else {
			masterWorkloads = master.MasterZosWorkload(k8sCluster, workloads.Master)
		}
		for _, m := range masterWorkloads {
			nodeWorkloads[master.NodeID] = append(nodeWorkloads[master.NodeID], zosTypes.NewWorkloadFromZosWorkload(m))
		}
	}
	for _, w := range k8sCluster.Workers {
		workerWorkloads := w.WorkerZosWorkload(k8sCluster)
		for _, wr := range workerWorkloads {
			nodeWorkloads[w.NodeID] = append(nodeWorkloads[w.NodeID], zosTypes.NewWorkloadFromZosWorkload(wr))
		}
	}

	for node, ws := range nodeWorkloads {
		dl := workloads.NewGridDeployment(d.tfPluginClient.TwinID, 0, ws)
		dl.Metadata, err = k8sCluster.GenerateMetadata()
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate deployment metadata")
		}

		deployments[node] = dl
	}
	return deployments, nil
}

// Deploy deploys a k8s cluster deployment
func (d *K8sDeployer) Deploy(ctx context.Context, k8sCluster *workloads.K8sCluster) error {
	if err := d.tfPluginClient.State.AssignNodesIPRange(k8sCluster); err != nil {
		return d.tfPluginClient.sentry.error(err)
	}

	err := k8sCluster.InvalidateBrokenAttributes(d.tfPluginClient.SubstrateConn)
	if err != nil {
		return d.tfPluginClient.sentry.error(err)
	}

	assignNodesFlistsAndEntryPoints(k8sCluster)

	if err := d.Validate(ctx, k8sCluster); err != nil {
		return d.tfPluginClient.sentry.error(err)
	}

	newDeployments, err := d.generateVersionlessDeployments(k8sCluster)
	if err != nil {
		return d.tfPluginClient.sentry.error(errors.Wrap(err, "could not generate k8s grid deployments"))
	}

	newDeploymentsSolutionProvider := make(map[uint32]*uint64)
	for _, master := range k8sCluster.Masters {
		newDeploymentsSolutionProvider[master.NodeID] = nil
	}

	k8sCluster.NodeDeploymentID, err = d.deployer.Deploy(ctx, k8sCluster.NodeDeploymentID, newDeployments, newDeploymentsSolutionProvider)

	// update deployments state
	// error is not returned immediately before updating state because of untracked failed deployments
	for _, master := range k8sCluster.Masters {
		if contractID, ok := k8sCluster.NodeDeploymentID[master.NodeID]; ok && contractID != 0 {
			d.tfPluginClient.State.StoreContractIDs(master.NodeID, contractID)
		}
	}
	for _, w := range k8sCluster.Workers {
		if contractID, ok := k8sCluster.NodeDeploymentID[w.NodeID]; ok && contractID != 0 {
			d.tfPluginClient.State.StoreContractIDs(w.NodeID, contractID)
		}
	}

	return err
}

// BatchDeploy deploys multiple clusters using the deployer
func (d *K8sDeployer) BatchDeploy(ctx context.Context, k8sClusters []*workloads.K8sCluster) error {
	newDeployments := make(map[uint32][]zosTypes.Deployment)
	newDeploymentsSolutionProvider := make(map[uint32][]*uint64)

	for _, k8sCluster := range k8sClusters {
		if err := d.tfPluginClient.State.AssignNodesIPRange(k8sCluster); err != nil {
			return d.tfPluginClient.sentry.error(err)
		}

		err := k8sCluster.InvalidateBrokenAttributes(d.tfPluginClient.SubstrateConn)
		if err != nil {
			return d.tfPluginClient.sentry.error(err)
		}

		assignNodesFlistsAndEntryPoints(k8sCluster)

		if err := d.Validate(ctx, k8sCluster); err != nil {
			return d.tfPluginClient.sentry.error(err)
		}

		dls, err := d.generateVersionlessDeployments(k8sCluster)
		if err != nil {
			return d.tfPluginClient.sentry.error(errors.Wrap(err, "could not generate k8s grid deployments"))
		}

		for nodeID, dl := range dls {
			// solution providers
			newDeploymentsSolutionProvider[nodeID] = nil

			if _, ok := newDeployments[nodeID]; !ok {
				newDeployments[nodeID] = []zosTypes.Deployment{dl}
				continue
			}
			newDeployments[nodeID] = append(newDeployments[nodeID], dl)
		}
	}

	newDls, err := d.deployer.BatchDeploy(ctx, newDeployments, newDeploymentsSolutionProvider)

	// update deployments state
	// error is not returned immediately before updating state because of untracked failed deployments
	for _, k8sCluster := range k8sClusters {
		if err := d.updateStateFromDeployments(k8sCluster, newDls); err != nil {
			// TODO: add name for cluster, or pick leader
			clusterName := "unknown"
			if len(k8sCluster.Masters) > 0 {
				clusterName = k8sCluster.Masters[0].Name
			}
			return d.tfPluginClient.sentry.error(errors.Wrapf(err, "failed to update cluster '%s' state", clusterName))
		}
	}

	return d.tfPluginClient.sentry.error(err)
}

// Cancel cancels a k8s cluster deployment
func (d *K8sDeployer) Cancel(ctx context.Context, k8sCluster *workloads.K8sCluster) (err error) {
	for nodeID, contractID := range k8sCluster.NodeDeploymentID {
		for _, master := range k8sCluster.Masters {
			if master.NodeID == nodeID {
				err = d.deployer.Cancel(ctx, contractID)
				if err != nil {
					return d.tfPluginClient.sentry.error(errors.Wrapf(err, "could not cancel master %s, contract %d", master.Name, contractID))

				}
				d.tfPluginClient.State.CurrentNodeDeployments[nodeID] = workloads.Delete(d.tfPluginClient.State.CurrentNodeDeployments[nodeID], contractID)
				delete(k8sCluster.NodeDeploymentID, nodeID)
			}
		}
		for _, worker := range k8sCluster.Workers {
			if worker.NodeID == nodeID {
				err = d.deployer.Cancel(ctx, contractID)
				if err != nil {
					return d.tfPluginClient.sentry.error(errors.Wrapf(err, "could not cancel worker %s, contract %d", worker.Name, contractID))
				}
				d.tfPluginClient.State.CurrentNodeDeployments[nodeID] = workloads.Delete(d.tfPluginClient.State.CurrentNodeDeployments[nodeID], contractID)
				delete(k8sCluster.NodeDeploymentID, nodeID)
				// break // TODO: multiple k8s nodes on same grid node?
			}
		}
	}

	return nil
}

func (d *K8sDeployer) updateStateFromDeployments(k8sCluster *workloads.K8sCluster, newDl map[uint32][]zosTypes.Deployment) error {
	var k8sNodes []uint32
	for _, master := range k8sCluster.Masters {
		k8sNodes = append(k8sNodes, master.NodeID)
	}
	for _, w := range k8sCluster.Workers {
		k8sNodes = append(k8sNodes, w.NodeID)
	}

	k8sCluster.NodeDeploymentID = map[uint32]uint64{}

	for _, k8sNode := range k8sNodes {
		for _, newDl := range newDl[k8sNode] {
			dlData, err := workloads.ParseDeploymentData(newDl.Metadata)
			if err != nil {
				return errors.Wrapf(err, "could not get deployment %d data", newDl.ContractID)
			}

			// TODO: indecator for a leader, or save all masters nodeids
			for _, master := range k8sCluster.Masters {
				if dlData.Name == master.Name {
					k8sCluster.NodeDeploymentID[master.NodeID] = newDl.ContractID
					break
				}
			}
		}
	}

	// Update current node deployments for all masters
	for _, master := range k8sCluster.Masters {
		if contractID, ok := k8sCluster.NodeDeploymentID[master.NodeID]; ok && contractID != 0 {
			if !workloads.Contains(d.tfPluginClient.State.CurrentNodeDeployments[master.NodeID], contractID) {
				d.tfPluginClient.State.CurrentNodeDeployments[master.NodeID] = append(d.tfPluginClient.State.CurrentNodeDeployments[master.NodeID], contractID)
			}
		}
	}

	// Update current node deployments for all workers
	for _, w := range k8sCluster.Workers {
		if contractID, ok := k8sCluster.NodeDeploymentID[w.NodeID]; ok && contractID != 0 {
			if !workloads.Contains(d.tfPluginClient.State.CurrentNodeDeployments[w.NodeID], contractID) {
				d.tfPluginClient.State.CurrentNodeDeployments[w.NodeID] = append(d.tfPluginClient.State.CurrentNodeDeployments[w.NodeID], contractID)
			}
		}
	}

	return nil
}

// UpdateFromRemote update a k8s cluster
func (d *K8sDeployer) UpdateFromRemote(ctx context.Context, k8sCluster *workloads.K8sCluster) error {
	if err := d.removeDeletedContracts(k8sCluster); err != nil {
		return d.tfPluginClient.sentry.error(errors.Wrap(err, "failed to remove deleted contracts"))
	}
	currentDeployments, err := d.deployer.GetDeployments(ctx, k8sCluster.NodeDeploymentID)
	if err != nil {
		return d.tfPluginClient.sentry.error(errors.Wrap(err, "failed to fetch remote deployments"))
	}
	zerolog.Debug().Msg("calling updateFromRemote")

	keyUpdated, tokenUpdated, networkUpdated := false, false, false
	// calculate k's properties from the currently deployed deployments
	for _, dl := range currentDeployments {
		for _, w := range dl.Workloads {
			if w.Type == zosTypes.ZMachineType {
				d, err := w.Workload().WorkloadData()
				if err != nil {
					zerolog.Error().Err(err).Msg("failed to get workload data")
				}
				SSHKey := d.(*zos.ZMachine).Env["SSH_KEY"]
				token := d.(*zos.ZMachine).Env["K3S_TOKEN"]
				networkName := string(d.(*zos.ZMachine).Network.Interfaces[0].Network)
				if !keyUpdated && SSHKey != k8sCluster.SSHKey {
					k8sCluster.SSHKey = SSHKey
					keyUpdated = true
				}
				if !tokenUpdated && token != k8sCluster.Token {
					k8sCluster.Token = token
					tokenUpdated = true
				}
				if !networkUpdated && networkName != k8sCluster.NetworkName {
					k8sCluster.NetworkName = networkName
					networkUpdated = true
				}
			}
		}
	}

	nodeDeploymentID := make(map[uint32]uint64)
	for node, dl := range currentDeployments {
		nodeDeploymentID[node] = dl.ContractID
	}
	k8sCluster.NodeDeploymentID = nodeDeploymentID
	// maps from workload name to (public ip, node id, disk size, actual workload)
	workloadNodeID := make(map[string]uint32)
	workloadDiskSize := make(map[string]uint64)
	workloadComputedIP := make(map[string]string)
	workloadComputedIP6 := make(map[string]string)
	workloadObj := make(map[string]gridtypes.Workload)

	publicIPs := make(map[string]string)
	publicIP6s := make(map[string]string)
	diskSize := make(map[string]uint64)
	for node, dl := range currentDeployments {
		for _, w := range dl.Workloads {
			switch w.Type {
			case zosTypes.ZMachineType:
				workloadNodeID[w.Name] = node
				workloadObj[w.Name] = *w.Workload()

			case zosTypes.PublicIPType:
				ipResult := zos.PublicIPResult{}
				// TODO: right?
				if err := json.Unmarshal(w.Result.Data, &ipResult); err != nil {
					return d.tfPluginClient.sentry.error(errors.Wrap(err, "failed to load public ip data"))
				}
				publicIPs[w.Name] = ipResult.IP.String()
				publicIP6s[w.Name] = ipResult.IPv6.String()

			case zosTypes.ZMountType:
				wl, err := w.Workload().WorkloadData()
				if err != nil {
					return d.tfPluginClient.sentry.error(errors.Wrap(err, "failed to load disk data"))
				}
				diskSize[w.Name] = uint64(wl.(*zos.ZMount).Size) / zosTypes.Gigabyte
			}
		}
	}
	for _, dl := range currentDeployments {
		for _, w := range dl.Workloads {
			if w.Type == zosTypes.ZMachineType {
				publicIPKey := fmt.Sprintf("%sip", w.Name)
				diskKey := fmt.Sprintf("%sdisk", w.Name)
				workloadDiskSize[w.Name] = diskSize[diskKey]
				workloadComputedIP[w.Name] = publicIPs[publicIPKey]
				workloadComputedIP6[w.Name] = publicIP6s[publicIPKey]
			}
		}
	}
	// update masters
	updatedMasters := make([]workloads.K8sNode, 0)
	for _, master := range k8sCluster.Masters {
		masterNodeID, ok := workloadNodeID[master.Name]
		if !ok {
			// master doesn't exist in any deployment, skip it
			continue
		}
		delete(workloadNodeID, master.Name)
		masterWorkload := workloadObj[master.Name]
		masterIP := workloadComputedIP[master.Name]
		masterIP6 := workloadComputedIP6[master.Name]
		masterDiskSize := workloadDiskSize[master.Name]

		m, err := workloads.NewK8sNodeFromWorkload(masterWorkload, masterNodeID, masterDiskSize, masterIP, masterIP6)
		if err != nil {
			return d.tfPluginClient.sentry.error(errors.Wrap(err, "failed to get master node from workload"))
		}
		updatedMasters = append(updatedMasters, m)
	}
	k8sCluster.Masters = updatedMasters

	// update workers
	workers := make([]workloads.K8sNode, 0)
	for _, w := range k8sCluster.Workers {
		workerNodeID, ok := workloadNodeID[w.Name]
		if !ok {
			// worker doesn't exist in any deployment, skip it
			continue
		}
		delete(workloadNodeID, w.Name)
		workerWorkload := workloadObj[w.Name]
		workerIP := workloadComputedIP[w.Name]
		workerIP6 := workloadComputedIP6[w.Name]

		workerDiskSize := workloadDiskSize[w.Name]
		w, err := workloads.NewK8sNodeFromWorkload(workerWorkload, workerNodeID, workerDiskSize, workerIP, workerIP6)
		if err != nil {
			return d.tfPluginClient.sentry.error(errors.Wrap(err, "failed to get worker data from workload"))
		}
		workers = append(workers, w)
	}
	// add missing workers (in case of failed deletions) TODO: why?
	for name, workerNodeID := range workloadNodeID {
		isMaster := false
		for _, master := range k8sCluster.Masters {
			if name == master.Name {
				isMaster = true
				break
			}
		}
		if isMaster {
			continue
		}

		workerWorkload := workloadObj[name]
		workerIP := workloadComputedIP[name]
		workerIP6 := workloadComputedIP6[name]
		workerDiskSize := workloadDiskSize[name]
		w, err := workloads.NewK8sNodeFromWorkload(workerWorkload, workerNodeID, workerDiskSize, workerIP, workerIP6)
		if err != nil {
			return d.tfPluginClient.sentry.error(errors.Wrap(err, "failed to get worker data from workload"))
		}
		workers = append(workers, w)
	}
	k8sCluster.Workers = workers
	zerolog.Debug().Msg("after updateFromRemote\n")
	enc := json.NewEncoder(log.Writer())
	enc.SetIndent("", "  ")
	err = enc.Encode(k8sCluster)
	if err != nil {
		return d.tfPluginClient.sentry.error(errors.Wrap(err, "failed to encode k8s deployer"))
	}

	return nil
}

func (d *K8sDeployer) removeDeletedContracts(k8sCluster *workloads.K8sCluster) error {
	sub := d.tfPluginClient.SubstrateConn
	nodeDeploymentID := make(map[uint32]uint64)
	for nodeID, deploymentID := range k8sCluster.NodeDeploymentID {
		cont, err := sub.GetContract(deploymentID)
		if err != nil {
			return errors.Wrap(err, "failed to get deployments")
		}
		if !cont.IsDeleted() {
			nodeDeploymentID[nodeID] = deploymentID
		}
	}
	k8sCluster.NodeDeploymentID = nodeDeploymentID
	return nil
}

// TODO: integrate new list private ips function
func (d *K8sDeployer) getK8sUsedIPs(k8s *workloads.K8sCluster) map[uint32][]byte {
	usedIPs := make(map[uint32][]byte)

	for _, master := range k8s.Masters {
		if master.IP != "" {
			ip := net.ParseIP(master.IP).To4()
			if ip != nil {
				usedIPs[master.NodeID] = append(usedIPs[master.NodeID], ip[3])
			}
		}
	}

	for _, w := range k8s.Workers {
		if w.IP != "" {
			ip := net.ParseIP(w.IP).To4()
			if ip != nil {
				usedIPs[w.NodeID] = append(usedIPs[w.NodeID], ip[3])
			}
		}
	}

	return usedIPs
}

func (d *K8sDeployer) getK8sFreeIP(ipRange gridtypes.IPNet, nodeID uint32, k8s *workloads.K8sCluster) (string, error) {
	nodeUsedIPs := d.getK8sUsedIPs(k8s)

	ip := ipRange.IP.To4()
	if ip == nil {
		return "", errors.Errorf("the provided ip range (%s) is not a valid ipv4", ipRange.String())
	}

	for i := 2; i < 255; i++ {
		hostID := byte(i)
		if !workloads.Contains(nodeUsedIPs[nodeID], hostID) {
			nodeUsedIPs[nodeID] = append(nodeUsedIPs[nodeID], hostID)
			ip[3] = hostID
			return ip.String(), nil
		}
	}
	return "", errors.New("all ips are used")
}

func (d *K8sDeployer) assignNodesIPs(k8sCluster *workloads.K8sCluster) error {
	for idx, master := range k8sCluster.Masters {
		masterNodeRange := k8sCluster.NodesIPRange[master.NodeID]
		if master.IP != "" || masterNodeRange.Contains(net.ParseIP(master.IP)) {
			continue
		}

		ip, err := d.getK8sFreeIP(masterNodeRange, master.NodeID, k8sCluster)
		if err != nil {
			return errors.Wrapf(err, "failed to find free ip for master %s", master.Name)
		}
		k8sCluster.Masters[idx].IP = ip

	}
	for idx, w := range k8sCluster.Workers {
		workerNodeRange := k8sCluster.NodesIPRange[w.NodeID]
		if w.IP != "" && workerNodeRange.Contains(net.ParseIP(w.IP)) {
			continue
		}

		ip, err := d.getK8sFreeIP(workerNodeRange, w.NodeID, k8sCluster)
		if err != nil {
			return errors.Wrapf(err, "failed to find free ip for worker %s", w.Name)
		}
		k8sCluster.Workers[idx].IP = ip
	}
	return nil
}

func assignNodesFlistsAndEntryPoints(k *workloads.K8sCluster) {
	// TODO: use Leader indecator or require cluster flist
	if k.Flist == "" && len(k.Masters) > 0 {
		k.Flist = k.Masters[0].Flist
	}
	if k.Entrypoint == "" {
		if len(k.Masters) > 0 && k.Masters[0].Entrypoint != "" {
			k.Entrypoint = k.Masters[0].Entrypoint
		} else {
			k.Entrypoint = "/sbin/zinit init" // set default value
		}
	}

	for i := range k.Masters {
		k.Masters[i].Flist = k.Flist
		k.Masters[i].Entrypoint = k.Entrypoint
	}
	for i := range k.Workers {
		k.Workers[i].Flist = k.Flist
		k.Workers[i].Entrypoint = k.Entrypoint
	}
}
