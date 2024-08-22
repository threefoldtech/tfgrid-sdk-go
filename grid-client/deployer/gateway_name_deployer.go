package deployer

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// GatewayNameDeployer for deploying a GatewayName
type GatewayNameDeployer struct {
	tfPluginClient *TFPluginClient
	deployer       MockDeployer
}

// NewGatewayNameDeployer generates new gateway name deployer
func NewGatewayNameDeployer(tfPluginClient *TFPluginClient) GatewayNameDeployer {
	deployer := NewDeployer(*tfPluginClient, true)
	gatewayName := GatewayNameDeployer{
		tfPluginClient: tfPluginClient,
		deployer:       &deployer,
	}

	return gatewayName
}

// Validate validates gatewayName deployer
func (d *GatewayNameDeployer) Validate(ctx context.Context, gw *workloads.GatewayNameProxy) error {
	sub := d.tfPluginClient.SubstrateConn
	if err := validateAccountBalanceForExtrinsics(sub, d.tfPluginClient.Identity); err != nil {
		return err
	}
	return client.AreNodesUp(ctx, sub, []uint32{gw.NodeID}, d.tfPluginClient.NcPool)
}

// GenerateVersionlessDeployments generates deployments for gateway name deployer without versions
func (d *GatewayNameDeployer) GenerateVersionlessDeployments(ctx context.Context, gw *workloads.GatewayNameProxy) (map[uint32]gridtypes.Deployment, error) {
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

// Deploy deploys the GatewayName deployments using the deployer
func (d *GatewayNameDeployer) Deploy(ctx context.Context, gw *workloads.GatewayNameProxy) error {
	if err := d.Validate(ctx, gw); err != nil {
		return err
	}
	newDeployments, err := d.GenerateVersionlessDeployments(ctx, gw)
	if err != nil {
		return errors.Wrap(err, "could not generate deployments data")
	}

	newDeploymentsSolutionProvider := make(map[uint32]*uint64)
	newDeploymentsSolutionProvider[gw.NodeID] = nil

	if err := d.InvalidateNameContract(ctx, gw); err != nil {
		return err
	}

	if gw.NameContractID == 0 {
		gw.NameContractID, err = d.tfPluginClient.SubstrateConn.CreateNameContract(d.tfPluginClient.Identity, gw.Name)
		if err != nil {
			return err
		}
	}

	gw.NodeDeploymentID, err = d.deployer.Deploy(ctx, gw.NodeDeploymentID, newDeployments, newDeploymentsSolutionProvider)
	if err != nil {
		cancelErr := d.tfPluginClient.SubstrateConn.CancelContract(d.tfPluginClient.Identity, gw.NameContractID)
		if cancelErr != nil {
			return fmt.Errorf("failed to deploy gateway name %v, failed to cancel gateway name contract %v", err, cancelErr)
		}
		return errors.Wrapf(err, "failed to deploy gateway name id: %v", gw.NodeDeploymentID)
	}
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
func (d *GatewayNameDeployer) BatchDeploy(ctx context.Context, gws []*workloads.GatewayNameProxy) error {
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

		if err := d.InvalidateNameContract(ctx, gw); err != nil {
			return err
		}
		if gw.NameContractID == 0 {
			gw.NameContractID, err = d.tfPluginClient.SubstrateConn.CreateNameContract(d.tfPluginClient.Identity, gw.Name)
			if err != nil {
				return err
			}
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

// Cancel cancels the gatewayName deployment
func (d *GatewayNameDeployer) Cancel(ctx context.Context, gw *workloads.GatewayNameProxy) (err error) {
	contractID := gw.NodeDeploymentID[gw.NodeID]
	err = d.deployer.Cancel(ctx, contractID)
	if err != nil {
		return err
	}

	gw.ContractID = 0
	delete(gw.NodeDeploymentID, gw.NodeID)
	d.tfPluginClient.State.CurrentNodeDeployments[gw.NodeID] = workloads.Delete(d.tfPluginClient.State.CurrentNodeDeployments[gw.NodeID], contractID)

	if gw.NameContractID != 0 {
		if err := d.tfPluginClient.SubstrateConn.EnsureContractCanceled(d.tfPluginClient.Identity, gw.NameContractID); err != nil {
			return err
		}
		gw.NameContractID = 0
	}

	return nil
}

func (d *GatewayNameDeployer) updateStateFromDeployments(ctx context.Context, gw *workloads.GatewayNameProxy, newDls map[uint32][]gridtypes.Deployment) error {
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

// InvalidateNameContract invalidates name contract
func (d *GatewayNameDeployer) InvalidateNameContract(ctx context.Context, gw *workloads.GatewayNameProxy) (err error) {
	if gw.NameContractID == 0 {
		return
	}

	gw.NameContractID, err = d.tfPluginClient.SubstrateConn.InvalidateNameContract(
		ctx,
		d.tfPluginClient.Identity,
		gw.NameContractID,
		gw.Name,
	)
	return
}

func (d *GatewayNameDeployer) syncContracts(ctx context.Context, gw *workloads.GatewayNameProxy) (err error) {
	if err := d.tfPluginClient.SubstrateConn.DeleteInvalidContracts(gw.NodeDeploymentID); err != nil {
		return err
	}
	valid, err := d.tfPluginClient.SubstrateConn.IsValidContract(gw.NameContractID)
	if err != nil {
		return err
	}
	if !valid {
		gw.NameContractID = 0
	}
	if gw.NameContractID == 0 && len(gw.NodeDeploymentID) == 0 {
		// delete resource in case nothing is active (reflects only on read)
		gw.ContractID = 0
	}
	return nil
}

// Sync syncs the gateway deployments
func (d *GatewayNameDeployer) Sync(ctx context.Context, gw *workloads.GatewayNameProxy) (err error) {
	if err := d.syncContracts(ctx, gw); err != nil {
		return errors.Wrap(err, "could not sync contracts")
	}
	dls, err := d.deployer.GetDeployments(ctx, gw.NodeDeploymentID)
	if err != nil {
		return errors.Wrap(err, "could not get deployment objects")
	}
	dl := dls[gw.NodeID]
	wl, _ := dl.Get(gridtypes.Name(gw.Name))

	gwWorkload := workloads.GatewayNameProxy{}
	gw.Backends = gwWorkload.Backends
	gw.Name = gwWorkload.Name
	gw.FQDN = gwWorkload.FQDN
	gw.TLSPassthrough = gwWorkload.TLSPassthrough
	gw.Network = gwWorkload.Network

	// if the node acknowledges it, we are golden
	if wl != nil && wl.Result.State.IsOkay() {
		gwWorkload, err := workloads.NewGatewayNameProxyFromZosWorkload(*wl.Workload)
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
