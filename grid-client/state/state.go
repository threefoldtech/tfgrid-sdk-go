// Package state for grid state
package state

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/pkg/errors"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	zosTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"golang.org/x/exp/maps"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// ContractIDs represents a slice of contract IDs
type ContractIDs []uint64

// State struct
type State struct {
	CurrentNodeDeployments map[uint32]ContractIDs

	Networks NetworkState

	NcPool    client.NodeClientGetter
	Substrate subi.SubstrateExt
}

// ErrNotFound for state not found instances
var ErrNotFound = errors.New("not found")

// NewState generates a new state
func NewState(ncPool client.NodeClientGetter, substrate subi.SubstrateExt) *State {
	return &State{
		CurrentNodeDeployments: make(map[uint32]ContractIDs),
		Networks:               NetworkState{State: make(map[string]Network)},
		NcPool:                 ncPool,
		Substrate:              substrate,
	}
}

func (st *State) StoreContractIDs(nodeID uint32, contractIDs ...uint64) {
	for _, contractID := range contractIDs {
		if !slices.Contains(st.CurrentNodeDeployments[nodeID], contractID) {
			st.CurrentNodeDeployments[nodeID] = append(st.CurrentNodeDeployments[nodeID], contractID)
		}
	}
}

func (st *State) RemoveContractIDs(nodeID uint32, contractIDs ...uint64) {
	for _, contractID := range contractIDs {
		st.CurrentNodeDeployments[nodeID] = workloads.Delete(st.CurrentNodeDeployments[nodeID], contractID)
	}
}

// LoadDiskFromGrid loads a disk from grid
func (st *State) LoadDiskFromGrid(ctx context.Context, nodeID uint32, name string, deploymentName string) (workloads.Disk, error) {
	wl, dl, err := st.GetWorkloadInDeployment(ctx, nodeID, name, deploymentName)
	if err != nil {
		return workloads.Disk{}, errors.Wrapf(err, "could not get workload from node %d within deployment %v", nodeID, dl)
	}

	return workloads.NewDiskFromWorkload(&wl)
}

// LoadVolumeFromGrid loads a volume from grid
func (st *State) LoadVolumeFromGrid(ctx context.Context, nodeID uint32, name string, deploymentName string) (workloads.Volume, error) {
	wl, dl, err := st.GetWorkloadInDeployment(ctx, nodeID, name, deploymentName)
	if err != nil {
		return workloads.Volume{}, errors.Wrapf(err, "could not get workload from node %d within deployment %v", nodeID, dl)
	}

	return workloads.NewVolumeFromWorkload(&wl)
}

// LoadGatewayFQDNFromGrid loads a gateway FQDN proxy from grid
func (st *State) LoadGatewayFQDNFromGrid(ctx context.Context, nodeID uint32, name string, deploymentName string) (workloads.GatewayFQDNProxy, error) {
	wl, dl, err := st.GetWorkloadInDeployment(ctx, nodeID, name, deploymentName)
	if err != nil {
		return workloads.GatewayFQDNProxy{}, errors.Wrapf(err, "could not get workload from node %d within deployment %v", nodeID, dl)
	}

	deploymentData, err := workloads.ParseDeploymentData(dl.Metadata)
	if err != nil {
		return workloads.GatewayFQDNProxy{}, errors.Wrapf(err, "could not generate deployment metadata for %s", name)
	}
	gateway, err := workloads.NewGatewayFQDNProxyFromZosWorkload(*wl.Workload3())
	if err != nil {
		return workloads.GatewayFQDNProxy{}, err
	}
	gateway.ContractID = dl.ContractID
	gateway.NodeID = nodeID
	gateway.SolutionType = deploymentData.ProjectName
	gateway.NodeDeploymentID = map[uint32]uint64{nodeID: dl.ContractID}
	return gateway, nil
}

// LoadQSFSFromGrid loads a QSFS from grid
func (st *State) LoadQSFSFromGrid(ctx context.Context, nodeID uint32, name string, deploymentName string) (workloads.QSFS, error) {
	wl, dl, err := st.GetWorkloadInDeployment(ctx, nodeID, name, deploymentName)
	if err != nil {
		return workloads.QSFS{}, errors.Wrapf(err, "could not get workload from node %d within deployment %v", nodeID, dl)
	}

	return workloads.NewQSFSFromWorkload(&wl)
}

// LoadGatewayNameFromGrid loads a gateway name proxy from grid
func (st *State) LoadGatewayNameFromGrid(ctx context.Context, nodeID uint32, name string, deploymentName string) (workloads.GatewayNameProxy, error) {
	wl, dl, err := st.GetWorkloadInDeployment(ctx, nodeID, deploymentName, deploymentName)
	if err != nil {
		return workloads.GatewayNameProxy{}, errors.Wrapf(err, "could not get workload from node %d within deployment %v", nodeID, dl)
	}

	nameContractID, err := st.Substrate.GetContractIDByNameRegistration(name)
	if err != nil {
		return workloads.GatewayNameProxy{}, errors.Wrapf(err, "failed to get gateway name contract %s", name)
	}
	deploymentData, err := workloads.ParseDeploymentData(dl.Metadata)
	if err != nil {
		return workloads.GatewayNameProxy{}, errors.Wrapf(err, "could not generate deployment metadata for %s", deploymentName)
	}
	gateway, err := workloads.NewGatewayNameProxyFromZosWorkload(*wl.Workload3())
	if err != nil {
		return workloads.GatewayNameProxy{}, err
	}
	gateway.NameContractID = nameContractID
	gateway.ContractID = dl.ContractID
	gateway.NodeID = nodeID
	gateway.SolutionType = deploymentData.ProjectName
	gateway.NodeDeploymentID = map[uint32]uint64{nodeID: dl.ContractID}
	return gateway, nil
}

// LoadZdbFromGrid loads a zdb from grid
func (st *State) LoadZdbFromGrid(ctx context.Context, nodeID uint32, name string, deploymentName string) (workloads.ZDB, error) {
	wl, dl, err := st.GetWorkloadInDeployment(ctx, nodeID, name, deploymentName)
	if err != nil {
		return workloads.ZDB{}, errors.Wrapf(err, "could not get workload from node %d within deployment %v", nodeID, dl)
	}

	return workloads.NewZDBFromWorkload(&wl)
}

// LoadVMFromGrid loads a vm from a grid
func (st *State) LoadVMFromGrid(ctx context.Context, nodeID uint32, name string, deploymentName string) (workloads.VM, error) {
	wl, dl, err := st.GetWorkloadInDeployment(ctx, nodeID, name, deploymentName)
	if err != nil {
		return workloads.VM{}, errors.Wrapf(err, "could not get workload from node %d", nodeID)
	}

	return workloads.NewVMFromWorkload(&wl, &dl, nodeID)
}

// LoadVMLightFromGrid loads a vm-light from a grid
func (st *State) LoadVMLightFromGrid(ctx context.Context, nodeID uint32, name string, deploymentName string) (workloads.VMLight, error) {
	wl, dl, err := st.GetWorkloadInDeployment(ctx, nodeID, name, deploymentName)
	if err != nil {
		return workloads.VMLight{}, errors.Wrapf(err, "could not get workload from node %d", nodeID)
	}

	return workloads.NewVMLightFromWorkload(&wl, &dl, nodeID)
}

// LoadK8sFromGrid loads k8s from grid
func (st *State) LoadK8sFromGrid(ctx context.Context, nodeIDs []uint32, deploymentName string) (workloads.K8sCluster, error) {
	clusterDeployments := make(map[uint32]zosTypes.Deployment)
	nodeDeploymentID := map[uint32]uint64{}
	for _, nodeID := range nodeIDs {
		_, deployment, err := st.GetWorkloadInDeployment(ctx, nodeID, "", deploymentName)
		if err != nil {
			return workloads.K8sCluster{}, errors.Wrapf(err, "could not get deployment %s", deploymentName)
		}
		clusterDeployments[nodeID] = deployment
		nodeDeploymentID[nodeID] = deployment.ContractID
	}

	cluster := workloads.K8sCluster{}

	for nodeID, deployment := range clusterDeployments {
		for _, workload := range deployment.Workloads {
			if workload.Type != zos.ZMachineType.String() {
				continue
			}
			workloadDiskSize, workloadComputedIP, workloadComputedIP6, err := st.computeK8sDeploymentResources(deployment)
			if err != nil {
				return workloads.K8sCluster{}, errors.Wrapf(err, "could not compute node %s, resources", workload.Name)
			}

			node, err := workloads.NewK8sNodeFromWorkload(*workload.Workload3(), nodeID, workloadDiskSize[workload.Name], workloadComputedIP[workload.Name], workloadComputedIP6[workload.Name])
			if err != nil {
				return workloads.K8sCluster{}, errors.Wrapf(err, "could not generate node data for %s", workload.Name)
			}

			isMaster, err := isMasterNode(*workload.Workload3())
			if err != nil {
				return workloads.K8sCluster{}, err
			}
			if isMaster {
				cluster.Master = &node
				deploymentData, err := workloads.ParseDeploymentData(deployment.Metadata)
				if err != nil {
					return workloads.K8sCluster{}, errors.Wrapf(err, "could not generate node deployment metadata for %s", workload.Name)
				}
				cluster.SolutionType = deploymentData.ProjectName
				continue
			}
			cluster.Workers = append(cluster.Workers, node)
		}
	}
	if cluster.Master == nil {
		return workloads.K8sCluster{}, errors.Wrapf(ErrNotFound, "failed to get master node for k8s cluster %s", deploymentName)
	}
	cluster.NodeDeploymentID = nodeDeploymentID
	cluster.NetworkName = cluster.Master.NetworkName
	cluster.SSHKey = cluster.Master.EnvVars["SSH_KEY"]
	cluster.Token = cluster.Master.EnvVars["K3S_TOKEN"]
	cluster.Flist = cluster.Master.Flist
	cluster.FlistChecksum = cluster.Master.FlistChecksum
	cluster.Entrypoint = cluster.Master.Entrypoint

	// get cluster IP ranges
	_, err := st.LoadNetworkFromGrid(ctx, cluster.NetworkName)
	if err != nil {
		return workloads.K8sCluster{}, errors.Wrapf(err, "failed to load network %s", cluster.NetworkName)
	}

	err = st.AssignNodesIPRange(&cluster)
	if err != nil {
		return workloads.K8sCluster{}, errors.Errorf("failed to assign ip ranges for k8s cluster %s", deploymentName)
	}

	return cluster, nil
}

func isMasterNode(workload gridtypes.Workload) (bool, error) {
	dataI, err := workload.WorkloadData()
	if err != nil {
		return false, errors.Wrapf(err, "could not get workload %s data", workload.Name)
	}
	data, ok := dataI.(*zos.ZMachine)
	if !ok {
		return false, errors.Wrapf(err, "could not create vm workload from data %v", dataI)
	}
	if data.Env["K3S_URL"] == "" {
		return true, nil
	}
	return false, nil
}

func (st *State) computeK8sDeploymentResources(dl zosTypes.Deployment) (
	workloadDiskSize map[string]uint64,
	workloadComputedIP map[string]string,
	workloadComputedIP6 map[string]string,
	err error,
) {
	workloadDiskSize = make(map[string]uint64)
	workloadComputedIP = make(map[string]string)
	workloadComputedIP6 = make(map[string]string)

	publicIPs := make(map[string]string)
	publicIP6s := make(map[string]string)
	diskSize := make(map[string]uint64)

	for _, w := range dl.Workloads {
		switch w.Type {
		case zos.PublicIPType.String():

			d := zos.PublicIPResult{}
			if err := json.Unmarshal(w.Result.Data, &d); err != nil {
				return workloadDiskSize, workloadComputedIP, workloadComputedIP6, errors.Wrap(err, "failed to load public ip data")
			}
			publicIPs[w.Name] = d.IP.String()
			publicIP6s[w.Name] = d.IPv6.String()

		case zos.ZMountType.String():

			d, err := w.Workload3().WorkloadData()
			if err != nil {
				return workloadDiskSize, workloadComputedIP, workloadComputedIP6, errors.Wrap(err, "failed to load disk data")
			}
			diskSize[w.Name] = uint64(d.(*zos.ZMount).Size / gridtypes.Gigabyte)
		}
	}

	for _, w := range dl.Workloads {
		if w.Type == zos.ZMachineType.String() {
			publicIPKey := fmt.Sprintf("%sip", w.Name)
			diskKey := fmt.Sprintf("%sdisk", w.Name)
			workloadDiskSize[w.Name] = diskSize[diskKey]
			workloadComputedIP[w.Name] = publicIPs[publicIPKey]
			workloadComputedIP6[w.Name] = publicIP6s[publicIPKey]
		}
	}

	return
}

// LoadNetworkFromGrid loads a network from grid
func (st *State) LoadNetworkFromGrid(ctx context.Context, name string) (znet workloads.ZNet, err error) {
	var zNets []workloads.ZNet
	nodeDeploymentsIDs := map[uint32]uint64{}
	publicNodeEndpoint := ""

	sub := st.Substrate
	for nodeID := range st.CurrentNodeDeployments {
		nodeClient, err := st.NcPool.GetNodeClient(sub, nodeID)
		if err != nil {
			return znet, errors.Wrapf(err, "could not get node client: %d", nodeID)
		}

		dls, err := nodeClient.DeploymentList(ctx)
		if err != nil {
			return znet, errors.Wrapf(err, "could not get all deployments from node %d", nodeID)
		}

		for _, dl := range dls {

			if len(strings.TrimSpace(dl.Metadata)) == 0 {
				dl.Metadata, err = getContractMetadata(sub, dl.ContractID, nodeID)
				if err != nil {
					return znet, err
				}
			}

			deploymentData, err := workloads.ParseDeploymentData(dl.Metadata)
			if err != nil {
				return znet, errors.Wrapf(err, "could not generate deployment metadata for %s", name)
			}

			for _, wl := range dl.Workloads {
				if wl.Type == zosTypes.NetworkType && wl.Name == name {
					znet, err = workloads.NewNetworkFromWorkload(wl, nodeID)
					if err != nil {
						return workloads.ZNet{}, errors.Wrapf(err, "failed to get network from workload %s", name)
					}

					znet.SolutionType = deploymentData.ProjectName
					zNets = append(zNets, znet)
					nodeDeploymentsIDs[nodeID] = dl.ContractID

					if znet.PublicNodeID == nodeID {
						// this is the network's public node
						endpoint, err := nodeClient.GetNodeEndpoint(ctx)
						if err != nil {
							return znet, errors.Wrapf(err, "failed to get node %d endpoint", nodeID)
						}
						publicNodeEndpoint = endpoint.String()
					}

					break
				}
			}
		}
	}

	if reflect.DeepEqual(znet, workloads.ZNet{}) {
		return znet, errors.Wrapf(ErrNotFound, "failed to get network %s", name)
	}

	// merge networks
	var nodes []uint32
	myceliumKeys := make(map[uint32][]byte)
	nodesIPRange := map[uint32]zosTypes.IPNet{}
	wgPort := map[uint32]int{}
	keys := map[uint32]wgtypes.Key{}
	for _, net := range zNets {
		maps.Copy(nodesIPRange, net.NodesIPRange)
		maps.Copy(wgPort, net.WGPort)
		maps.Copy(keys, net.Keys)
		maps.Copy(myceliumKeys, net.MyceliumKeys)
		nodes = append(nodes, net.Nodes...)
	}

	znet.NodeDeploymentID = nodeDeploymentsIDs
	znet.Nodes = nodes
	znet.NodesIPRange = nodesIPRange
	znet.MyceliumKeys = myceliumKeys
	znet.Keys = keys
	znet.WGPort = wgPort

	if znet.AddWGAccess {
		znet.AccessWGConfig = workloads.GenerateWGConfig(
			workloads.WgIP(*znet.ExternalIP).IP.String(),
			znet.ExternalSK.String(),
			znet.Keys[znet.PublicNodeID].PublicKey().String(),
			fmt.Sprintf("%s:%d", publicNodeEndpoint, znet.WGPort[znet.PublicNodeID]),
			znet.IPRange.String(),
		)
	}

	st.Networks.UpdateNetworkSubnets(znet.Name, znet.NodesIPRange)
	return znet, nil
}

// LoadNetworkLightFromGrid loads a network-light from grid
func (st *State) LoadNetworkLightFromGrid(ctx context.Context, name string) (znet workloads.ZNetLight, err error) {
	var zNets []workloads.ZNetLight
	nodeDeploymentsIDs := map[uint32]uint64{}

	sub := st.Substrate
	for nodeID := range st.CurrentNodeDeployments {

		nodeClient, err := st.NcPool.GetNodeClient(sub, nodeID)
		if err != nil {
			return znet, errors.Wrapf(err, "could not get node client: %d", nodeID)
		}

		dls, err := nodeClient.DeploymentList(ctx)
		if err != nil {
			return znet, errors.Wrapf(err, "could not get all deployments from node %d", nodeID)
		}

		for _, dl := range dls {
			if len(strings.TrimSpace(dl.Metadata)) == 0 {
				dl.Metadata, err = getContractMetadata(sub, dl.ContractID, nodeID)
				if err != nil {
					return znet, err
				}
			}

			deploymentData, err := workloads.ParseDeploymentData(dl.Metadata)
			if err != nil {
				return znet, errors.Wrapf(err, "could not generate deployment metadata for %s", name)
			}

			for _, wl := range dl.Workloads {
				if wl.Type == zosTypes.NetworkLightType && wl.Name == name {
					znet, err = workloads.NewNetworkLightFromWorkload(wl, nodeID)
					if err != nil {
						return workloads.ZNetLight{}, errors.Wrapf(err, "failed to get network from workload %s", name)
					}

					znet.SolutionType = deploymentData.ProjectName
					zNets = append(zNets, znet)
					nodeDeploymentsIDs[nodeID] = dl.ContractID
					break
				}
			}
		}
	}

	if reflect.DeepEqual(znet, workloads.ZNetLight{}) {
		return znet, errors.Wrapf(ErrNotFound, "failed to get network %s", name)
	}

	// merge networks
	var nodes []uint32
	nodesIPRange := make(map[uint32]zosTypes.IPNet)
	myceliumKeys := make(map[uint32][]byte)
	for _, net := range zNets {
		maps.Copy(nodesIPRange, net.NodesIPRange)
		nodes = append(nodes, net.Nodes...)
		maps.Copy(myceliumKeys, net.MyceliumKeys)
	}

	znet.NodeDeploymentID = nodeDeploymentsIDs
	znet.Nodes = nodes
	znet.NodesIPRange = nodesIPRange
	znet.MyceliumKeys = myceliumKeys

	st.Networks.UpdateNetworkSubnets(znet.Name, znet.NodesIPRange)
	return znet, nil
}

// LoadDeploymentFromGrid loads deployment from grid
func (st *State) LoadDeploymentFromGrid(ctx context.Context, nodeID uint32, name string) (workloads.Deployment, error) {
	_, deployment, err := st.GetWorkloadInDeployment(ctx, nodeID, "", name)
	if err != nil {
		return workloads.Deployment{}, err
	}
	d, err := workloads.NewDeploymentFromZosDeployment(deployment, nodeID)
	if err != nil {
		return workloads.Deployment{}, err
	}
	if d.NetworkName == "" {
		return d, nil
	}

	_, err = st.LoadNetworkFromGrid(ctx, d.NetworkName)
	if err != nil {
		_, err = st.LoadNetworkLightFromGrid(ctx, d.NetworkName)
		if err != nil {
			return workloads.Deployment{}, errors.Wrapf(err, "failed to load network %s", d.NetworkName)
		}
	}
	d.IPrange = st.Networks.GetNetwork(d.NetworkName).Subnets[nodeID]

	return d, nil
}

// GetWorkloadInDeployment return a workload in a deployment using their names and node ID
// if name is empty it returns a deployment with name equal to deploymentName and empty workload
func (st *State) GetWorkloadInDeployment(ctx context.Context, nodeID uint32, name string, deploymentName string) (zosTypes.Workload, zosTypes.Deployment, error) {
	sub := st.Substrate
	nodeClient, err := st.NcPool.GetNodeClient(sub, nodeID)
	if err != nil {
		return zosTypes.Workload{}, zosTypes.Deployment{}, errors.Wrapf(err, "could not get node client: %d", nodeID)
	}

	if _, ok := st.CurrentNodeDeployments[nodeID]; !ok {
		return zosTypes.Workload{}, zosTypes.Deployment{}, errors.Wrapf(ErrNotFound, "failed to find deployment %s on node %d", name, nodeID)
	}

	dls, err := nodeClient.DeploymentList(ctx)
	if err != nil {
		return zosTypes.Workload{}, zosTypes.Deployment{}, errors.Wrapf(err, "could not get deployments from node %d", nodeID)
	}

	for _, dl := range dls {
		if len(strings.TrimSpace(dl.Metadata)) == 0 {
			if err != nil {
				return zosTypes.Workload{}, zosTypes.Deployment{}, errors.Wrapf(err, "could not get contract %d from node %d", dl.ContractID, nodeID)
			}
			dl.Metadata, err = getContractMetadata(sub, dl.ContractID, nodeID)

			if len(strings.TrimSpace(dl.Metadata)) == 0 {
				return zosTypes.Workload{}, zosTypes.Deployment{}, errors.Wrapf(err, "contract %d doesn't have metadata", dl.ContractID)
			}
		}

		dlData, err := workloads.ParseDeploymentData(dl.Metadata)
		if err != nil {
			return zosTypes.Workload{}, zosTypes.Deployment{}, errors.Wrapf(err, "could not get deployment %d data", dl.ContractID)
		}

		if dlData.Name != deploymentName {
			continue
		}

		if name == "" {
			return zosTypes.Workload{}, dl, nil
		}

		for _, wl := range dl.Workloads {
			if wl.Name == name {
				return wl, dl, nil
			}
		}
	}
	return zosTypes.Workload{}, zosTypes.Deployment{}, errors.Wrapf(ErrNotFound, "failed to find workload '%s'", name)
}

// AssignNodesIPRange to assign ip range of k8s cluster nodes
func (st *State) AssignNodesIPRange(k *workloads.K8sCluster) (err error) {
	network := st.Networks.GetNetwork(k.NetworkName)
	nodesIPRange := make(map[uint32]gridtypes.IPNet)
	nodesIPRange[k.Master.NodeID], err = gridtypes.ParseIPNet(network.GetNodeSubnet(k.Master.NodeID))
	if err != nil {
		return errors.Wrap(err, "could not parse master node ip range")
	}
	for _, worker := range k.Workers {
		nodesIPRange[worker.NodeID], err = gridtypes.ParseIPNet(network.GetNodeSubnet(worker.NodeID))
		if err != nil {
			return errors.Wrapf(err, "could not parse worker node (%d) ip range", worker.NodeID)
		}
	}
	k.NodesIPRange = nodesIPRange

	return nil
}

func getContractMetadata(sub subi.SubstrateExt, contractID uint64, nodeID uint32) (string, error) {
	contract, err := sub.GetContract(contractID)
	if err != nil {
		return "", errors.Wrapf(err, "could not get contract %d from node %d", contractID, nodeID)
	}

	metadata := contract.ContractType.NodeContract.DeploymentData
	if len(strings.TrimSpace(metadata)) == 0 {
		return "", errors.Wrapf(err, "contract %d doesn't have metadata", contractID)
	}

	return metadata, nil
}
