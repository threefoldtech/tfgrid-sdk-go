package app

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/convert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/generator"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/types"
	proxy_types "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// Up deploys the services described in the config file
func (a *App) Up(ctx context.Context) error {
	deploymentData, err := convert.ConvertToDeploymentData(a.Config)
	if err != nil {
		return err
	}

	err = getMissingNodes(ctx, deploymentData.NetworkNodeMap, a.Client)
	if err != nil {
		return err
	}

	networks := generateNetworks(deploymentData.NetworkNodeMap, a)

	resolvedServices, err := deploymentData.ServicesGraph.ResolveDependencies(deploymentData.ServicesGraph.Root, []*types.DRNode{}, []*types.DRNode{})

	if err != nil {
		return err
	}

	dls := make([]*workloads.Deployment, 0)
	for _, resService := range resolvedServices {
		if resService.Name == "root" {
			continue
		}

		serviceName := resService.Name
		service := deploymentData.ServicesGraph.Nodes[serviceName].Service
		var network *workloads.ZNet

		if service.Network == "" {
			network = networks[generator.GenerateDefaultNetworkName(a.Config.Services)]
			service.Network = network.Name
		} else {
			network = networks[service.Network]
		}

		vm := workloads.VM{
			Name:        serviceName,
			Flist:       service.Flist,
			Entrypoint:  service.Entrypoint,
			CPU:         int(service.Resources.CPU),
			Memory:      int(service.Resources.Memory),
			RootfsSize:  int(service.Resources.Rootfs),
			NetworkName: network.Name,
		}

		assignEnvs(&vm, service.Environment)

		disks, err := assignMounts(&vm, service.Volumes, a.Config.Volumes)
		if err != nil {
			return fmt.Errorf("failed to assign mounts %w", err)
		}

		if err := assignNetworksTypes(&vm, service.IPTypes); err != nil {
			return fmt.Errorf("failed to assign networks %w", err)
		}

		dl := &workloads.Deployment{
			Name:         a.getDeploymentName(serviceName),
			NodeID:       deploymentData.NetworkNodeMap[service.Network].NodeID,
			SolutionType: a.getProjectName(service.Network),
			NetworkName:  network.Name,
		}

		dl.Vms = append(dl.Vms, vm)
		dl.Disks = append(dl.Disks, disks...)

		dls = append(dls, dl)
	}

	log.Info().Str("status", "started").Msg("deploying networks...")

	for _, network := range networks {
		log.Info().Str("network name", network.Name).Uint32("node_id", network.Nodes[0]).Msg("deploying...")

		if err := a.Client.NetworkDeployer.Deploy(ctx, network); err != nil {
			return err
		}
	}
	log.Info().Str("status", "done").Msg("networks deployed successfully")

	deployed := make([]*workloads.Deployment, 0)
	for _, dl := range dls {
		log.Info().Str("deployment", dl.Name).Msg("deploying...")
		if err := a.Client.DeploymentDeployer.Deploy(ctx, dl); err != nil {
			log.Info().Msg("an error occurred while deploying the deployment, canceling all deployments")

			for _, network := range networks {
				if err := a.Client.NetworkDeployer.Cancel(ctx, network); err != nil {
					return err
				}
			}

			for _, deployment := range deployed {
				if err := a.Client.DeploymentDeployer.Cancel(ctx, deployment); err != nil {
					return err
				}
			}
			log.Info().Msg("all deployments canceled successfully")
			return err
		}
		log.Info().Str("deployment", dl.Name).Msg("deployed successfully")

		deployed = append(deployed, dl)
	}

	log.Info().Msg("all services deployed successfully")
	return nil
}

func generateNetworks(networkNodeMap map[string]*types.NetworkData, app *App) map[string]*workloads.ZNet {
	zNets := make(map[string]*workloads.ZNet, 0)
	defaultNetName := generator.GenerateDefaultNetworkName(app.Config.Services)

	if _, ok := networkNodeMap[defaultNetName]; ok {
		zNets[defaultNetName] = &workloads.ZNet{
			Name: defaultNetName,
			IPRange: gridtypes.NewIPNet(net.IPNet{
				IP:   net.IPv4(10, 20, 0, 0),
				Mask: net.CIDRMask(16, 32),
			}),
			AddWGAccess:  false,
			Nodes:        []uint32{networkNodeMap[defaultNetName].NodeID},
			SolutionType: app.getProjectName(defaultNetName),
		}
	}

	for networkName := range app.Config.Networks {
		network := app.Config.Networks[networkName]
		zNets[networkName] = &workloads.ZNet{
			Name:         network.Name,
			Description:  network.Description,
			IPRange:      gridtypes.NewIPNet(generator.GenerateIPNet(network.IPRange.IP, network.IPRange.Mask)),
			AddWGAccess:  network.AddWGAccess,
			MyceliumKeys: network.MyceliumKeys,
			Nodes:        []uint32{networkNodeMap[networkName].NodeID},
			SolutionType: app.getProjectName(networkName),
		}
	}

	return zNets
}

// TODO: Calculate total MRU and SRU while populating the deployment data
func getMissingNodes(ctx context.Context, networkNodeMap map[string]*types.NetworkData, client *deployer.TFPluginClient) error {
	for _, deploymentData := range networkNodeMap {
		if deploymentData.NodeID != 0 {
			continue
		}

		// freeCRU is not in NodeFilter?
		var freeMRU, freeSRU uint64

		for _, service := range deploymentData.Services {
			freeMRU += service.Resources.Memory
			freeSRU += service.Resources.Rootfs
		}

		filter := proxy_types.NodeFilter{
			Status:  []string{"up"},
			FreeSRU: &freeSRU,
			FreeMRU: &freeMRU,
		}

		nodes, _, err := client.GridProxyClient.Nodes(ctx, filter, proxy_types.Limit{})
		if err != nil {
			return err
		}

		if len(nodes) == 0 || (len(nodes) == 1 && nodes[0].NodeID == 1) {
			return fmt.Errorf("no available nodes")
		}

		// TODO: still need to agree on logic to select the node
		for _, node := range nodes {
			if node.NodeID != 1 {
				deploymentData.NodeID = uint32(node.NodeID)
				break
			}
		}
	}

	return nil
}

func assignEnvs(vm *workloads.VM, envs []string) {
	env := make(map[string]string, 0)
	for _, envVar := range envs {
		keyValuePair := strings.Split(envVar, "=")
		env[keyValuePair[0]] = keyValuePair[1]
	}

	vm.EnvVars = env
}

// TODO: Create a parser to parse the size given to each field in service
func assignMounts(vm *workloads.VM, serviceVolumes []string, volumes map[string]types.Volume) ([]workloads.Disk, error) {
	var disks []workloads.Disk
	mounts := make([]workloads.Mount, 0)
	for _, volumeName := range serviceVolumes {
		volume := volumes[volumeName]

		size, err := strconv.Atoi(strings.TrimSuffix(volume.Size, "GB"))

		if err != nil {
			return nil, err
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
	vm.Mounts = mounts

	return disks, nil
}

func assignNetworksTypes(vm *workloads.VM, ipTypes []string) error {
	for _, ipType := range ipTypes {
		switch ipType {
		case "ipv4":
			vm.PublicIP = true
		case "ipv6":
			vm.PublicIP6 = true
		case "ygg":
			vm.Planetary = true
		case "myc":
			seed, err := getRandomMyceliumIPSeed()
			if err != nil {
				return fmt.Errorf("failed to get mycelium seed %w", err)
			}
			vm.MyceliumIPSeed = seed
		}
	}

	return nil
}

func getRandomMyceliumIPSeed() ([]byte, error) {
	key := make([]byte, zos.MyceliumIPSeedLen)
	_, err := rand.Read(key)
	return key, err
}
