// Package cmd for handling commands
package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/threefoldtech/tfgrid-sdk-go/grid3-go/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid3-go/workloads"
)

// GetVM gets a vm with its project name
func GetVM(t deployer.TFPluginClient, name string) (workloads.Deployment, error) {
	contracts, err := t.ContractsGetter.ListContractsOfProjectName(name)
	if err != nil {
		return workloads.Deployment{}, err
	}
	var nodeID uint32
	var contractID uint64
	for _, contract := range contracts.NodeContracts {
		var deploymentData workloads.DeploymentData
		err := json.Unmarshal([]byte(contract.DeploymentData), &deploymentData)
		if err != nil {
			return workloads.Deployment{}, err
		}
		if deploymentData.Type != "vm" || deploymentData.ProjectName != name {
			continue
		}
		nodeID = contract.NodeID
		contractID, err = strconv.ParseUint(contract.ContractID, 0, 64)
		if err != nil {
			return workloads.Deployment{}, err
		}

		t.State.CurrentNodeDeployments[nodeID] = []uint64{contractID}
		break
	}
	if nodeID == 0 {
		return workloads.Deployment{}, fmt.Errorf("no vm with name %s found", name)
	}

	return t.State.LoadDeploymentFromGrid(nodeID, name)
}

// GetK8sCluster gets a kubernetes cluster with its project name
func GetK8sCluster(t deployer.TFPluginClient, name string) (workloads.K8sCluster, error) {
	contracts, err := t.ContractsGetter.ListContractsOfProjectName(name)
	if err != nil {
		return workloads.K8sCluster{}, err
	}
	var nodeIDs []uint32
	for _, contract := range contracts.NodeContracts {
		var deploymentData workloads.DeploymentData
		err := json.Unmarshal([]byte(contract.DeploymentData), &deploymentData)
		if err != nil {
			return workloads.K8sCluster{}, err
		}
		if deploymentData.Type != "kubernetes" || deploymentData.Name != name {
			continue
		}
		nodeIDs = append(nodeIDs, contract.NodeID)
		contractID, err := strconv.ParseUint(contract.ContractID, 0, 64)
		if err != nil {
			return workloads.K8sCluster{}, err
		}
		t.State.CurrentNodeDeployments[contract.NodeID] = append(t.State.CurrentNodeDeployments[contract.NodeID], contractID)
	}
	if nodeIDs == nil {
		return workloads.K8sCluster{}, fmt.Errorf("no k8s cluster with name %s found", name)
	}
	return t.State.LoadK8sFromGrid(nodeIDs, name)
}

// GetGatewayName gets a gateway name with its project name
func GetGatewayName(t deployer.TFPluginClient, name string) (workloads.GatewayNameProxy, error) {
	contracts, err := t.ContractsGetter.ListContractsOfProjectName(name)
	if err != nil {
		return workloads.GatewayNameProxy{}, err
	}
	var nodeID uint32
	var contractID uint64
	for _, contract := range contracts.NodeContracts {
		var deploymentData workloads.DeploymentData
		err := json.Unmarshal([]byte(contract.DeploymentData), &deploymentData)
		if err != nil {
			return workloads.GatewayNameProxy{}, err
		}
		if deploymentData.Type != "Gateway Name" || deploymentData.Name != name {
			continue
		}
		nodeID = contract.NodeID
		contractID, err = strconv.ParseUint(contract.ContractID, 0, 64)
		if err != nil {
			return workloads.GatewayNameProxy{}, err
		}

		t.State.CurrentNodeDeployments[nodeID] = []uint64{contractID}
		break
	}
	if nodeID == 0 {
		return workloads.GatewayNameProxy{}, fmt.Errorf("no gateway name with name %s found", name)
	}
	return t.State.LoadGatewayNameFromGrid(nodeID, name, name)
}

// GetGatewayFQDN gets a gateway fqdn with its project name
func GetGatewayFQDN(t deployer.TFPluginClient, name string) (workloads.GatewayFQDNProxy, error) {
	contracts, err := t.ContractsGetter.ListContractsOfProjectName(name)
	if err != nil {
		return workloads.GatewayFQDNProxy{}, err
	}
	var nodeID uint32
	var contractID uint64
	for _, contract := range contracts.NodeContracts {
		var deploymentData workloads.DeploymentData
		err := json.Unmarshal([]byte(contract.DeploymentData), &deploymentData)
		if err != nil {
			return workloads.GatewayFQDNProxy{}, err
		}
		if deploymentData.Type != "Gateway Fqdn" || deploymentData.Name != name {
			continue
		}
		nodeID = contract.NodeID
		contractID, err = strconv.ParseUint(contract.ContractID, 0, 64)
		if err != nil {
			return workloads.GatewayFQDNProxy{}, err
		}

		t.State.CurrentNodeDeployments[nodeID] = []uint64{contractID}
		break
	}
	if nodeID == 0 {
		return workloads.GatewayFQDNProxy{}, fmt.Errorf("no gateway fqdn with name %s found", name)
	}
	return t.State.LoadGatewayFQDNFromGrid(nodeID, name, name)
}
