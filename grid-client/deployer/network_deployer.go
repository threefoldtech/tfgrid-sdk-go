package deployer

import (
	"context"
	"fmt"
	"net"
	"slices"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// NetworkDeployer struct
type NetworkDeployer struct {
	tfPluginClient *TFPluginClient
	deployer       MockDeployer
}

// NewNetworkDeployer generates a new network deployer
func NewNetworkDeployer(tfPluginClient *TFPluginClient) NetworkDeployer {
	deployer := NewDeployer(*tfPluginClient, true)
	return NetworkDeployer{
		tfPluginClient: tfPluginClient,
		deployer:       &deployer,
	}
}

// Validate validates a network deployer
func (d *NetworkDeployer) Validate(ctx context.Context, networks []workloads.Network) ([]workloads.Network, error) {
	sub := d.tfPluginClient.SubstrateConn
	var multiErr error

	if err := validateAccountBalanceForExtrinsics(sub, d.tfPluginClient.Identity); err != nil {
		return nil, err
	}

	filteredZNets := make([]workloads.Network, 0)

	for _, znet := range networks {
		if err := znet.Validate(); err != nil {
			multiErr = multierror.Append(multiErr, err)
			continue
		}

		if err := znet.InvalidateBrokenAttributes(d.tfPluginClient.SubstrateConn, d.tfPluginClient.NcPool); err != nil {
			multiErr = multierror.Append(multiErr, err)
			continue
		}

		filteredZNets = append(filteredZNets, znet)
	}

	return filteredZNets, multiErr
}

// GenerateVersionlessDeployments generates deployments for network deployer without versions.
func (d *NetworkDeployer) GenerateVersionlessDeployments(ctx context.Context, zNets []workloads.Network) (map[uint32][]zos.Deployment, error) {
	deployments := make(map[uint32][]zos.Deployment)
	endpoints := make(map[uint32]net.IP)
	nodeUsedPorts := make(map[uint32][]uint16)
	var multiErr error

	allNodes, publicNode, err := d.calcPublicNode(ctx, zNets, endpoints)
	if err != nil {
		multiErr = multierror.Append(multiErr, err)
	}

	for _, znet := range zNets {
		deployment, err := znet.GenerateVersionlessDeployments(
			ctx, d.tfPluginClient.NcPool, d.tfPluginClient.SubstrateConn,
			d.tfPluginClient.TwinID, publicNode, allNodes, endpoints, nodeUsedPorts,
		)
		if err != nil {
			multiErr = multierror.Append(multiErr, err)
			continue
		}
		for nodeID, dl := range deployment {
			deployments[nodeID] = append(deployments[nodeID], dl)
		}
	}

	return deployments, multiErr
}

// Deploy deploys the network deployments using the deployer
func (d *NetworkDeployer) Deploy(ctx context.Context, znet workloads.Network) error {
	zNets, err := d.Validate(ctx, []workloads.Network{znet})
	if err != nil {
		return err
	}

	nodeDeployments, err := d.GenerateVersionlessDeployments(ctx, zNets)
	if err != nil {
		return errors.Wrap(err, "could not generate deployments data")
	}

	newDeployments := make(map[uint32]zos.Deployment)
	for node, deployments := range nodeDeployments {
		if len(deployments) != 1 {
			// this should never happen
			log.Debug().Uint32("node id", node).Msgf("got number of deployment %d, should be 1", len(deployments))
			continue
		}
		newDeployments[node] = deployments[0]
	}

	newDeploymentsSolutionProvider := make(map[uint32]*uint64)
	for _, nodeID := range znet.GetNodes() {
		// solution providers
		newDeploymentsSolutionProvider[nodeID] = nil
	}

	oldDeployments := znet.GetNodeDeploymentID()
	nodeDeploymentIDs, err := d.deployer.Deploy(ctx, oldDeployments, newDeployments, newDeploymentsSolutionProvider)
	znet.SetNodeDeploymentID(nodeDeploymentIDs)

	// update deployment and plugin state
	// error is not returned immediately before updating state because of untracked failed deployments
	nodesUsed := znet.GetNodes()
	if znet.GetPublicNodeID() != 0 && !slices.Contains(znet.GetNodes(), znet.GetPublicNodeID()) {
		nodesUsed = append(nodesUsed, znet.GetPublicNodeID())
	}

	for _, nodeID := range nodesUsed {
		if contractID, ok := znet.GetNodeDeploymentID()[nodeID]; ok && contractID != 0 {
			d.tfPluginClient.State.Networks.UpdateNetworkSubnets(znet.GetName(), znet.GetNodesIPRange())
			d.tfPluginClient.State.StoreContractIDs(nodeID, contractID)
		}
	}

	for nodeID, contract := range oldDeployments {
		// public node is removed
		if _, ok := znet.GetNodeDeploymentID()[nodeID]; !ok {
			d.tfPluginClient.State.RemoveContractIDs(nodeID, contract)
		}
	}

	if err != nil {
		return errors.Wrapf(err, "could not deploy network %s", znet.GetName())
	}

	dls, err := d.deployer.GetDeployments(ctx, znet.GetNodeDeploymentID())
	if err != nil {
		return errors.Wrap(err, "failed to get deployment objects")
	}

	if err := znet.ReadNodesConfig(ctx, dls); err != nil {
		return errors.Wrap(err, "could not read node's data")
	}

	return nil
}

// BatchDeploy deploys multiple network deployments using the deployer
func (d *NetworkDeployer) BatchDeploy(ctx context.Context, zNets []workloads.Network, updateMetadata ...bool) error {
	var multiErr error
	filteredZNets, err := d.Validate(ctx, zNets)
	if err != nil {
		multiErr = multierror.Append(multiErr, err)
	}
	if len(filteredZNets) == 0 {
		return multiErr
	}

	newDeployments, err := d.GenerateVersionlessDeployments(ctx, filteredZNets)
	if err != nil {
		multiErr = multierror.Append(multiErr, err)
	}
	if len(newDeployments) == 0 {
		return multiErr
	}

	newDls, err := d.deployer.BatchDeploy(ctx, newDeployments, make(map[uint32][]*uint64))
	if err != nil {
		multiErr = multierror.Append(multiErr, err)
	}

	update := true
	if len(updateMetadata) != 0 {
		update = updateMetadata[0]
	}

	// update deployment and plugin state
	// error is not returned immediately before updating state because of untracked failed deployments
	for _, znet := range zNets {
		if err := d.updateStateFromDeployments(ctx, znet, newDls, update); err != nil {
			return errors.Wrapf(err, "failed to update network '%s' state", znet.GetName())
		}
	}

	return multiErr
}

// Cancel cancels all the deployments
func (d *NetworkDeployer) Cancel(ctx context.Context, znet workloads.Network) error {
	err := validateAccountBalanceForExtrinsics(d.tfPluginClient.SubstrateConn, d.tfPluginClient.Identity)
	if err != nil {
		return err
	}

	for nodeID, contractID := range znet.GetNodeDeploymentID() {
		err = d.deployer.Cancel(ctx, contractID)
		if err != nil {
			return errors.Wrapf(err, "could not cancel network %s, contract %d", znet.GetName(), contractID)
		}
		znetDeploymentsIDs := znet.GetNodeDeploymentID()
		delete(znetDeploymentsIDs, nodeID)
		znet.SetNodeDeploymentID(znetDeploymentsIDs)
		d.tfPluginClient.State.CurrentNodeDeployments[nodeID] = workloads.Delete(d.tfPluginClient.State.CurrentNodeDeployments[nodeID], contractID)
	}

	// delete network from state if all contracts was deleted
	d.tfPluginClient.State.Networks.DeleteNetwork(znet.GetName())

	dls, err := d.deployer.GetDeployments(ctx, znet.GetNodeDeploymentID())
	if err != nil {
		return errors.Wrap(err, "failed to get deployment objects")
	}

	if err := znet.ReadNodesConfig(ctx, dls); err != nil {
		return errors.Wrap(err, "could not read node's data")
	}

	return nil
}

// BatchCancel cancels all contracts for given networks. if one contracts failed all networks will not be canceled
// and state won't be updated.
func (d *NetworkDeployer) BatchCancel(ctx context.Context, znets []workloads.Network) error {
	var contracts []uint64
	for _, znet := range znets {
		for _, contractID := range znet.GetNodeDeploymentID() {
			if contractID != 0 {
				contracts = append(contracts, contractID)
			}
		}
	}
	err := d.tfPluginClient.BatchCancelContract(contracts)
	if err != nil {
		return fmt.Errorf("failed to cancel contracts: %w", err)
	}
	for _, znet := range znets {
		for nodeID, contractID := range znet.GetNodeDeploymentID() {
			d.tfPluginClient.State.CurrentNodeDeployments[nodeID] = workloads.Delete(d.tfPluginClient.State.CurrentNodeDeployments[nodeID], contractID)
		}

		d.tfPluginClient.State.Networks.DeleteNetwork(znet.GetName())
		znet.SetNodeDeploymentID(make(map[uint32]uint64))
		znet.SetKeys(make(map[uint32]wgtypes.Key))
		znet.SetWGPort(make(map[uint32]int))
		znet.SetNodesIPRange(make(map[uint32]zos.IPNet))
		znet.SetAccessWGConfig("")
	}
	return nil
}

func (d *NetworkDeployer) updateStateFromDeployments(
	ctx context.Context,
	znet workloads.Network,
	dls map[uint32][]zos.Deployment,
	updateMetadata bool,
) error {
	znet.SetNodeDeploymentID(map[uint32]uint64{})

	nodesUsed := znet.GetNodes()
	if znet.GetPublicNodeID() != 0 && !slices.Contains(znet.GetNodes(), znet.GetPublicNodeID()) {
		nodesUsed = append(nodesUsed, znet.GetPublicNodeID())
	}

	for _, nodeID := range nodesUsed {
		// assign NodeDeploymentIDs
		for _, dl := range dls[nodeID] {
			dlData, err := workloads.ParseDeploymentData(dl.Metadata)
			if err != nil {
				return errors.Wrapf(err, "could not get deployment %d data", dl.ContractID)
			}
			if dlData.Name == znet.GetName() && dl.ContractID != 0 {
				znet.GetNodeDeploymentID()[nodeID] = dl.ContractID
			}
		}

		if contractID, ok := znet.GetNodeDeploymentID()[nodeID]; ok && contractID != 0 {
			d.tfPluginClient.State.Networks.UpdateNetworkSubnets(znet.GetName(), znet.GetNodesIPRange())
			d.tfPluginClient.State.StoreContractIDs(nodeID, contractID)
		}
	}

	if !updateMetadata {
		return nil
	}

	deployments, err := d.deployer.GetDeployments(ctx, znet.GetNodeDeploymentID())
	if err != nil {
		return errors.Wrap(err, "failed to get deployment objects")
	}

	if err := znet.ReadNodesConfig(ctx, deployments); err != nil {
		return errors.Wrapf(err, "could not read node's data for network %s", znet.GetName())
	}

	return nil
}

func (d *NetworkDeployer) calcPublicNode(ctx context.Context, znets []workloads.Network, endpoints map[uint32]net.IP) (map[uint32]struct{}, uint32, error) {
	allNodes := make(map[uint32]struct{})
	var multiErr error

	for _, znet := range znets {
		for _, node := range znet.GetNodes() {
			allNodes[node] = struct{}{}
		}
		if znet.GetPublicNodeID() != 0 {
			allNodes[znet.GetPublicNodeID()] = struct{}{}
		}
	}

	var publicNode uint32
	var err error
	for nodeID := range allNodes {
		// we check first that there is a public node in all nodes in the networks.
		// that way we can save extra requests to get a new public node
		// and its used ports and endpoints. this public node also might not be used
		// as access node if there is no need for it.
		if endpoints[nodeID] != nil && endpoints[nodeID].To4() != nil {
			publicNode = nodeID
			break
		}
	}

	// if none of the nodes used are public (have ipv4 endpoint) we do
	// an extra check that at least one network needs a public node before
	// fetching a public node. we can use this one node as an access point
	// to all networks that need it.
	if publicNode == 0 && needPublicNode(znets) {
		publicNode, err = GetPublicNode(ctx, *d.tfPluginClient, nil)
		// we don't return immediately here as there might be some
		// networks that don't need a public node so they should continue
		// processing fine
		if err != nil {
			multiErr = multierror.Append(
				multiErr,
				fmt.Errorf("failed to get public node: %w", err),
			)
		}
	}

	return allNodes, publicNode, multiErr
}

func needPublicNode(znets []workloads.Network) bool {
	// entering here means all nodes for all networks are either hidden or ipv6 only
	// we need an extra public node in two cases:
	// - the user asked for WireGuard access
	// - there are multiple nodes in the network and none of them have ipv4.
	//   because networks must communicate through ipv4
	for _, znet := range znets {
		if znet.GetAddWGAccess() {
			return true
		}
		if len(znet.GetNodes()) > 1 {
			return true
		}
	}
	return false
}
