package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/config"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
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

	config := config.NewConfig()
	err = config.LoadConfigFromReader(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from file %w", err)
	}

	if err := config.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("failed to validate config %w", err)
	}

	client, err := deployer.NewTFPluginClient(mnemonic, deployer.WithNetwork(net))
	if err != nil {
		return nil, fmt.Errorf("failed to load grid client %w", err)
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

		assignEnvs(&vm, val.Environment)

		disks, err := assignMounts(&vm, val.Volumes, a.Config.Storage)
		if err != nil {
			return fmt.Errorf("failed to assign mounts %w", err)
		}

		if err := assignNetworks(&vm, val.Networks, a.Config.Networks, &network); err != nil {
			return fmt.Errorf("failed to assign networks %w", err)
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

func (a *App) Ps(ctx context.Context, flags *pflag.FlagSet) error {
	verbose, outputFile, err := parsePsFlags(flags)
	if err != nil {
		return err
	}

	var output strings.Builder
	if !verbose {
		output.WriteString(fmt.Sprintf("%-15s | %-15s | %-15s | %-10s | %s\n", "Name", "Network", "Storage", "State", "IP Address"))
		output.WriteString(strings.Repeat("-", 79) + "\n")
	}
	for key, val := range a.Config.Services {
		if err := a.loadCurrentNodeDeplyments(a.getProjectName(key, a.Client.TwinID)); err != nil {
			return err
		}

		wl, dl, err := a.Client.State.GetWorkloadInDeployment(ctx, uint32(val.NodeID), key, key)
		if err != nil {
			return err
		}

		vm, err := workloads.NewVMFromWorkload(&wl, &dl)
		if err != nil {
			return err
		}

		addresses := getVmAddresses(vm)

		s, err := json.MarshalIndent(dl, "", "  ")
		if err != nil {
			return err
		}

		if verbose {
			if outputFile != "" {
				output.WriteString(fmt.Sprintf("\"%s\": %s,\n", key, string(s)))
			} else {
				output.WriteString(fmt.Sprintf("deplyment: %s\n%s\n\n\n", key, string(s)))
			}
		} else {
			var wl gridtypes.Workload

			for _, workload := range dl.Workloads {
				if workload.Type == "zmachine" {
					wl = workload
					break
				}
			}

			var wlData pkg.WorkloadData
			err = json.Unmarshal(wl.Data, &wlData)
			if err != nil {
				return err
			}

			output.WriteString(fmt.Sprintf("%-15s | %-15s | %-15s | %-10s | %s \n", wl.Name, wlData.Network.Interfaces[0].Network, wlData.Mounts[0].Name, wl.Result.State, addresses))
		}
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(fmt.Sprintf("{\n%s\n}", strings.TrimSuffix(output.String(), ",\n"))), 0644); err != nil {
			return err
		}
		return nil
	} else {
		// for better formatting
		println("\n" + output.String())
	}

	return nil
}

func parsePsFlags(flags *pflag.FlagSet) (bool, string, error) {
	verbose, err := flags.GetBool("verbose")
	if err != nil {
		return verbose, "", err
	}

	outputFile, err := flags.GetString("output")
	if err != nil {
		return verbose, outputFile, err
	}

	return verbose, outputFile, nil
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
	key = strings.TrimSuffix(key, "net")
	return fmt.Sprintf("compose/%v/%v", twinId, key)
}

func (a *App) loadCurrentNodeDeplyments(projectName string) error {
	contracts, err := a.Client.ContractsGetter.ListContractsOfProjectName(projectName, true)
	if err != nil {
		return err
	}

	var nodeID uint32

	for _, contract := range contracts.NodeContracts {
		contractID, err := strconv.ParseUint(contract.ContractID, 10, 64)
		if err != nil {
			return err
		}

		nodeID = contract.NodeID
		a.checkIfExistAndAppend(nodeID, contractID)
	}

	return nil
}

func (a *App) checkIfExistAndAppend(node uint32, contractID uint64) {
	for _, n := range a.Client.State.CurrentNodeDeployments[node] {
		if n == contractID {
			return
		}
	}

	a.Client.State.CurrentNodeDeployments[node] = append(a.Client.State.CurrentNodeDeployments[node], contractID)
}
