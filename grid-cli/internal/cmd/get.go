// Package cmd for handling commands
package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/graphql"
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
	// try to get contracts with the new project name format "vm/<name>"
	projectName := fmt.Sprintf("vm/%s", name)

	var networkContractIDs map[uint32]uint64

	nodeContractIDs, err := t.ContractsGetter.GetNodeContractsByTypeAndName(projectName, workloads.VMType, name)
	if err == nil {
		networkContractIDs, err = t.ContractsGetter.GetNodeContractsByTypeAndName(projectName, workloads.NetworkType, fmt.Sprintf("%snetwork", name))
		if err != nil {
			return workloads.Deployment{}, err
		}

	} else if errors.Is(err, graphql.ErrorContractsNotFound) {
		// if could not find any contracts try to get contracts with the old project name format "<name>"
		nodeContractIDs, err = t.ContractsGetter.GetNodeContractsByTypeAndName(name, workloads.VMType, name)
		if err != nil {
			return workloads.Deployment{}, err
		}

		networkContractIDs, err = t.ContractsGetter.GetNodeContractsByTypeAndName(name, workloads.NetworkType, fmt.Sprintf("%snetwork", name))
		if err != nil {
			return workloads.Deployment{}, err
		}
	} else {
		return workloads.Deployment{}, err
	}

	for node, contractID := range networkContractIDs {
		checkIfExistAndAppend(t, node, contractID)
	}

	var nodeID uint32
	for node, contractID := range nodeContractIDs {
		checkIfExistAndAppend(t, node, contractID)
		nodeID = node
	}

	return t.State.LoadDeploymentFromGrid(ctx, nodeID, name)
}

// GetK8sCluster gets a kubernetes cluster with its project name
func GetK8sCluster(ctx context.Context, t deployer.TFPluginClient, name string) (workloads.K8sCluster, error) {
	// try to get contracts with the new project name format "kubernetes/<name>"
	projectName := fmt.Sprintf("kubernetes/%s", name)

	var networkContractIDs map[uint32]uint64

	nodeContractIDs, err := t.ContractsGetter.GetNodeContractsByTypeAndName(projectName, workloads.K8sType, name)
	if err == nil {
		networkContractIDs, err = t.ContractsGetter.GetNodeContractsByTypeAndName(projectName, workloads.NetworkType, fmt.Sprintf("%snetwork", name))
		if err != nil {
			return workloads.K8sCluster{}, err
		}

	} else if errors.Is(err, graphql.ErrorContractsNotFound) {
		// if could not find any contracts try to get contracts with the old project name format "<name>"
		nodeContractIDs, err = t.ContractsGetter.GetNodeContractsByTypeAndName(name, workloads.K8sType, name)
		if err != nil {
			return workloads.K8sCluster{}, err
		}

		networkContractIDs, err = t.ContractsGetter.GetNodeContractsByTypeAndName(name, workloads.NetworkType, fmt.Sprintf("%snetwork", name))
		if err != nil {
			return workloads.K8sCluster{}, err
		}

	} else {
		return workloads.K8sCluster{}, err
	}

	var nodeIDs []uint32
	for node, contractID := range nodeContractIDs {
		t.State.CurrentNodeDeployments[node] = []uint64{contractID}
		nodeIDs = append(nodeIDs, node)
	}

	for node, contractID := range networkContractIDs {
		t.State.CurrentNodeDeployments[node] = append(t.State.CurrentNodeDeployments[node], contractID)
		nodeIDs = append(nodeIDs, node)
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
