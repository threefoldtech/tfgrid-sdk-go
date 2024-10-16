package deployer

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
)

// DeploymentDeployer for deploying a deployment
type DeploymentDeployer struct {
	tfPluginClient *TFPluginClient
	deployer       MockDeployer
}

// NewDeploymentDeployer generates a new deployer for a deployment
func NewDeploymentDeployer(tfPluginClient *TFPluginClient) DeploymentDeployer {
	deployer := NewDeployer(*tfPluginClient, true)
	return DeploymentDeployer{
		tfPluginClient: tfPluginClient,
		deployer:       &deployer,
	}
}

// Validate validates a deployment deployer
func (d *DeploymentDeployer) Validate(ctx context.Context, dls []*workloads.Deployment) error {
	if err := validateAccountBalanceForExtrinsics(d.tfPluginClient.SubstrateConn, d.tfPluginClient.Identity); err != nil {
		return err
	}

	for _, dl := range dls {
		if err := dl.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// GenerateVersionlessDeployments generates a new deployment without a version
func (d *DeploymentDeployer) GenerateVersionlessDeployments(ctx context.Context, dls []*workloads.Deployment) (map[uint32][]zos.Deployment, error) {
	gridDlsPerNodes := make(map[uint32][]zos.Deployment)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs error

	newDls, err := d.assignPrivateIPs(ctx, dls)
	if err != nil {
		errs = multierror.Append(errs, errors.Wrap(err, "failed to assign node ips"))
	}

	for _, dl := range newDls {
		wg.Add(1)
		go func(dl *workloads.Deployment) {
			defer wg.Done()
			newDl := workloads.NewGridDeployment(d.tfPluginClient.TwinID, 0, []zos.Workload{})
			for _, disk := range dl.Disks {
				newDl.Workloads = append(newDl.Workloads, disk.ZosWorkload())
			}
			for _, volume := range dl.Volumes {
				newDl.Workloads = append(newDl.Workloads, volume.ZosWorkload())
			}
			for _, zdb := range dl.Zdbs {
				newDl.Workloads = append(newDl.Workloads, zdb.ZosWorkload())
			}
			for _, vm := range dl.Vms {
				newDl.Workloads = append(newDl.Workloads, vm.ZosWorkload()...)
			}
			for _, vm := range dl.VmsLight {
				newDl.Workloads = append(newDl.Workloads, vm.ZosWorkload()...)
			}

			for idx, q := range dl.QSFS {
				qsfsWorkload, err := q.ZosWorkload()
				if err != nil {
					mu.Lock()
					defer mu.Unlock()
					errs = multierror.Append(errs, errors.Wrapf(err, "failed to generate QSFS %d in deployment '%s'", idx, dl.Name))
					return
				}
				newDl.Workloads = append(newDl.Workloads, qsfsWorkload)
			}

			mu.Lock()
			defer mu.Unlock()
			newDl.Metadata, err = dl.GenerateMetadata()
			if err != nil {
				errs = multierror.Append(errs, errors.Wrapf(err, "failed to generate deployment '%s' metadata", dl.Name))
				return
			}

			gridDlsPerNodes[dl.NodeID] = append(gridDlsPerNodes[dl.NodeID], newDl)
		}(dl)
	}

	wg.Wait()
	return gridDlsPerNodes, errs
}

// Deploy deploys a new deployment
func (d *DeploymentDeployer) Deploy(ctx context.Context, dl *workloads.Deployment) error {
	if err := d.Validate(ctx, []*workloads.Deployment{dl}); err != nil {
		return fmt.Errorf("invalid deployment: %w", err)
	}

	dlsPerNodes, err := d.GenerateVersionlessDeployments(ctx, []*workloads.Deployment{dl})
	if err != nil {
		return errors.Wrap(err, "could not generate deployments data")
	}

	if len(dlsPerNodes[dl.NodeID]) == 0 {
		return fmt.Errorf("failed to generate the grid deployment")
	}

	dl.NodeDeploymentID, err = d.deployer.Deploy(
		ctx, dl.NodeDeploymentID,
		map[uint32]zos.Deployment{dl.NodeID: dlsPerNodes[dl.NodeID][0]},
		map[uint32]*uint64{dl.NodeID: dl.SolutionProvider},
	)

	// update deployment and plugin state
	// error is not returned immediately before updating state because of untracked failed deployments
	if contractID, ok := dl.NodeDeploymentID[dl.NodeID]; ok && contractID != 0 {
		dl.ContractID = contractID
		d.tfPluginClient.State.StoreContractIDs(dl.NodeID, dl.ContractID)
	}

	return err
}

// BatchDeploy deploys multiple deployments using the deployer
func (d *DeploymentDeployer) BatchDeploy(ctx context.Context, dls []*workloads.Deployment) error {
	newDeploymentsSolutionProvider := make(map[uint32][]*uint64)
	var multiErr error

	if err := d.Validate(ctx, dls); err != nil {
		multiErr = multierror.Append(multiErr, fmt.Errorf("invalid deployments: %w", err))
	}

	newDeployments, err := d.GenerateVersionlessDeployments(ctx, dls)
	if err != nil {
		multiErr = multierror.Append(multiErr, fmt.Errorf("could not generate grid deployments: %w", err))
	}

	if len(newDeployments) == 0 {
		return errors.Wrap(multiErr, "failed to generate the grid deployments")
	}

	newDls, err := d.deployer.BatchDeploy(ctx, newDeployments, newDeploymentsSolutionProvider)
	if err != nil {
		multiErr = multierror.Append(multiErr, err)
	}

	// update deployment and plugin state
	// error is not returned immediately before updating state because of untracked failed deployments
	for _, dl := range dls {
		if err := d.updateStateFromDeployments(dl, newDls); err != nil {
			multiErr = multierror.Append(multiErr, errors.Wrapf(err, "failed to update deployment '%s' state", dl.Name))
		}
	}

	return multiErr
}

// Cancel cancels deployments
func (d *DeploymentDeployer) Cancel(ctx context.Context, dl *workloads.Deployment) error {
	if err := d.Validate(ctx, []*workloads.Deployment{dl}); err != nil {
		return err
	}

	err := d.deployer.Cancel(ctx, dl.ContractID)
	if err != nil {
		return err
	}

	// update state
	delete(dl.NodeDeploymentID, dl.NodeID)
	d.tfPluginClient.State.RemoveContractIDs(dl.NodeID, dl.ContractID)
	dl.ContractID = 0

	return nil
}

func (d *DeploymentDeployer) updateStateFromDeployments(dl *workloads.Deployment, newDls map[uint32][]zos.Deployment) error {
	dl.NodeDeploymentID = map[uint32]uint64{}

	for _, newDl := range newDls[dl.NodeID] {
		dlData, err := workloads.ParseDeploymentData(newDl.Metadata)
		if err != nil {
			return errors.Wrapf(err, "could not get deployment %d data", newDl.ContractID)
		}

		if dlData.Name == dl.Name {
			if newDl.ContractID != 0 {
				dl.NodeDeploymentID[dl.NodeID] = newDl.ContractID
				dl.ContractID = newDl.ContractID
				d.tfPluginClient.State.StoreContractIDs(dl.NodeID, dl.ContractID)
			}
		}
	}

	return nil
}

func (d *DeploymentDeployer) getUsedHostIDsOfNodeWithinNetwork(ctx context.Context, nodeID uint32, networkName string) ([]byte, error) {
	var usedHostIDs []byte

	nodeClient, err := d.tfPluginClient.NcPool.GetNodeClient(d.tfPluginClient.SubstrateConn, nodeID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get node client for node %d", nodeID)
	}

	privateIPs, err := nodeClient.NetworkListPrivateIPs(ctx, networkName)
	if err != nil {
		return nil, errors.Wrapf(err, "could not list private ips from node %d", nodeID)
	}

	network := d.tfPluginClient.State.Networks.GetNetwork(networkName)
	ipRange := network.GetNodeSubnet(nodeID)

	_, ipRangeCIDR, err := net.ParseCIDR(ipRange)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid ip range %s", ipRange)
	}

	// get used host IDs of the deployment node using the private IPs list of the node
	for _, privateIP := range privateIPs {
		parsedPrivateIP := net.ParseIP(privateIP).To4()

		if parsedPrivateIP == nil {
			log.Debug().Err(fmt.Errorf("[Error]: network %s has invalid private ip %s", networkName, privateIP)).Send()
			continue
		}

		if ipRangeCIDR.Contains(parsedPrivateIP) {
			usedHostIDs = append(usedHostIDs, parsedPrivateIP[3])
			continue
		}

		log.Debug().Err(fmt.Errorf("[Error]: network %s ip range %s doesn't contain the private ip %s found in the network", networkName, ipRange, privateIP)).Send()
	}

	return usedHostIDs, nil
}

func (d *DeploymentDeployer) calculateNetworksUsedIPs(ctx context.Context, dls []*workloads.Deployment) (map[string]map[uint32][]byte, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs error
	usedHosts := make(map[string]map[uint32][]byte)

	// calculate used host IDs per network
	for _, dl := range dls {
		if len(dl.Vms) == 0 && len(dl.VmsLight) == 0 {
			continue
		}

		mu.Lock()
		if _, ok := usedHosts[dl.NetworkName]; !ok {
			usedHosts[dl.NetworkName] = make(map[uint32][]byte)
		}

		if _, ok := usedHosts[dl.NetworkName][dl.NodeID]; ok {
			mu.Unlock()
			continue
		}
		mu.Unlock()

		wg.Add(1)
		go func(dl *workloads.Deployment) {
			defer wg.Done()
			usedHostIDs, err := d.getUsedHostIDsOfNodeWithinNetwork(ctx, dl.NodeID, dl.NetworkName)
			if err != nil {
				mu.Lock()
				defer mu.Unlock()
				errs = multierror.Append(errs, errors.Wrapf(err, "failed to get used host ids for network %s node %d", dl.NetworkName, dl.NodeID))
				return
			}

			mu.Lock()
			defer mu.Unlock()
			usedHosts[dl.NetworkName][dl.NodeID] = append(usedHosts[dl.NetworkName][dl.NodeID], usedHostIDs...)
		}(dl)
	}

	wg.Wait()
	return usedHosts, errs
}

func (d *DeploymentDeployer) assignPrivateIPs(ctx context.Context, dls []*workloads.Deployment) ([]*workloads.Deployment, error) {
	var newDls []*workloads.Deployment
	var errs error

	usedHosts, err := d.calculateNetworksUsedIPs(ctx, dls)
	if err != nil {
		errs = multierror.Append(errs, errors.Wrap(err, "couldn't calculate networks used ips"))
	}

	for _, dl := range dls {
		if len(dl.Vms) == 0 && len(dl.VmsLight) == 0 {
			newDls = append(newDls, dl)
			continue
		}

		network := d.tfPluginClient.State.Networks.GetNetwork(dl.NetworkName)
		ipRange := network.GetNodeSubnet(dl.NodeID)

		ip, ipRangeCIDR, err := net.ParseCIDR(ipRange)
		if err != nil {
			errs = multierror.Append(errs, errors.Wrapf(err, "invalid ip range %s", ipRange))
			continue
		}

		curHostID := byte(2)

		for idx, vm := range dl.Vms {
			vmIP, err := vm.AssignPrivateIP(dl.NetworkName, ipRange, dl.NodeID,
				ipRangeCIDR, ip, curHostID, usedHosts,
			)
			if err != nil {
				errs = multierror.Append(errs, errors.Wrapf(err, "failed to assign IP to vm %s", vm.Name))
				continue
			}

			dl.Vms[idx].IP = vmIP
		}

		for idx, vmLight := range dl.VmsLight {
			vmLightIP, err := vmLight.AssignPrivateIP(dl.NetworkName, ipRange, dl.NodeID,
				ipRangeCIDR, ip, curHostID, usedHosts,
			)
			if err != nil {
				errs = multierror.Append(errs, errors.Wrapf(err, "failed to assign IP to vm %s", vmLight.Name))
				continue
			}

			dl.VmsLight[idx].IP = vmLightIP
		}

		dl.IPrange = ipRange
		newDls = append(newDls, dl)
	}

	return newDls, errs
}

func (d *DeploymentDeployer) syncContract(dl *workloads.Deployment) error {
	sub := d.tfPluginClient.SubstrateConn

	if dl.ContractID == 0 {
		return nil
	}

	valid, err := sub.IsValidContract(dl.ContractID)
	if err != nil {
		return errors.Wrap(err, "error checking contract validity")
	}

	if !valid {
		dl.ContractID = 0
	}

	return nil
}

// Sync syncs the deployments // TODO: remove
func (d *DeploymentDeployer) Sync(ctx context.Context, dl *workloads.Deployment) error {
	err := d.syncContract(dl)
	if err != nil {
		return err
	}
	currentDeployments, err := d.deployer.GetDeployments(ctx, dl.NodeDeploymentID)
	if err != nil {
		return errors.Wrap(err, "failed to get deployments to update local state")
	}

	deployment := currentDeployments[dl.NodeID]

	if dl.ContractID == 0 {
		dl.Nullify()
		return nil
	}

	vms := make([]workloads.VM, 0)
	vmsLight := make([]workloads.VMLight, 0)
	zdbs := make([]workloads.ZDB, 0)
	qsfs := make([]workloads.QSFS, 0)
	disks := make([]workloads.Disk, 0)
	volumes := make([]workloads.Volume, 0)

	for _, w := range deployment.Workloads {
		if !w.Result.State.IsOkay() {
			continue
		}

		switch w.Type {
		case zos.ZMachineType:
			vm, err := workloads.NewVMFromWorkload(&w, &deployment, dl.NodeID)
			if err != nil {
				log.Error().Err(err).Msgf("error parsing vm")
				continue
			}
			vms = append(vms, vm)

		case zos.ZMachineLightType:
			vmLight, err := workloads.NewVMLightFromWorkload(&w, &deployment, dl.NodeID)
			if err != nil {
				log.Error().Err(err).Msgf("error parsing vm-light")
				continue
			}
			vmsLight = append(vmsLight, vmLight)

		case zos.ZDBType:
			zdb, err := workloads.NewZDBFromWorkload(&w)
			if err != nil {
				log.Error().Err(err).Msgf("error parsing zdb")
				continue
			}

			zdbs = append(zdbs, zdb)
		case zos.QuantumSafeFSType:
			q, err := workloads.NewQSFSFromWorkload(&w)
			if err != nil {
				log.Error().Err(err).Msgf("error parsing qsfs")
				continue
			}

			qsfs = append(qsfs, q)

		case zos.ZMountType:
			disk, err := workloads.NewDiskFromWorkload(&w)
			if err != nil {
				log.Error().Err(err).Msgf("error parsing disk")
				continue
			}

			disks = append(disks, disk)
		case zos.VolumeType:
			volume, err := workloads.NewVolumeFromWorkload(&w)
			if err != nil {
				log.Error().Err(err).Msgf("error parsing volume")
				continue
			}

			volumes = append(volumes, volume)
		}
	}

	dl.Match(disks, qsfs, zdbs, vms, vmsLight, volumes)

	dl.Disks = disks
	dl.QSFS = qsfs
	dl.Zdbs = zdbs
	dl.Vms = vms
	dl.VmsLight = vmsLight
	dl.Volumes = volumes

	return nil
}
