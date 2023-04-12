// Package cmd for handling commands
package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/threefoldtech/tfgrid-sdk-go/grid3-go/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid3-go/workloads"
)

var (
	vmType          = "vm"
	gatewayNameType = "Gateway Name"
	gatewayFQDNType = "Gateway Fqdn"
	k8sType         = "kubernetes"
	// networkType     = "network"
)

// GetVM gets a vm with its project name
func GetVM(t deployer.TFPluginClient, name string) (workloads.Deployment, error) {
	nodeContractIDs, err := getContractsByTypeAndName(t, name, vmType, name)
	if err != nil {
		return workloads.Deployment{}, err
	}
	var nodeID uint32
	for node, contractID := range nodeContractIDs {
		t.State.CurrentNodeDeployments[node] = []uint64{contractID}
		nodeID = node
	}

	return t.State.LoadDeploymentFromGrid(nodeID, name)
}

// GetK8sCluster gets a kubernetes cluster with its project name
func GetK8sCluster(t deployer.TFPluginClient, name string) (workloads.K8sCluster, error) {
	nodeContractIDs, err := getContractsByTypeAndName(t, name, k8sType, name)
	if err != nil {
		return workloads.K8sCluster{}, err
	}
	var nodeIDs []uint32
	for node, contractID := range nodeContractIDs {
		t.State.CurrentNodeDeployments[node] = []uint64{contractID}
		nodeIDs = append(nodeIDs, node)
	}

	return t.State.LoadK8sFromGrid(nodeIDs, name)
}

// GetGatewayName gets a gateway name with its project name
func GetGatewayName(t deployer.TFPluginClient, name string) (workloads.GatewayNameProxy, error) {
	nodeContractIDs, err := getContractsByTypeAndName(t, name, gatewayNameType, name)
	if err != nil {
		return workloads.GatewayNameProxy{}, err
	}
	var nodeID uint32
	for node, contractID := range nodeContractIDs {
		t.State.CurrentNodeDeployments[node] = []uint64{contractID}
		nodeID = node
	}

	return t.State.LoadGatewayNameFromGrid(nodeID, name, name)
}

// GetGatewayFQDN gets a gateway fqdn with its project name
func GetGatewayFQDN(t deployer.TFPluginClient, name string) (workloads.GatewayFQDNProxy, error) {
	nodeContractIDs, err := getContractsByTypeAndName(t, name, gatewayFQDNType, name)
	if err != nil {
		return workloads.GatewayFQDNProxy{}, err
	}
	var nodeID uint32
	for node, contractID := range nodeContractIDs {
		t.State.CurrentNodeDeployments[node] = []uint64{contractID}
		nodeID = node
	}

	return t.State.LoadGatewayFQDNFromGrid(nodeID, name, name)
}

func getContractsByTypeAndName(t deployer.TFPluginClient, projectName, deploymentType, deploymentName string) (map[uint32]uint64, error) {
	contracts, err := t.ContractsGetter.ListContractsOfProjectName(projectName)
	if err != nil {
		return map[uint32]uint64{}, err
	}
	nodeContractIDs := make(map[uint32]uint64)
	for _, contract := range contracts.NodeContracts {
		var deploymentData workloads.DeploymentData
		err := json.Unmarshal([]byte(contract.DeploymentData), &deploymentData)
		if err != nil {
			return map[uint32]uint64{}, err
		}
		if deploymentData.Type != deploymentType || deploymentData.Name != deploymentName {
			continue
		}
		contractID, err := strconv.ParseUint(contract.ContractID, 0, 64)
		if err != nil {
			return map[uint32]uint64{}, err
		}
		nodeContractIDs[contract.NodeID] = contractID
		// only k8s and network have multiple contracts
		if deploymentType == vmType || deploymentType == gatewayFQDNType || deploymentType == gatewayNameType {
			break
		}
	}
	if len(nodeContractIDs) == 0 {
		return map[uint32]uint64{}, fmt.Errorf("no %s with name %s found", deploymentType, deploymentName)
	}
	return nodeContractIDs, nil
}
