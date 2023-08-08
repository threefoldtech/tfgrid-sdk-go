// package tfplugin for threefold plugin client interface and implementation
package tfplugin

import (
	"context"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	gridDeployer "github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/graphql"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// TFPluginClientInterface interface for tfPluginClient
type TFPluginClientInterface interface {
	DeployNetwork(ctx context.Context, znet *workloads.ZNet) error
	DeployDeployment(ctx context.Context, dl *workloads.Deployment) error
	DeployGatewayName(ctx context.Context, gw *workloads.GatewayNameProxy) error
	LoadVMFromGrid(nodeID uint32, name string, deploymentName string) (workloads.VM, error)
	LoadGatewayNameFromGrid(nodeID uint32, name string, deploymentName string) (workloads.GatewayNameProxy, error)
	ListContractsOfProjectName(projectName string) (graphql.Contracts, error)
	CancelByProjectName(projectName string) error
	GetAvailableNode(ctx context.Context, options types.NodeFilter, rootfs uint64) (uint32, error)
	GetGridNetwork() string
	SetState(nodeID uint32, contractIDs []uint64)
}

// NewTFPluginClient returns new tfPluginClient given mnemonics and grid network
func NewTFPluginClient(mnemonics, network string) (TFPluginClient, error) {
	t, err := gridDeployer.NewTFPluginClient(mnemonics, "sr25519", network, "", "", "", 100, false)
	if err != nil {
		return TFPluginClient{}, err
	}
	return TFPluginClient{
		&t,
	}, nil
}

// tfPluginClient wraps grid-client tfPluginClient
type TFPluginClient struct {
	tfPluginClient *gridDeployer.TFPluginClient
}

// DeployNetwork deploys a network deployment to Threefold grid
func (t *TFPluginClient) DeployNetwork(ctx context.Context, znet *workloads.ZNet) error {
	return t.tfPluginClient.NetworkDeployer.Deploy(ctx, znet)
}

// DeployDeployment deploys a deployment to Threefold grid
func (t *TFPluginClient) DeployDeployment(ctx context.Context, dl *workloads.Deployment) error {
	return t.tfPluginClient.DeploymentDeployer.Deploy(ctx, dl)
}

// DeployNameGateway deploys a GatewayName deployment to Threefold grid
func (t *TFPluginClient) DeployGatewayName(ctx context.Context, gw *workloads.GatewayNameProxy) error {
	return t.tfPluginClient.GatewayNameDeployer.Deploy(ctx, gw)
}

// LoadVMFromGrid loads a VM from Threefold grid
func (t *TFPluginClient) LoadVMFromGrid(nodeID uint32, name string, deploymentName string) (workloads.VM, error) {
	return t.tfPluginClient.State.LoadVMFromGrid(nodeID, name, deploymentName)
}

// LoadGatewayNameFromGrid loads a GatewayName from Threefold grid
func (t *TFPluginClient) LoadGatewayNameFromGrid(nodeID uint32, name string, deploymentName string) (workloads.GatewayNameProxy, error) {
	return t.tfPluginClient.State.LoadGatewayNameFromGrid(nodeID, name, deploymentName)
}

// ListContractsOfProjectName returns contracts for a project name from Threefold grid
func (t *TFPluginClient) ListContractsOfProjectName(projectName string) (graphql.Contracts, error) {
	return t.tfPluginClient.ContractsGetter.ListContractsOfProjectName(projectName)
}

// CancelByProjectName cancels a contract on Threefold grid
func (t *TFPluginClient) CancelByProjectName(projectName string) error {
	return t.tfPluginClient.CancelByProjectName(projectName)
}

// GetAvailableNode returns nodes that match the given filter with rootfs specified in GBs
func (t *TFPluginClient) GetAvailableNode(ctx context.Context, options types.NodeFilter, rootfs uint64) (uint32, error) {
	nodes, err := deployer.FilterNodes(ctx, *t.tfPluginClient, options, nil, nil, []uint64{rootfs * 1024 * 1024 * 1024})
	if err != nil {
		return 0, err
	}
	return uint32(nodes[0].NodeID), nil
}

// GetGridNetwork returns the current grid network
func (t *TFPluginClient) GetGridNetwork() string {
	return t.tfPluginClient.Network
}

// SetState set the state of tf plugin client
func (t *TFPluginClient) SetState(nodeID uint32, contractIDs []uint64) {
	t.tfPluginClient.State.CurrentNodeDeployments[nodeID] = contractIDs
}
