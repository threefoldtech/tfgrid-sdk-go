package app

import (
	"context"
	"net"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// deployVM deploys a vm on the grid
func deployVM(ctx context.Context, client *deployer.TFPluginClient, vm workloads.VM, disks []workloads.Disk, network *workloads.ZNet, dlName string, healthCheck *types.HealthCheck) (workloads.Deployment, error) {
	// volumes is nil until it is clear what it is used for
	dl := workloads.NewDeployment(dlName, network.Nodes[0], network.SolutionType, nil, network.Name, disks, nil, []workloads.VM{vm}, nil, nil)
	if err := dl.Validate(); err != nil {
		return workloads.Deployment{}, err
	}

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

// buildStorage converts the config volumes to disks and mounts and returns them.
func buildStorage(serviceVolumes []string, volumes map[string]types.Volume) ([]workloads.Disk, []workloads.Mount, error) {
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

// buildNetworks converts the networks in the config to ZNet workloads.
// TODO: needs to be refactored
func buildNetworks(networkNodeMap map[string]*types.NetworkData, networks map[string]types.Network, defaultNetName string, getProjectName func(string) string) map[string]*workloads.ZNet {
	zNets := make(map[string]*workloads.ZNet, 0)
	if _, ok := networkNodeMap[defaultNetName]; ok {
		zNets[defaultNetName] = &workloads.ZNet{
			Name: defaultNetName,
			IPRange: gridtypes.NewIPNet(net.IPNet{
				IP:   net.IPv4(10, 20, 0, 0),
				Mask: net.CIDRMask(16, 32),
			}),
			AddWGAccess:  false,
			Nodes:        []uint32{networkNodeMap[defaultNetName].NodeID},
			SolutionType: getProjectName(defaultNetName),
		}
	}

	for networkName, network := range networks {
		zNets[networkName] = &workloads.ZNet{
			Name:         network.Name,
			Description:  network.Description,
			IPRange:      gridtypes.NewIPNet(generateIPNet(network.IPRange.IP, network.IPRange.Mask)),
			AddWGAccess:  network.AddWGAccess,
			MyceliumKeys: network.MyceliumKeys,
			Nodes:        []uint32{networkNodeMap[networkName].NodeID},
			SolutionType: getProjectName(networkName),
		}

	}

	return zNets
}

// generateIPNet generates a net.IPNet from the given IP and mask.
func generateIPNet(ip types.IP, mask types.IPMask) net.IPNet {
	var ipNet net.IPNet

	switch ip.Type {
	case "ipv4":
		ipSplit := strings.Split(ip.IP, ".")
		byte1, _ := strconv.ParseUint(ipSplit[0], 10, 8)
		byte2, _ := strconv.ParseUint(ipSplit[1], 10, 8)
		byte3, _ := strconv.ParseUint(ipSplit[2], 10, 8)
		byte4, _ := strconv.ParseUint(ipSplit[3], 10, 8)

		ipNet.IP = net.IPv4(byte(byte1), byte(byte2), byte(byte3), byte(byte4))
	default:
		return ipNet
	}

	var maskIP net.IPMask

	switch mask.Type {
	case "cidr":
		maskSplit := strings.Split(mask.Mask, "/")
		maskOnes, _ := strconv.ParseInt(maskSplit[0], 10, 8)
		maskBits, _ := strconv.ParseInt(maskSplit[1], 10, 8)

		maskIP = net.CIDRMask(int(maskOnes), int(maskBits))
		ipNet.Mask = maskIP
	default:
		return ipNet
	}

	return ipNet
}

// assignMyCeliumKeys assigns mycelium keys to the network nodes.
func assignMyCeliumKeys(network *workloads.ZNet, myceliumIPSeed []byte) error {
	keys := make(map[uint32][]byte)
	if len(myceliumIPSeed) != 0 {
		for _, node := range network.Nodes {
			key, err := workloads.RandomMyceliumKey()
			if err != nil {
				return err
			}
			keys[node] = key
		}
	}

	network.MyceliumKeys = keys
	return nil
}

func rollback(ctx context.Context, client *deployer.TFPluginClient, deployedDls []*workloads.Deployment, deployedNets map[string]*workloads.ZNet, err error) error {
	log.Info().Msg("an error occurred while deploying, canceling all deployments")
	log.Info().Msg("canceling networks...")
	for _, network := range deployedNets {
		if err := client.NetworkDeployer.Cancel(ctx, network); err != nil {
			return err
		}
	}

	log.Info().Msg("canceling deployments...")
	for _, deployment := range deployedDls {
		if err := client.DeploymentDeployer.Cancel(ctx, deployment); err != nil {
			return err
		}
	}

	log.Info().Msg("all deployments canceled successfully")
	return err
}
