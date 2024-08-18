package deploy

import (
	"context"
	"crypto/rand"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/types"
	proxy_types "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func DeployVM(ctx context.Context, t *deployer.TFPluginClient, vm workloads.VM, disks []workloads.Disk, network *workloads.ZNet, dlName string, healthChecks ...types.HealthCheck) (workloads.Deployment, error) {
	log.Info().Str("name", network.Name).Uint32("node_id", network.Nodes[0]).Msg("deploying network...")
	if err := t.NetworkDeployer.Deploy(ctx, network); err != nil {
		return workloads.Deployment{}, err
	}
	log.Info().Msg("deployed successfully")

	dl := workloads.NewDeployment(dlName, network.Nodes[0], network.SolutionType, nil, network.Name, disks, nil, []workloads.VM{vm}, nil)
	log.Info().Str("name", vm.Name).Uint32("node_id", dl.NodeID).Msg("deploying vm...")
	if err := t.DeploymentDeployer.Deploy(ctx, &dl); err != nil {
		return workloads.Deployment{}, err
	}
	log.Info().Msg("deployed successfully")

	resDl, err := t.State.LoadDeploymentFromGrid(ctx, dl.NodeID, dl.Name)
	if err != nil {
		return workloads.Deployment{}, errors.Wrapf(err, "failed to load vm from node %d", dl.NodeID)
	}

	if len(healthChecks) > 0 {
		log.Info().Msg("running health checks...")
		for _, hc := range healthChecks {
			log.Info().Str("addr", strings.Split(resDl.Vms[0].ComputedIP, "/")[0]).Msg("")
			if err := runHealthCheck(hc, "/home/eyad/Downloads/temp/id_rsa", "root", strings.Split(resDl.Vms[0].ComputedIP, "/")[0]); err != nil {
				return resDl, err
			}
		}
	}

	return resDl, nil
}

func BuildVM(vm *workloads.VM, service *types.Service) error {
	assignEnvs(vm, service.Environment)
	if err := assignNetworksTypes(vm, service.IPTypes); err != nil {
		return err
	}
	return nil
}

// TODO: Create a parser to parse the size given to each field in service
func BuildDisks(vm *workloads.VM, serviceVolumes []string, volumes map[string]types.Volume) ([]workloads.Disk, error) {
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
		key, value, _ := strings.Cut(envVar, "=")
		env[key] = value
	}

	vm.EnvVars = env
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
