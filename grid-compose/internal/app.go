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
	networks := a.generateNetworks()
	deployments := a.generateInitDeployments()

	for key, val := range a.Config.Services {
		deployment := deployments[val.DeployTo]

		var network *workloads.ZNet
		if deployment.Name == "" {
			network = &workloads.ZNet{
				Name:  deployment.Name + "net",
				Nodes: []uint32{deployment.NodeID},
				IPRange: gridtypes.NewIPNet(net.IPNet{
					IP:   net.IPv4(10, 20, 0, 0),
					Mask: net.CIDRMask(16, 32),
				}),
			}
			deployment.NetworkName = network.Name
		} else {
			network = networks[a.Config.Deployments[val.DeployTo].Network.Name]
		}

		network.SolutionType = a.getProjectName(deployment.Name)

		vm := workloads.VM{
			Name:        key,
			Flist:       val.Flist,
			Entrypoint:  val.Entrypoint,
			CPU:         int(val.Resources.CPU),
			Memory:      int(val.Resources.Memory),
			NetworkName: network.Name,
			RootfsSize:  int(val.Resources.Rootfs),
		}

		assignEnvs(&vm, val.Environment)

		disks, err := assignMounts(&vm, val.Volumes, a.Config.Storage)
		if err != nil {
			return fmt.Errorf("failed to assign mounts %w", err)
		}

		if err := assignNetworksTypes(&vm, val.NetworkTypes); err != nil {
			return fmt.Errorf("failed to assign networks %w", err)
		}

		deployment.Vms = append(deployment.Vms, vm)
		deployment.Disks = append(deployment.Disks, disks...)
	}

	log.Info().Str("status", "started").Msg("deploying networks...")
	for _, val := range networks {
		if err := a.Client.NetworkDeployer.Deploy(ctx, val); err != nil {
			return err
		}
	}
	log.Info().Str("status", "done").Msg("networks deployed successfully")

	for key, val := range deployments {
		log.Info().Str("deployment", key).Msg("deploying...")

		if err := a.Client.DeploymentDeployer.Deploy(ctx, val); err != nil {
			for _, val := range networks {
				if err := a.Client.NetworkDeployer.Cancel(ctx, val); err != nil {
					return err
				}
			}
			return err
		}
		log.Info().Str("deployment", key).Msg("deployed successfully")
	}

	log.Info().Msg("all services deployed successfully")
	return nil
}

func (a *App) Ps(ctx context.Context, flags *pflag.FlagSet) error {
	verbose, outputFile, err := parsePsFlags(flags)
	if err != nil {
		return err
	}

	var output strings.Builder
	if !verbose {
		output.WriteString(fmt.Sprintf("%-15s | %-15s | %-15s | %-15s | %-10s | %s\n", "Deployment Name", "Network", "Service Name", "Storage", "State", "IP Address"))
		output.WriteString(strings.Repeat("-", 79) + "\n")
	}

	outputMap := make(map[string]struct {
		Deployment types.Deployment
		Workloads  []struct {
			Workload     gridtypes.Workload
			WorkloadData types.WorkloadData
			Addresses    string
		}
	})

	for _, deployment := range a.Config.Deployments {
		if err := a.loadCurrentNodeDeployments(a.getProjectName(deployment.Name)); err != nil {
			return err
		}

		outputMap[deployment.Name] = struct {
			Deployment types.Deployment
			Workloads  []struct {
				Workload     gridtypes.Workload
				WorkloadData types.WorkloadData
				Addresses    string
			}
		}{
			Deployment: deployment,
			Workloads: []struct {
				Workload     gridtypes.Workload
				WorkloadData types.WorkloadData
				Addresses    string
			}{},
		}

		for _, workloadName := range deployment.Workloads {
			wlStruct := struct {
				Workload     gridtypes.Workload
				WorkloadData types.WorkloadData
				Addresses    string
			}{
				Workload:     gridtypes.Workload{},
				WorkloadData: types.WorkloadData{},
			}

			wl, dl, err := a.Client.State.GetWorkloadInDeployment(ctx, deployment.NodeID, workloadName, deployment.Name)
			if err != nil {
				return err
			}

			wlStruct.Workload = wl

			if wl.Type == "zmachine" {
				err = json.Unmarshal(wl.Data, &wlStruct.WorkloadData)
				if err != nil {
					return err
				}
			}

			var wlData types.WorkloadData
			err = json.Unmarshal(wl.Data, &wlData)
			if err != nil {
				return err
			}

			vm, err := workloads.NewVMFromWorkload(&wl, &dl)
			if err != nil {
				return err
			}

			addresses := getVmAddresses(vm)

			wlStruct.Addresses = addresses

			deploymentEntry := outputMap[deployment.Name]
			deploymentEntry.Workloads = append(deploymentEntry.Workloads, wlStruct)
			outputMap[deployment.Name] = deploymentEntry
		}
	}

	for key, val := range outputMap {
		fmt.Printf("%+v\n", val)
		output.WriteString(fmt.Sprintf("%-15s | %-15s | ", key, a.Config.Networks[val.Deployment.Network.Name].Name))

		for _, wl := range val.Workloads {
			output.WriteString(fmt.Sprintf("%-15s | %-15s | %-10s | %s \n", wl.Workload.Name, wl.WorkloadData.Mounts[0].Name, wl.Workload.Result.State, wl.Addresses))
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
	for _, val := range a.Config.Deployments {
		projectName := a.getProjectName(val.Name)
		log.Info().Str("projectName", projectName).Msg("canceling deployments")
		if err := a.Client.CancelByProjectName(projectName); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) generateNetworks() map[string]*workloads.ZNet {
	zNets := make(map[string]*workloads.ZNet, 0)

	for key, network := range a.Config.Networks {
		zNet := workloads.ZNet{
			Name:         network.Name,
			Description:  network.Description,
			Nodes:        network.Nodes,
			IPRange:      gridtypes.NewIPNet(generateIPNet(network.IPRange.IP, network.IPRange.Mask)),
			AddWGAccess:  network.AddWGAccess,
			MyceliumKeys: network.MyceliumKeys,
		}

		zNets[key] = &zNet
	}

	return zNets
}

func (a *App) generateInitDeployments() map[string]*workloads.Deployment {
	workloadsDeployments := make(map[string]*workloads.Deployment, 0)

	for key, deployment := range a.Config.Deployments {
		var networkName string
		if deployment.Network != nil {
			networkName = a.Config.Networks[deployment.Network.Name].Name
		}
		workloadsDeployment := workloads.NewDeployment(deployment.Name, deployment.NodeID, a.getProjectName(deployment.Name), nil, networkName, make([]workloads.Disk, 0), nil, make([]workloads.VM, 0), nil)
		workloadsDeployments[key] = &workloadsDeployment
	}

	return workloadsDeployments
}

func (a *App) getProjectName(key string) string {
	return fmt.Sprintf("compose/%v/%v", a.Client.TwinID, key)
}

func (a *App) loadCurrentNodeDeployments(projectName string) error {
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
