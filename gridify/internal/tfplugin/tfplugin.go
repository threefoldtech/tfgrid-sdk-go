// package tfplugin for threefold plugin client interface and implementation
package tfplugin

import (
	"context"

	"github.com/pkg/errors"
	gridDeployer "github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/graphql"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// TFPluginClientInterface interface for tfPluginClient
type TFPluginClientInterface interface {
	DeployNetwork(ctx context.Context, znet *workloads.ZNet) error
	DeployDeployment(ctx context.Context, dl *workloads.Deployment) error
	DeployGatewayName(ctx context.Context, gw *workloads.GatewayNameProxy) error
	LoadVMFromGrid(nodeID uint32, name string, deploymentName string) (workloads.VM, error)
	LoadGatewayNameFromGrid(nodeID uint32, name string, deploymentName string) (workloads.GatewayNameProxy, error)
	ListContractsOfProjectName(projectName string) (graphql.Contracts, error)
	CancelContract(contractID uint64) error
	FilterNodes(filter types.NodeFilter, pagination types.Limit) (res []types.Node, totalCount int, err error)
	GetGridNetwork() string
	GetDeployment(nodeID uint32, contractID uint64) (gridtypes.Deployment, error)
}

// NewTFPluginClient returns new tfPluginClient given mnemonics and grid network
func NewTFPluginClient(mnemonics, network string) (TFPluginClient, error) {
	t, err := gridDeployer.NewTFPluginClient(mnemonics, "sr25519", network, "", "", "", 100, true, false)
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

// CancelContract cancels a contract on Threefold grid
func (t *TFPluginClient) CancelContract(contractID uint64) error {
	return t.tfPluginClient.SubstrateConn.CancelContract(t.tfPluginClient.Identity, contractID)
}

// FilterNodes returns nodes that match the given filter
func (t *TFPluginClient) FilterNodes(filter types.NodeFilter, pagination types.Limit) (res []types.Node, totalCount int, err error) {
	return t.tfPluginClient.GridProxyClient.Nodes(filter, pagination)
}

// GetGridNetwork returns the current grid network
func (t *TFPluginClient) GetGridNetwork() string {
	return t.tfPluginClient.Network
}

// GetDeployment returns a deployment using node ID and it's contract ID
func (t *TFPluginClient) GetDeployment(nodeID uint32, contractID uint64) (gridtypes.Deployment, error) {
	nodeClient, err := t.tfPluginClient.NcPool.GetNodeClient(t.tfPluginClient.SubstrateConn, nodeID)
	if err != nil {
		return gridtypes.Deployment{}, errors.Wrapf(err, "failed to get node client for node %d", nodeID)
	}
	dl, err := nodeClient.DeploymentGet(context.Background(), contractID)
	if err != nil {
		return gridtypes.Deployment{}, errors.Wrapf(err, "failed to get deployment %d from node %d", contractID, nodeID)
	}
	return dl, err
}
