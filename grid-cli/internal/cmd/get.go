// Package cmd for handling commands
package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

func checkIfExistAndAppend(t deployer.TFPluginClient, node uint32, contractID uint64) {
	for _, n := range t.State.CurrentNodeDeployments[node] {
		if n == contractID {
			return
		}
	}

	t.State.CurrentNodeDeployments[node] = append(t.State.CurrentNodeDeployments[node], contractID)
}

// GetVM gets a vm with its project name
func GetVM(ctx context.Context, t deployer.TFPluginClient, name string) (workloads.Deployment, error) {
	projectName := name

	// try to get contracts with the old project name format "<name>"
	contracts, err := t.ContractsGetter.ListContractsOfProjectName(projectName, true)
	if err != nil {
		return workloads.Deployment{}, err
	}

	if len(contracts.NodeContracts) == 0 {
		// if could not find any contracts try to get contracts with the new project name format "vm/<name>"
		projectName = fmt.Sprintf("vm/%s", name)
		contracts, err = t.ContractsGetter.ListContractsOfProjectName(projectName, true)
		if err != nil {
			return workloads.Deployment{}, err
		}

		if len(contracts.NodeContracts) == 0 {
			return workloads.Deployment{}, fmt.Errorf("couldn't find any contracts with name %s", name)
		}
	}

	var nodeID uint32

	for _, contract := range contracts.NodeContracts {
		contractID, err := strconv.ParseUint(contract.ContractID, 10, 64)
		if err != nil {
			return workloads.Deployment{}, err
		}

		nodeID = contract.NodeID
		checkIfExistAndAppend(t, nodeID, contractID)

	}

	return t.State.LoadDeploymentFromGrid(ctx, nodeID, name)
}

// GetK8sCluster gets a kubernetes cluster with its project name
func GetK8sCluster(ctx context.Context, t deployer.TFPluginClient, name string) (workloads.K8sCluster, error) {
	projectName := name

	// try to get contracts with the old project name format "<name>"
	contracts, err := t.ContractsGetter.ListContractsOfProjectName(projectName, true)
	if err != nil {
		return workloads.K8sCluster{}, err
	}

	if len(contracts.NodeContracts) == 0 {
		// if could not find any contracts try to get contracts with the new project name format "kubernetes/<name>"
		projectName = fmt.Sprintf("kubernetes/%s", name)
		contracts, err = t.ContractsGetter.ListContractsOfProjectName(projectName, true)
		if err != nil {
			return workloads.K8sCluster{}, err
		}

		if len(contracts.NodeContracts) == 0 {
			return workloads.K8sCluster{}, fmt.Errorf("couldn't find any contracts with name %s", name)
		}
	}

	var nodeIDs []uint32
	for _, contract := range contracts.NodeContracts {
		contractID, err := strconv.ParseUint(contract.ContractID, 10, 64)
		if err != nil {
			return workloads.K8sCluster{}, err
		}

		checkIfExistAndAppend(t, contract.NodeID, contractID)
		nodeIDs = append(nodeIDs, contract.NodeID)
	}

	return t.State.LoadK8sFromGrid(ctx, nodeIDs, name)
}

// GetGatewayName gets a gateway name with its project name
func GetGatewayName(ctx context.Context, t deployer.TFPluginClient, name string) (workloads.GatewayNameProxy, error) {
	nodeContractIDs, err := t.ContractsGetter.GetNodeContractsByTypeAndName(name, workloads.GatewayNameType, name)
	if err != nil {
		return workloads.GatewayNameProxy{}, err
	}
	var nodeID uint32
	for node, contractID := range nodeContractIDs {
		t.State.CurrentNodeDeployments[node] = []uint64{contractID}
		nodeID = node
	}

	return t.State.LoadGatewayNameFromGrid(ctx, nodeID, name, name)
}

// GetGatewayFQDN gets a gateway fqdn with its project name
func GetGatewayFQDN(ctx context.Context, t deployer.TFPluginClient, name string) (workloads.GatewayFQDNProxy, error) {
	nodeContractIDs, err := t.ContractsGetter.GetNodeContractsByTypeAndName(name, workloads.GatewayFQDNType, name)
	if err != nil {
		return workloads.GatewayFQDNProxy{}, err
	}
	var nodeID uint32
	for node, contractID := range nodeContractIDs {
		t.State.CurrentNodeDeployments[node] = []uint64{contractID}
		nodeID = node
	}

	return t.State.LoadGatewayFQDNFromGrid(ctx, nodeID, name, name)
}

// GetDeployment gets a deployment with its project name
func GetDeployment(ctx context.Context, t deployer.TFPluginClient, name string) (workloads.Deployment, error) {
	nodeContractIDs, err := t.ContractsGetter.GetNodeContractsByTypeAndName(name, workloads.VMType, name)
	if err != nil {
		return workloads.Deployment{}, err
	}

	var nodeID uint32
	for node, contractID := range nodeContractIDs {
		checkIfExistAndAppend(t, node, contractID)
		nodeID = node
	}

	return t.State.LoadDeploymentFromGrid(ctx, nodeID, name)
}
