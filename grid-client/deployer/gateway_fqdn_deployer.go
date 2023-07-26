// Package deployer is grid deployer
package deployer

import (
	"context"

	"github.com/pkg/errors"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// GatewayFQDNDeployer for deploying a GatewayFqdn
type GatewayFQDNDeployer struct {
	tfPluginClient *TFPluginClient
	deployer       MockDeployer
}

// NewGatewayFqdnDeployer generates new gateway fqdn deployer
func NewGatewayFqdnDeployer(tfPluginClient *TFPluginClient) GatewayFQDNDeployer {
	deployer := NewDeployer(*tfPluginClient, true)
	gatewayFQDN := GatewayFQDNDeployer{
		tfPluginClient: tfPluginClient,
		deployer:       &deployer,
	}

	return gatewayFQDN
}

// Validate validates gateway FQDN deployer
func (d *GatewayFQDNDeployer) Validate(ctx context.Context, gw *workloads.GatewayFQDNProxy) error {
	sub := d.tfPluginClient.SubstrateConn
	if len(gw.Name) == 0 {
		return errors.New("gateway workload must have a name")
	}
	if err := validateAccountBalanceForExtrinsics(sub, d.tfPluginClient.Identity); err != nil {
		return err
	}

	nodeClient, err := d.tfPluginClient.NcPool.GetNodeClient(sub, gw.NodeID)
	if err != nil {
		return errors.Wrapf(err, "failed to get node client with ID %d", gw.NodeID)
	}

	cfg, err := nodeClient.NetworkGetPublicConfig(ctx)
	if err != nil {
		return errors.Wrapf(err, "couldn't get node %d public config", gw.NodeID)
	}

	if cfg.IPv4.IP == nil {
		return errors.Errorf("node %d doesn't contain a public IP in its public config", gw.NodeID)
	}

	return client.AreNodesUp(ctx, sub, []uint32{gw.NodeID}, d.tfPluginClient.NcPool)
}

// GenerateVersionlessDeployments generates deployments for gatewayFqdn deployer without versions
func (d *GatewayFQDNDeployer) GenerateVersionlessDeployments(ctx context.Context, gw *workloads.GatewayFQDNProxy) (map[uint32]gridtypes.Deployment, error) {
	deployments := make(map[uint32]gridtypes.Deployment)
	var err error

	dl := workloads.NewGridDeployment(d.tfPluginClient.TwinID, []gridtypes.Workload{})
	dl.Workloads = append(dl.Workloads, gw.ZosWorkload())

	dl.Metadata, err = gw.GenerateMetadata()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate gateway FQDN deployment %s metadata", gw.Name)
	}

	deployments[gw.NodeID] = dl
	return deployments, nil
}

// Deploy deploys the GatewayFQDN deployments using the deployer
func (d *GatewayFQDNDeployer) Deploy(ctx context.Context, gw *workloads.GatewayFQDNProxy) error {
	if err := d.Validate(ctx, gw); err != nil {
		return err
	}

	newDeployments, err := d.GenerateVersionlessDeployments(ctx, gw)
	if err != nil {
		return errors.Wrap(err, "could not generate deployments data")
	}

	// TODO: solution providers
	newDeploymentsSolutionProvider := make(map[uint32]*uint64)
	newDeploymentsSolutionProvider[gw.NodeID] = nil

	gw.NodeDeploymentID, err = d.deployer.Deploy(ctx, gw.NodeDeploymentID, newDeployments, newDeploymentsSolutionProvider)

	// update state
	// error is not returned immediately before updating state because of untracked failed deployments
	if contractID, ok := gw.NodeDeploymentID[gw.NodeID]; ok && contractID != 0 {
		gw.ContractID = contractID
		if !workloads.Contains(d.tfPluginClient.State.CurrentNodeDeployments[gw.NodeID], gw.ContractID) {
			d.tfPluginClient.State.CurrentNodeDeployments[gw.NodeID] = append(d.tfPluginClient.State.CurrentNodeDeployments[gw.NodeID], gw.ContractID)
		}
	}

	return err
}

// BatchDeploy deploys multiple deployments using the deployer
func (d *GatewayFQDNDeployer) BatchDeploy(ctx context.Context, gws []*workloads.GatewayFQDNProxy) error {
	newDeployments := make(map[uint32][]gridtypes.Deployment)
	newDeploymentsSolutionProvider := make(map[uint32][]*uint64)

	for _, gw := range gws {
		if err := d.Validate(ctx, gw); err != nil {
			return err
		}

		dls, err := d.GenerateVersionlessDeployments(ctx, gw)
		if err != nil {
			return errors.Wrap(err, "could not generate deployments data")
		}

		for nodeID, dl := range dls {
			// solution providers
			newDeploymentsSolutionProvider[nodeID] = nil

			if _, ok := newDeployments[nodeID]; !ok {
				newDeployments[nodeID] = []gridtypes.Deployment{dl}
				continue
			}
			newDeployments[nodeID] = append(newDeployments[nodeID], dl)
		}
	}

	newDls, err := d.deployer.BatchDeploy(ctx, newDeployments, newDeploymentsSolutionProvider)

	// update state
	// error is not returned immediately before updating state because of untracked failed deployments
	for _, gw := range gws {
		if err := d.updateStateFromDeployments(ctx, gw, newDls); err != nil {
			return errors.Wrapf(err, "failed to update gateway fqdn '%s' state", gw.Name)
		}
	}

	return err
}

// Cancel cancels a gateway deployment
func (d *GatewayFQDNDeployer) Cancel(ctx context.Context, gw *workloads.GatewayFQDNProxy) (err error) {
	if err := d.Validate(ctx, gw); err != nil {
		return err
	}

	contractID := gw.NodeDeploymentID[gw.NodeID]
	err = d.deployer.Cancel(ctx, []uint64{contractID})
	if err != nil {
		return err
	}

	// update state
	gw.ContractID = 0
	delete(gw.NodeDeploymentID, gw.NodeID)
	d.tfPluginClient.State.CurrentNodeDeployments[gw.NodeID] = workloads.Delete(d.tfPluginClient.State.CurrentNodeDeployments[gw.NodeID], contractID)

	return nil
}

func (d *GatewayFQDNDeployer) updateStateFromDeployments(ctx context.Context, gw *workloads.GatewayFQDNProxy, newDls map[uint32][]gridtypes.Deployment) error {
	gw.NodeDeploymentID = map[uint32]uint64{}

	for _, newDl := range newDls[gw.NodeID] {
		dlData, err := workloads.ParseDeploymentData(newDl.Metadata)
		if err != nil {
			return errors.Wrapf(err, "could not get deployment %d data", newDl.ContractID)
		}

		if dlData.Name == gw.Name {
			gw.NodeDeploymentID[gw.NodeID] = newDl.ContractID
		}
	}

	if contractID, ok := gw.NodeDeploymentID[gw.NodeID]; ok && contractID != 0 {
		gw.ContractID = contractID
		if !workloads.Contains(d.tfPluginClient.State.CurrentNodeDeployments[gw.NodeID], gw.ContractID) {
			d.tfPluginClient.State.CurrentNodeDeployments[gw.NodeID] = append(d.tfPluginClient.State.CurrentNodeDeployments[gw.NodeID], gw.ContractID)
		}
	}

	return nil
}

// TODO: check sync added or not ??
func (d *GatewayFQDNDeployer) syncContracts(ctx context.Context, gw *workloads.GatewayFQDNProxy) (err error) {
	if err := d.tfPluginClient.SubstrateConn.DeleteInvalidContracts(gw.NodeDeploymentID); err != nil {
		return err
	}
	if len(gw.NodeDeploymentID) == 0 {
		gw.ContractID = 0
	}
	return nil
}

// Sync syncs the gateway deployments
func (d *GatewayFQDNDeployer) Sync(ctx context.Context, gw *workloads.GatewayFQDNProxy) error {
	if err := d.syncContracts(ctx, gw); err != nil {
		return errors.Wrap(err, "could not sync contracts")
	}

	dls, err := d.deployer.GetDeployments(ctx, gw.NodeDeploymentID)
	if err != nil {
		return errors.Wrap(err, "could not get deployment objects")
	}

	dl := dls[gw.NodeID]
	wl, _ := dl.Get(gridtypes.Name(gw.Name))

	gwWorkload := workloads.GatewayFQDNProxy{}
	gw.Backends = gwWorkload.Backends
	gw.Name = gwWorkload.Name
	gw.FQDN = gwWorkload.FQDN
	gw.TLSPassthrough = gwWorkload.TLSPassthrough
	gw.Network = gwWorkload.Network

	if wl != nil && wl.Result.State.IsOkay() {
		gwWorkload, err := workloads.NewGatewayFQDNProxyFromZosWorkload(*wl.Workload)
		gw.Backends = gwWorkload.Backends
		gw.Name = gwWorkload.Name
		gw.FQDN = gwWorkload.FQDN
		gw.TLSPassthrough = gwWorkload.TLSPassthrough
		gw.Network = gwWorkload.Network

		if err != nil {
			return err
		}
	}

	return nil
}
