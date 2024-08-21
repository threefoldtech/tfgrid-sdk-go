package deploy

import (
	"context"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/types"
)

// DeployVM deploys a vm on the grid
func DeployVM(ctx context.Context, client *deployer.TFPluginClient, vm workloads.VM, disks []workloads.Disk, network *workloads.ZNet, dlName string, healthCheck *types.HealthCheck) (workloads.Deployment, error) {
	dl := workloads.NewDeployment(dlName, network.Nodes[0], network.SolutionType, nil, network.Name, disks, nil, []workloads.VM{vm}, nil)
	log.Info().Str("name", vm.Name).Uint32("node_id", dl.NodeID).Msg("deploying vm...")
	if err := client.DeploymentDeployer.Deploy(ctx, &dl); err != nil {
		return workloads.Deployment{}, err
	}
	log.Info().Msg("deployed successfully")

	resDl, err := client.State.LoadDeploymentFromGrid(ctx, dl.NodeID, dl.Name)
	if err != nil {
		return workloads.Deployment{}, errors.Wrapf(err, "failed to load vm from node %d", dl.NodeID)
	}

	if healthCheck != nil {
		log.Info().Msg("running health check...")

		log.Info().Str("addr", strings.Split(resDl.Vms[0].ComputedIP, "/")[0]).Msg("")
		if err := runHealthCheck(*healthCheck, "root", strings.Split(resDl.Vms[0].ComputedIP, "/")[0]); err != nil {
			return resDl, err
		}

	}

	return resDl, nil
}

// BuildStorage converts the config volumes to disks and mounts and returns them.
// TODO: Create a parser to parse the size given to each field in service
func BuildStorage(serviceVolumes []string, volumes map[string]types.Volume) ([]workloads.Disk, []workloads.Mount, error) {
	var disks []workloads.Disk
	mounts := make([]workloads.Mount, 0)
	for _, volumeName := range serviceVolumes {
		volume := volumes[volumeName]

		size, err := strconv.Atoi(strings.TrimSuffix(volume.Size, "GB"))

		if err != nil {
			return nil, nil, err
		}

		disk := workloads.Disk{
			Name:   volumeName,
			SizeGB: size,
		}

		disks = append(disks, disk)

		mounts = append(mounts, workloads.Mount{
			DiskName:   disk.Name,
			MountPoint: volume.MountPoint,
		})
	}

	return disks, mounts, nil
}
