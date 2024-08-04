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
	types "github.com/threefoldtech/tfgrid-sdk-go/grid-compose/pkg"
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
	// deployments := make(map[string]*workloads.Deployment, 0)
	networks := generateNetworks(a.Config.Networks)
	networkDeploymentsMap := make(map[string]types.DeploymentData, 0)

	for key, val := range a.Config.Services {
		// network := workloads.ZNet{
		// 	Name:  networkName,
		// 	Nodes: []uint32{val.NodeID},
		// 	IPRange: gridtypes.NewIPNet(net.IPNet{
		// 		IP:   net.IPv4(10, 20, 0, 0),
		// 		Mask: net.CIDRMask(16, 32),
		// 	}),
		// 	SolutionType: projectName,
		// }

		var network *workloads.ZNet
		if val.Networks == nil || len(val.Networks) == 0 {
			network = &workloads.ZNet{
				Name:  key + "net",
				Nodes: []uint32{val.NodeID},
				IPRange: gridtypes.NewIPNet(net.IPNet{
					IP:   net.IPv4(10, 20, 0, 0),
					Mask: net.CIDRMask(16, 32),
				}),
			}
		} else {
			network = networks[val.Networks[0]]
		}
		network.SolutionType = a.Config.ProjectName

		vm := workloads.VM{
			Name:        key,
			Flist:       val.Flist,
			Entrypoint:  val.Entrypoint,
			CPU:         int(val.Resources.CPU),
			Memory:      int(val.Resources.Memory),
			NetworkName: network.Name,
		}

		assignEnvs(&vm, val.Environment)

		disks, err := assignMounts(&vm, val.Volumes, a.Config.Storage)
		if err != nil {
			return fmt.Errorf("failed to assign mounts %w", err)
		}

		if err := assignNetworksTypes(&vm, val.NetworkTypes); err != nil {
			return fmt.Errorf("failed to assign networks %w", err)
		}

		// if err := a.Client.NetworkDeployer.Deploy(context.Background(), &network); err != nil {
		// 	return err
		// }

		// dl := workloads.NewDeployment(vm.Name, uint32(val.NodeID), projectName, nil, network.Name, disks, nil, nil, nil)
		deploymentData := networkDeploymentsMap[network.Name]

		deploymentData.Vms = append(deploymentData.Vms, vm)
		deploymentData.Disks = append(deploymentData.Disks, disks...)
		if !checkIfNodeIDExist(val.NodeID, deploymentData.NodeIDs) {
			deploymentData.NodeIDs = append(deploymentData.NodeIDs, val.NodeID)
		}
		networkDeploymentsMap[network.Name] = deploymentData
	}

	log.Info().Str("status", "started").Msg("deploying networks...")
	for _, val := range networks {
		if err := a.Client.NetworkDeployer.Deploy(ctx, val); err != nil {
			return err
		}
	}
	log.Info().Str("status", "done").Msg("networks deployed successfully")

	for key, val := range networkDeploymentsMap {
		for _, nodeID := range val.NodeIDs {
			dlName := a.getDeploymentName()
			log.Info().Str("deployment", dlName).Str("services", fmt.Sprintf("%v", val.Vms)).Msg("deploying...")

			dl := workloads.NewDeployment(dlName, nodeID, a.Config.ProjectName, nil, key, val.Disks, nil, val.Vms, nil)
			if err := a.Client.DeploymentDeployer.Deploy(ctx, &dl); err != nil {
				for _, val := range networks {
					if err := a.Client.NetworkDeployer.Cancel(ctx, val); err != nil {
						return err
					}
				}
				return err
			}

			log.Info().Str("deployment", dlName).Msg("deployed successfully")
		}
	}

	log.Info().Msg("all services deployed successfully")
	return nil
}

func checkIfNodeIDExist(nodeID uint32, nodes []uint32) bool {
	for _, node := range nodes {
		if node == nodeID {
			return true
		}
	}

	return false
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
		if err := a.loadCurrentNodeDeplyments(a.Config.ProjectName); err != nil {
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

			var wlData types.WorkloadData
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

	projectName := a.Config.ProjectName
	log.Info().Str("projectName", projectName).Msg("canceling deployments")
	if err := a.Client.CancelByProjectName(projectName); err != nil {
		return err
	}

	return nil
}

func (a *App) getProjectName(key string) string {
	key = strings.TrimSuffix(key, "net")
	return fmt.Sprintf("compose/%v/%v", a.Client.TwinID, key)
}

func (a *App) getDeploymentName() string {
	return fmt.Sprintf("dl_%v_%v", a.Client.TwinID, generateRandString(5))
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
