// Package state for grid state
package state

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
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
	// TODO: remove it and merge with deployments
	CurrentNodeNetworks map[uint32]ContractIDs

	Networks NetworkState

	NcPool    client.NodeClientGetter
	Substrate subi.SubstrateExt
}

// NewState generates a new state
func NewState(ncPool client.NodeClientGetter, substrate subi.SubstrateExt) *State {
	return &State{
		CurrentNodeDeployments: make(map[uint32]ContractIDs),
		CurrentNodeNetworks:    make(map[uint32]ContractIDs),
		Networks:               NetworkState{},
		NcPool:                 ncPool,
		Substrate:              substrate,
	}
}

// LoadDiskFromGrid loads a disk from grid
func (st *State) LoadDiskFromGrid(nodeID uint32, name string, deploymentName string) (workloads.Disk, error) {
	wl, dl, err := st.GetWorkloadInDeployment(nodeID, name, deploymentName)
	if err != nil {
		return workloads.Disk{}, errors.Wrapf(err, "could not get workload from node %d within deployment %v", nodeID, dl)
	}

	return workloads.NewDiskFromWorkload(&wl)
}

// LoadGatewayFQDNFromGrid loads a gateway FQDN proxy from grid
func (st *State) LoadGatewayFQDNFromGrid(nodeID uint32, name string, deploymentName string) (workloads.GatewayFQDNProxy, error) {
	wl, dl, err := st.GetWorkloadInDeployment(nodeID, name, deploymentName)
	if err != nil {
		return workloads.GatewayFQDNProxy{}, errors.Wrapf(err, "could not get workload from node %d within deployment %v", nodeID, dl)
	}

	deploymentData, err := workloads.ParseDeploymentData(dl.Metadata)
	if err != nil {
		return workloads.GatewayFQDNProxy{}, errors.Wrapf(err, "could not generate deployment metadata for %s", name)
	}
	gateway, err := workloads.NewGatewayFQDNProxyFromZosWorkload(wl)
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
func (st *State) LoadQSFSFromGrid(nodeID uint32, name string, deploymentName string) (workloads.QSFS, error) {
	wl, dl, err := st.GetWorkloadInDeployment(nodeID, name, deploymentName)
	if err != nil {
		return workloads.QSFS{}, errors.Wrapf(err, "could not get workload from node %d within deployment %v", nodeID, dl)
	}

	return workloads.NewQSFSFromWorkload(&wl)
}

// LoadGatewayNameFromGrid loads a gateway name proxy from grid
func (st *State) LoadGatewayNameFromGrid(nodeID uint32, name string, deploymentName string) (workloads.GatewayNameProxy, error) {
	wl, dl, err := st.GetWorkloadInDeployment(nodeID, name, deploymentName)
	if err != nil {
		return workloads.GatewayNameProxy{}, errors.Wrapf(err, "could not get workload from node %d within deployment %v", nodeID, dl)
	}

	nameContractID, err := st.Substrate.GetContractIDByNameRegistration(wl.Name.String())
	if err != nil {
		return workloads.GatewayNameProxy{}, errors.Wrapf(err, "failed to get gateway name contract %s", name)
	}
	deploymentData, err := workloads.ParseDeploymentData(dl.Metadata)
	if err != nil {
		return workloads.GatewayNameProxy{}, errors.Wrapf(err, "could not generate deployment metadata for %s", name)
	}
	gateway, err := workloads.NewGatewayNameProxyFromZosWorkload(wl)
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
func (st *State) LoadZdbFromGrid(nodeID uint32, name string, deploymentName string) (workloads.ZDB, error) {
	wl, dl, err := st.GetWorkloadInDeployment(nodeID, name, deploymentName)
	if err != nil {
		return workloads.ZDB{}, errors.Wrapf(err, "could not get workload from node %d within deployment %v", nodeID, dl)
	}

	return workloads.NewZDBFromWorkload(&wl)
}

// LoadVMFromGrid loads a vm from a grid
func (st *State) LoadVMFromGrid(nodeID uint32, name string, deploymentName string) (workloads.VM, error) {
	wl, dl, err := st.GetWorkloadInDeployment(nodeID, name, deploymentName)
	if err != nil {
		return workloads.VM{}, errors.Wrapf(err, "could not get workload from node %d", nodeID)
	}

	return workloads.NewVMFromWorkload(&wl, &dl)
}

// LoadK8sFromGrid loads k8s from grid
func (st *State) LoadK8sFromGrid(nodeIDs []uint32, deploymentName string) (workloads.K8sCluster, error) {
	clusterDeployments := make(map[uint32]gridtypes.Deployment)
	nodeDeploymentID := map[uint32]uint64{}
	for _, nodeID := range nodeIDs {
		_, deployment, err := st.GetWorkloadInDeployment(nodeID, "", deploymentName)
		if err != nil {
			return workloads.K8sCluster{}, errors.Wrapf(err, "could not get deployment %s", deploymentName)
		}
		clusterDeployments[nodeID] = deployment
		nodeDeploymentID[nodeID] = deployment.ContractID
	}

	cluster := workloads.K8sCluster{}

	for nodeID, deployment := range clusterDeployments {
		for _, workload := range deployment.Workloads {
			if workload.Type != zos.ZMachineType {
				continue
			}
			workloadDiskSize, workloadComputedIP, workloadComputedIP6, err := st.computeK8sDeploymentResources(nodeID, deployment)
			if err != nil {
				return workloads.K8sCluster{}, errors.Wrapf(err, "could not compute node %s, resources", workload.Name)
			}

			node, err := workloads.NewK8sNodeFromWorkload(workload, nodeID, workloadDiskSize[workload.Name.String()], workloadComputedIP[workload.Name.String()], workloadComputedIP6[workload.Name.String()])
			if err != nil {
				return workloads.K8sCluster{}, errors.Wrapf(err, "could not generate node data for %s", workload.Name)
			}

			isMaster, err := isMasterNode(workload)
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
		return workloads.K8sCluster{}, errors.Errorf("failed to get master node for k8s cluster %s", deploymentName)
	}
	cluster.NodeDeploymentID = nodeDeploymentID
	cluster.NetworkName = cluster.Master.NetworkName
	cluster.SSHKey = cluster.Master.SSHKey
	cluster.Token = cluster.Master.Token

	// get cluster IP ranges
	_, err := st.LoadNetworkFromGrid(cluster.NetworkName)
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

func (st *State) computeK8sDeploymentResources(nodeID uint32, dl gridtypes.Deployment) (
	workloadDiskSize map[string]int,
	workloadComputedIP map[string]string,
	workloadComputedIP6 map[string]string,
	err error,
) {
	workloadDiskSize = make(map[string]int)
	workloadComputedIP = make(map[string]string)
	workloadComputedIP6 = make(map[string]string)

	publicIPs := make(map[string]string)
	publicIP6s := make(map[string]string)
	diskSize := make(map[string]int)

	for _, w := range dl.Workloads {
		switch w.Type {
		case zos.PublicIPType:

			d := zos.PublicIPResult{}
			if err := json.Unmarshal(w.Result.Data, &d); err != nil {
				return workloadDiskSize, workloadComputedIP, workloadComputedIP6, errors.Wrap(err, "failed to load public ip data")
			}
			publicIPs[string(w.Name)] = d.IP.String()
			publicIP6s[string(w.Name)] = d.IPv6.String()

		case zos.ZMountType:

			d, err := w.WorkloadData()
			if err != nil {
				return workloadDiskSize, workloadComputedIP, workloadComputedIP6, errors.Wrap(err, "failed to load disk data")
			}
			diskSize[string(w.Name)] = int(d.(*zos.ZMount).Size / gridtypes.Gigabyte)
		}
	}

	for _, w := range dl.Workloads {
		if w.Type == zos.ZMachineType {
			publicIPKey := fmt.Sprintf("%sip", w.Name)
			diskKey := fmt.Sprintf("%sdisk", w.Name)
			workloadDiskSize[string(w.Name)] = diskSize[diskKey]
			workloadComputedIP[string(w.Name)] = publicIPs[publicIPKey]
			workloadComputedIP6[string(w.Name)] = publicIP6s[publicIPKey]
		}
	}

	return
}

// LoadNetworkFromGrid loads a network from grid
func (st *State) LoadNetworkFromGrid(name string) (znet workloads.ZNet, err error) {
	var zNets []workloads.ZNet
	nodeDeploymentsIDs := map[uint32]uint64{}
	metdata := workloads.NetworkMetaData{}

	sub := st.Substrate
	for nodeID := range st.CurrentNodeNetworks {
		nodeClient, err := st.NcPool.GetNodeClient(sub, nodeID)
		if err != nil {
			return znet, errors.Wrapf(err, "could not get node client: %d", nodeID)
		}

		for _, contractID := range st.CurrentNodeNetworks[nodeID] {
			dl, err := nodeClient.DeploymentGet(context.Background(), contractID)
			if err != nil {
				return znet, errors.Wrapf(err, "could not get network deployment %d from node %d", contractID, nodeID)
			}

			deploymentData, err := workloads.ParseDeploymentData(dl.Metadata)
			if err != nil {
				return znet, errors.Wrapf(err, "could not generate deployment metadata for %s", name)
			}

			for _, wl := range dl.Workloads {
				if wl.Type == zos.NetworkType && wl.Name == gridtypes.Name(name) {
					znet, err = workloads.NewNetworkFromWorkload(wl, nodeID)
					znet.SolutionType = deploymentData.ProjectName
					zNets = append(zNets, znet)
					nodeDeploymentsIDs[nodeID] = dl.ContractID
					if err != nil {
						return workloads.ZNet{}, errors.Wrapf(err, "failed to get network from workload %s", name)
					}

					if metdata.PublicNodeID == 0 {
						if err := json.Unmarshal([]byte(wl.Metadata), &metdata); err != nil {
							return workloads.ZNet{}, errors.Wrapf(err, "failed to parse network metadata from workload %s", name)
						}
					}

					break
				}
			}
		}
	}

	if reflect.DeepEqual(znet, workloads.ZNet{}) {
		return znet, errors.Errorf("failed to get network %s", name)
	}

	// merge networks
	var nodes []uint32
	nodesIPRange := map[uint32]gridtypes.IPNet{}
	wgPort := map[uint32]int{}
	keys := map[uint32]wgtypes.Key{}
	for _, net := range zNets {
		maps.Copy(nodesIPRange, net.NodesIPRange)
		maps.Copy(wgPort, net.WGPort)
		maps.Copy(keys, net.Keys)
		nodes = append(nodes, net.Nodes...)
	}

	znet.NodeDeploymentID = nodeDeploymentsIDs
	znet.Nodes = nodes
	znet.NodesIPRange = nodesIPRange
	znet.Keys = keys
	znet.WGPort = wgPort
	if metdata.UserAcessIP != "" {
		ip, err := gridtypes.ParseIPNet(metdata.UserAcessIP)
		if err != nil {
			return znet, errors.Wrapf(err, "failed to parse user access ip from metadata string %s", metdata.UserAcessIP)
		}
		znet.ExternalIP = &ip
	}

	if metdata.PrivateKey != "" {
		key, err := wgtypes.ParseKey(metdata.PrivateKey)
		if err != nil {
			return znet, errors.Wrapf(err, "failed to parse user access private key from metadata string %s", metdata.PrivateKey)
		}
		znet.ExternalSK = key
	}

	znet.PublicNodeID = metdata.PublicNodeID
	znet.AddWGAccess = (metdata.PrivateKey != "")

	st.Networks.UpdateNetwork(znet.Name, znet.NodesIPRange)
	return znet, nil
}

// LoadDeploymentFromGrid loads deployment from grid
func (st *State) LoadDeploymentFromGrid(nodeID uint32, name string) (workloads.Deployment, error) {
	_, deployment, err := st.GetWorkloadInDeployment(nodeID, "", name)
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

	_, err = st.LoadNetworkFromGrid(d.NetworkName)
	if err != nil {
		return workloads.Deployment{}, errors.Wrapf(err, "failed to load network %s", d.NetworkName)
	}
	d.IPrange = st.Networks.GetNetwork(d.NetworkName).Subnets[nodeID]

	return d, nil
}

// GetWorkloadInDeployment return a workload in a deployment using their names and node ID
// if name is empty it returns a deployment with name equal to deploymentName and empty workload
func (st *State) GetWorkloadInDeployment(nodeID uint32, name string, deploymentName string) (gridtypes.Workload, gridtypes.Deployment, error) {
	sub := st.Substrate
	if contractIDs, ok := st.CurrentNodeDeployments[nodeID]; ok {
		nodeClient, err := st.NcPool.GetNodeClient(sub, nodeID)
		if err != nil {
			return gridtypes.Workload{}, gridtypes.Deployment{}, errors.Wrapf(err, "could not get node client: %d", nodeID)
		}

		for _, contractID := range contractIDs {
			dl, err := nodeClient.DeploymentGet(context.Background(), contractID)
			if err != nil {
				return gridtypes.Workload{}, gridtypes.Deployment{}, errors.Wrapf(err, "could not get deployment %d from node %d", contractID, nodeID)
			}

			dlData, err := workloads.ParseDeploymentData(dl.Metadata)
			if err != nil {
				return gridtypes.Workload{}, gridtypes.Deployment{}, errors.Wrapf(err, "could not get deployment %d data", contractID)
			}

			if dlData.Name != deploymentName {
				continue
			}

			if name == "" {
				return gridtypes.Workload{}, dl, nil
			}

			for _, workload := range dl.Workloads {
				if workload.Name == gridtypes.Name(name) {
					return workload, dl, nil
				}
			}
		}
		return gridtypes.Workload{}, gridtypes.Deployment{}, errors.Errorf("could not get workload with name %s", name)
	}
	return gridtypes.Workload{}, gridtypes.Deployment{}, errors.Errorf("could not get workload '%s' with node ID %d", name, nodeID)
}

// AssignNodesIPRange to assign ip range of k8s cluster nodes
func (st *State) AssignNodesIPRange(k *workloads.K8sCluster) (err error) {
	network := st.Networks.GetNetwork(k.NetworkName)
	nodesIPRange := make(map[uint32]gridtypes.IPNet)
	nodesIPRange[k.Master.Node], err = gridtypes.ParseIPNet(network.GetNodeSubnet(k.Master.Node))
	if err != nil {
		return errors.Wrap(err, "could not parse master node ip range")
	}
	for _, worker := range k.Workers {
		nodesIPRange[worker.Node], err = gridtypes.ParseIPNet(network.GetNodeSubnet(worker.Node))
		if err != nil {
			return errors.Wrapf(err, "could not parse worker node (%d) ip range", worker.Node)
		}
	}
	k.NodesIPRange = nodesIPRange

	return nil
}
