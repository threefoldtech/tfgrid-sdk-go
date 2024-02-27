package deployer

import (
	"context"
	"fmt"
	"net"
	"slices"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
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

	var wg sync.WaitGroup
	var errs error

	for _, dl := range dls {
		wg.Add(1)
		go func(dl *workloads.Deployment) {
			defer wg.Done()
			if err := dl.Validate(); err != nil {
				errs = multierror.Append(err)
				return
			}
		}(dl)
	}

	wg.Wait()
	return errs
}

// GenerateVersionlessDeployments generates a new deployment without a version
func (d *DeploymentDeployer) GenerateVersionlessDeployments(ctx context.Context, dls []*workloads.Deployment) (map[uint32][]gridtypes.Deployment, error) {
	gridDlsPerNodes := make(map[uint32][]gridtypes.Deployment)

	err := d.assignPrivateIPs(ctx, dls)
	if err != nil {
		return nil, errors.Wrap(err, "failed to assign node ips")
	}

	var wg sync.WaitGroup
	var lock sync.Mutex
	var errs error

	for _, dl := range dls {
		wg.Add(1)
		go func(dl *workloads.Deployment) {
			defer wg.Done()
			newDl := workloads.NewGridDeployment(d.tfPluginClient.TwinID, []gridtypes.Workload{})
			for _, disk := range dl.Disks {
				newDl.Workloads = append(newDl.Workloads, disk.ZosWorkload())
			}
			for _, zdb := range dl.Zdbs {
				newDl.Workloads = append(newDl.Workloads, zdb.ZosWorkload())
			}
			for _, vm := range dl.Vms {
				newDl.Workloads = append(newDl.Workloads, vm.ZosWorkload()...)
			}

			for idx, q := range dl.QSFS {
				qsfsWorkload, err := q.ZosWorkload()
				if err != nil {
					errs = multierror.Append(errors.Wrapf(err, "failed to generate QSFS %d in deployment '%s'", idx, dl.Name))
					return
				}
				newDl.Workloads = append(newDl.Workloads, qsfsWorkload)
			}

			newDl.Metadata, err = dl.GenerateMetadata()
			if err != nil {
				errs = multierror.Append(errors.Wrapf(err, "failed to generate deployment '%s' metadata", dl.Name))
				return
			}

			lock.Lock()
			gridDlsPerNodes[dl.NodeID] = append(gridDlsPerNodes[dl.NodeID], newDl)
			lock.Unlock()
		}(dl)
	}

	wg.Wait()

	if errs != nil {
		return nil, errs
	}
	return gridDlsPerNodes, nil
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

	if len(dlsPerNodes[dl.NodeID]) <= 0 {
		return errors.Wrap(err, "failed to generate the grid deployment")
	}

	dl.NodeDeploymentID, err = d.deployer.Deploy(
		ctx, dl.NodeDeploymentID,
		map[uint32]gridtypes.Deployment{dl.NodeID: dlsPerNodes[dl.NodeID][0]},
		map[uint32]*uint64{dl.NodeID: dl.SolutionProvider},
	)

	// update deployment and plugin state
	// error is not returned immediately before updating state because of untracked failed deployments
	if contractID, ok := dl.NodeDeploymentID[dl.NodeID]; ok && contractID != 0 {
		dl.ContractID = contractID
		d.tfPluginClient.State.StoreContractIDs(dl.NodeID, []uint64{dl.ContractID})
	}

	return err
}

// BatchDeploy deploys multiple deployments using the deployer
func (d *DeploymentDeployer) BatchDeploy(ctx context.Context, dls []*workloads.Deployment) error {
	newDeploymentsSolutionProvider := make(map[uint32][]*uint64)
	var multiErr error

	if err := d.Validate(ctx, dls); err != nil {
		return fmt.Errorf("invalid deployment: %w", err)
	}

	newDeployments, err := d.GenerateVersionlessDeployments(ctx, dls)
	if err != nil {
		return fmt.Errorf("could not generate grid deployments: %w", err)
	}

	newDls, err := d.deployer.BatchDeploy(ctx, newDeployments, newDeploymentsSolutionProvider)
	if err != nil {
		multiErr = multierror.Append(multiErr, err)
	}

	// update deployment and plugin state
	// error is not returned immediately before updating state because of untracked failed deployments
	for _, dl := range dls {
		if err := d.updateStateFromDeployments(ctx, dl, newDls); err != nil {
			return errors.Wrapf(err, "failed to update deployment '%s' state", dl.Name)
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
	d.tfPluginClient.State.RemoveContractIDs(dl.NodeID, []uint64{dl.ContractID})
	dl.ContractID = 0

	return nil
}

func (d *DeploymentDeployer) updateStateFromDeployments(ctx context.Context, dl *workloads.Deployment, newDls map[uint32][]gridtypes.Deployment) error {
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
				d.tfPluginClient.State.StoreContractIDs(dl.NodeID, []uint64{dl.ContractID})
			}
		}
	}

	return nil
}

func (d *DeploymentDeployer) calculateNetworksUsedIPs(ctx context.Context, dls []*workloads.Deployment) (map[string]map[uint32][]byte, error) {
	var wg sync.WaitGroup
	var lock sync.Mutex
	var errs error
	usedHosts := make(map[string]map[uint32][]byte)

	// calculate used host IDs per network
	for _, dl := range dls {
		lock.Lock()
		if _, ok := usedHosts[dl.NetworkName]; !ok {
			usedHosts[dl.NetworkName] = make(map[uint32][]byte)
		}

		if _, ok := usedHosts[dl.NetworkName][dl.NodeID]; ok {
			continue
		}
		lock.Unlock()

		wg.Add(1)
		go func(dl *workloads.Deployment) {
			defer wg.Done()

			nodeClient, err := d.tfPluginClient.NcPool.GetNodeClient(d.tfPluginClient.SubstrateConn, dl.NodeID)
			if err != nil {
				errs = multierror.Append(errors.Wrapf(err, "could not get node client for node %d", dl.NodeID))
				return
			}

			privateIPs, err := nodeClient.NetworkListPrivateIPs(ctx, dl.NetworkName)
			if err != nil {
				errs = multierror.Append(errors.Wrapf(err, "could not list private ips from node %d", dl.NodeID))
				return
			}

			network := d.tfPluginClient.State.Networks.GetNetwork(dl.NetworkName)
			ipRange := network.GetNodeSubnet(dl.NodeID)

			_, ipRangeCIDR, err := net.ParseCIDR(ipRange)
			if err != nil {
				errs = multierror.Append(errors.Wrapf(err, "invalid ip range %s", ipRange))
				return
			}

			// get used host IDs of the deployment node using the private IPs list of the node
			for _, privateIP := range privateIPs {
				parsedPrivateIP := net.ParseIP(privateIP).To4()

				if ipRangeCIDR.Contains(parsedPrivateIP) {
					lock.Lock()
					usedHosts[dl.NetworkName][dl.NodeID] = append(usedHosts[dl.NetworkName][dl.NodeID], parsedPrivateIP[3])
					lock.Unlock()
				}
			}
		}(dl)
	}

	wg.Wait()
	return usedHosts, errs
}

func (d *DeploymentDeployer) assignPrivateIPs(ctx context.Context, dls []*workloads.Deployment) error {
	var wg sync.WaitGroup
	var lock sync.Mutex
	var errs error

	usedHosts, err := d.calculateNetworksUsedIPs(ctx, dls)
	if err != nil {
		return errors.Wrap(err, "couldn't calculate network used ips")
	}

	for _, dl := range dls {
		if len(dl.Vms) == 0 {
			continue
		}

		wg.Add(1)
		go func(dl *workloads.Deployment) {
			defer wg.Done()
			network := d.tfPluginClient.State.Networks.GetNetwork(dl.NetworkName)
			ipRange := network.GetNodeSubnet(dl.NodeID)

			ip, ipRangeCIDR, err := net.ParseCIDR(ipRange)
			if err != nil {
				errs = multierror.Append(errors.Wrapf(err, "invalid ip range %s", ipRange))
				return
			}

			curHostID := byte(2)

			for idx, vm := range dl.Vms {
				vmIP := net.ParseIP(vm.IP).To4()

				// if vm private ip is given
				if vmIP != nil {
					vmHostID := vmIP[3] // host ID of the private ip

					// TODO: use of a duplicate IP vs an updated vm with a new/old IP
					if slices.Contains(usedHosts[dl.NetworkName][dl.NodeID], vmHostID) {
						continue
						// return fmt.Errorf("duplicate private ip '%v' in vm '%s' is used", vmIP, vm.Name)
					}

					if !ipRangeCIDR.Contains(vmIP) {
						errs = multierror.Append(fmt.Errorf("deployment ip range '%v' doesn't contain ip '%v' for vm '%s'", ipRange, vmIP, vm.Name))
						return
					}

					lock.Lock()
					usedHosts[dl.NetworkName][dl.NodeID] = append(usedHosts[dl.NetworkName][dl.NodeID], vmHostID)
					lock.Unlock()

					continue
				}

				// try to find available host ID in the deployment ip range
				for slices.Contains(usedHosts[dl.NetworkName][dl.NodeID], curHostID) {
					if curHostID == 254 {
						errs = multierror.Append(errors.New("all 253 ips of the network are exhausted"))
						return
					}
					curHostID++
				}

				lock.Lock()
				usedHosts[dl.NetworkName][dl.NodeID] = append(usedHosts[dl.NetworkName][dl.NodeID], curHostID)
				lock.Unlock()

				vmIP = ip.To4()
				vmIP[3] = curHostID
				dl.Vms[idx].IP = vmIP.String()
			}

			dl.IPrange = ipRange
		}(dl)
	}

	wg.Wait()
	return errs
}

func (d *DeploymentDeployer) syncContract(ctx context.Context, dl *workloads.Deployment) error {
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
	err := d.syncContract(ctx, dl)
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
	zdbs := make([]workloads.ZDB, 0)
	qsfs := make([]workloads.QSFS, 0)
	disks := make([]workloads.Disk, 0)

	for _, w := range deployment.Workloads {
		if !w.Result.State.IsOkay() {
			continue
		}

		switch w.Type {
		case zos.ZMachineType:
			vm, err := workloads.NewVMFromWorkload(&w, &deployment)
			if err != nil {
				log.Error().Err(err).Msgf("error parsing vm")
				continue
			}
			vms = append(vms, vm)

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
		}
	}

	dl.Match(disks, qsfs, zdbs, vms)

	dl.Disks = disks
	dl.QSFS = qsfs
	dl.Zdbs = zdbs
	dl.Vms = vms

	return nil
}
