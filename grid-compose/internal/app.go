package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/config"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"gopkg.in/yaml.v2"
)

type App struct {
	Client *deployer.TFPluginClient
	Config *config.Config
}

func NewApp(net, mnemonic, configPath string) (*App, error) {
	configFile, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}

	defer configFile.Close()

	config, err := loadConfigFromReader(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from file: %w", err)
	}

	if err := config.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("failed to validate config %w", err)
	}

	client, err := deployer.NewTFPluginClient(mnemonic, deployer.WithNetwork(net))
	if err != nil {
		return nil, fmt.Errorf("failed to load grid client: %w", err)
	}

	return &App{
		Config: config,
		Client: &client,
	}, nil
}

func (a *App) Up(ctx context.Context) error {
	deployments := make(map[string]*workloads.Deployment, 0)

	for key, val := range a.Config.Services {
		projectName := a.getProjectName(key, a.Client.TwinID)
		networkName := key + "net"

		network := workloads.ZNet{
			Name:  networkName,
			Nodes: []uint32{uint32(val.NodeID)},
			IPRange: gridtypes.NewIPNet(net.IPNet{
				IP:   net.IPv4(10, 20, 0, 0),
				Mask: net.CIDRMask(16, 32),
			}),
			SolutionType: projectName,
		}

		vm := workloads.VM{
			Name:        key,
			Flist:       val.Flist,
			Entrypoint:  val.Entrypoint,
			CPU:         int(val.Resources.CPU),
			Memory:      int(val.Resources.Memory),
			NetworkName: networkName,
		}

		env := make(map[string]string, 0)

		for _, envVar := range val.Environment {
			split := strings.Split(envVar, "=")
			env[split[0]] = split[1]
		}

		vm.EnvVars = env

		var mounts []workloads.Mount
		var disks []workloads.Disk
		for _, volume := range val.Volumes {
			split := strings.Split(volume, ":")

			storage := a.Config.Storage[split[0]]
			size, _ := strconv.Atoi(strings.TrimSuffix(storage.Size, "GB"))
			disk := workloads.Disk{
				Name:   split[0],
				SizeGB: size,
			}

			disks = append(disks, disk)

			mounts = append(mounts, workloads.Mount{
				DiskName:   disk.Name,
				MountPoint: split[1],
			})
		}
		vm.Mounts = mounts

		for _, net := range val.Networks {
			switch a.Config.Networks[net].Type {
			case "wg":
				network.AddWGAccess = true
			case "ip4":
				vm.PublicIP = true
			case "ip6":
				vm.PublicIP6 = true
			case "ygg":
				vm.Planetary = true
			case "myc":
				seed, err := getRandomMyceliumIPSeed()
				if err != nil {
					return fmt.Errorf("failed to get mycelium seed: %w", err)
				}
				vm.MyceliumIPSeed = seed
			}
		}
		if err := a.Client.NetworkDeployer.Deploy(context.Background(), &network); err != nil {
			return err
		}

		dl := workloads.NewDeployment(vm.Name, uint32(val.NodeID), projectName, nil, networkName, disks, nil, []workloads.VM{vm}, nil)
		if err := a.Client.DeploymentDeployer.Deploy(context.Background(), &dl); err != nil {
			if err := a.Client.NetworkDeployer.Cancel(context.Background(), &network); err != nil {
				return err
			}
			return err
		}

		deployments[dl.Name] = &dl
	}

	for name, dl := range deployments {
		vmState, err := a.Client.State.LoadVMFromGrid(ctx, uint32(dl.NodeID), name, name)
		if err != nil {
			return fmt.Errorf("%w vm %s", err, name)
		}

		log.Info().Str("ip", vmState.IP).Str("vm name", name).Msg("vm deployed")
	}

	return nil
}

func (a *App) Ps(ctx context.Context) error {
	twindId := uint64(a.Client.TwinID)
	filters := types.ContractFilter{
		TwinID: &twindId,
	}
	limits := types.Limit{
		Size: 100,
	}
	cache := make(map[string]bool, 0)

	contracts, _, err := a.Client.GridProxyClient.Contracts(ctx, filters, limits)

	if err != nil {
		return err
	}

	for _, contract := range contracts {
		if contract.Type != "node" || contract.State == "Deleted" {
			continue
		}

		details, err := workloads.ParseDeploymentData(contract.Details.(types.NodeContractDetails).DeploymentData)
		if err != nil {
			return err
		}

		if strings.Split(details.ProjectName, "/")[0] != "compose" || cache[details.ProjectName] {
			continue
		}

		res, err := GetVM(ctx, a.Client, details.Name)
		if err != nil {
			return err
		}

		s, err := json.MarshalIndent(res, "", "\t")
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		log.Info().Msg("vm:\n" + string(s))
		cache[details.ProjectName] = true
	}

	return nil
}

func (a *App) Down() error {
	for key := range a.Config.Services {
		projectName := a.getProjectName(key, a.Client.TwinID)
		log.Info().Str("projectName", projectName).Msg("canceling deployments")
		if err := a.Client.CancelByProjectName(projectName); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) getProjectName(key string, twinId uint32) string {
	return fmt.Sprintf("compose/%v/%v", twinId, key)
}

func loadConfigFromReader(configFile io.Reader) (*config.Config, error) {
	content, err := io.ReadAll(configFile)
	if err != nil {
		return &config.Config{}, fmt.Errorf("failed to read file: %w", err)
	}

	var config config.Config
	if err := yaml.Unmarshal(content, &config); err != nil {
		return &config, fmt.Errorf("failed to parse file: %w", err)
	}

	return &config, nil
}
