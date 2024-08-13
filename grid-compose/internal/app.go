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
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/config"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/utils"
)

// App is the main application struct that holds the client and the config data
type App struct {
	Client *deployer.TFPluginClient
	Config *config.Config
}

// NewApp creates a new instance of the application
func NewApp(net, mnemonic, configPath string) (*App, error) {
	if !utils.ValidateCredentials(mnemonic, net) {
		return nil, fmt.Errorf("invalid mnemonic or network")
	}

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

// Up deploys the services described in the config file
func (a *App) Up(ctx context.Context) error {
	err := a.generateMissingNodes(ctx)
	if err != nil {
		return err
	}

	networks := a.generateNetworks()
	dls := []*workloads.Deployment{}

	for networkName, deploymentData := range a.Config.DeploymentData {
		network := networks[networkName]
		projectName := a.getProjectName(networkName)

		network.SolutionType = projectName
		network.Nodes = []uint32{deploymentData.NodeID}
		dl := &workloads.Deployment{
			Name:         a.getDeploymentName(networkName),
			NodeID:       deploymentData.NodeID,
			SolutionType: projectName,
			NetworkName:  network.Name,
		}

		for _, service := range deploymentData.Services {
			vm := workloads.VM{
				Name:        service.Name,
				Flist:       service.Flist,
				Entrypoint:  service.Entrypoint,
				CPU:         int(service.Resources.CPU),
				Memory:      int(service.Resources.Memory),
				RootfsSize:  int(service.Resources.Rootfs),
				NetworkName: network.Name,
			}

			utils.AssignEnvs(&vm, service.Environment)

			disks, err := utils.AssignMounts(&vm, service.Volumes, a.Config.Volumes)
			if err != nil {
				return fmt.Errorf("failed to assign mounts %w", err)
			}

			if err := utils.AssignNetworksTypes(&vm, service.IPTypes); err != nil {
				return fmt.Errorf("failed to assign networks %w", err)
			}

			dl.Vms = append(dl.Vms, vm)
			dl.Disks = append(dl.Disks, disks...)
		}

		dls = append(dls, dl)
	}

	log.Info().Str("status", "started").Msg("deploying networks...")

	for _, network := range networks {
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

// Ps lists the deployed services
func (a *App) Ps(ctx context.Context, flags *pflag.FlagSet) error {
	verbose, outputFile, err := parsePsFlags(flags)
	if err != nil {
		return err
	}

	var output strings.Builder
	outputMap := make(map[string]gridtypes.Deployment)

	if !verbose {
		output.WriteString(fmt.Sprintf("%-15s | %-15s | %-15s | %-15s | %-10s | %s\n", "Deployment Name", "Network", "Services", "Storage", "State", "IP Address"))
		output.WriteString(strings.Repeat("-", 100) + "\n")
	}

	for networkName, deploymentData := range a.Config.DeploymentData {
		projectName := a.getProjectName(networkName)

		if err := a.loadCurrentNodeDeployments(projectName); err != nil {
			return err
		}

		contracts, err := a.Client.ContractsGetter.ListContractsOfProjectName(projectName)
		if err != nil {
			return err
		}

		for _, contract := range contracts.NodeContracts {
			contractDlData, err := workloads.ParseDeploymentData(contract.DeploymentData)
			if err != nil {
				return err
			}

			if contractDlData.Type == "network" {
				continue
			}

			dlAdded := false
			for _, service := range deploymentData.Services {
				wl, dl, err := a.Client.State.GetWorkloadInDeployment(ctx, contract.NodeID, service.Name, contractDlData.Name)
				if err != nil {
					return err
				}

				vm, err := workloads.NewVMFromWorkload(&wl, &dl)
				if err != nil {
					return err
				}

				if !verbose {
					if !dlAdded {
						output.WriteString(fmt.Sprintf("%-15s | %-15s | %-15s | %-15s | %-10s | %s\n", contractDlData.Name, vm.NetworkName, vm.Name, vm.Mounts[0].DiskName, wl.Result.State, utils.GetVmAddresses(vm)))
						dlAdded = true
					} else {
						output.WriteString(fmt.Sprintf("%-15s | %-15s | %-15s | %-15s | %-10s | %s\n", strings.Repeat("-", 15), strings.Repeat("-", 15), vm.Name, vm.Mounts[0].DiskName, wl.Result.State, utils.GetVmAddresses(vm)))
					}
				}
				outputMap[contractDlData.Name] = dl
			}
		}
	}

	if verbose {
		out, err := json.MarshalIndent(outputMap, "", "  ")
		if err != nil {
			return err
		}
		if outputFile == "" {
			fmt.Println(string(out))
			return nil
		}

		if err := os.WriteFile(outputFile, out, 0644); err != nil {
			return err
		}

		return nil
	}

	// print for better formatting
	fmt.Printf("\n%s\n", output.String())
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

// Down cancels all the deployments
func (a *App) Down() error {
	for networkName := range a.Config.DeploymentData {
		projectName := a.getProjectName(networkName)
		log.Info().Str("projectName", projectName).Msg("canceling deployments")
		if err := a.Client.CancelByProjectName(projectName); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) generateNetworks() map[string]*workloads.ZNet {
	zNets := make(map[string]*workloads.ZNet, 0)
	defNetName := utils.GenerateDefaultNetworkName(a.Config.Services)
	for networkName := range a.Config.DeploymentData {
		if networkName == defNetName {
			zNets[networkName] = &workloads.ZNet{
				Name: networkName,
				IPRange: gridtypes.NewIPNet(net.IPNet{
					IP:   net.IPv4(10, 20, 0, 0),
					Mask: net.CIDRMask(16, 32),
				}),
				AddWGAccess: false,
			}
		} else {
			network := a.Config.Networks[networkName]
			zNets[networkName] = &workloads.ZNet{
				Name:         network.Name,
				Description:  network.Description,
				IPRange:      gridtypes.NewIPNet(utils.GenerateIPNet(network.IPRange.IP, network.IPRange.Mask)),
				AddWGAccess:  network.AddWGAccess,
				MyceliumKeys: network.MyceliumKeys,
			}
		}
	}

	return zNets
}

func (a *App) getProjectName(key string) string {
	return fmt.Sprintf("compose/%v/%v", a.Client.TwinID, key)
}

func (a *App) getDeploymentName(key string) string {
	return fmt.Sprintf("dl_%v", key)
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

// TODO: Calculate total MRU and SRU while populating the deployment data
func (a *App) generateMissingNodes(ctx context.Context) error {
	for _, deploymentData := range a.Config.DeploymentData {
		if deploymentData.NodeID != 0 {
			continue
		}

		// freeCRU is not in NodeFilter?
		var freeMRU, freeSRU uint64

		for _, service := range deploymentData.Services {
			freeMRU += service.Resources.Memory
			freeSRU += service.Resources.Rootfs
		}

		filter := types.NodeFilter{
			Status:  []string{"up"},
			FreeSRU: &freeSRU,
			FreeMRU: &freeMRU,
		}

		nodes, _, err := a.Client.GridProxyClient.Nodes(ctx, filter, types.Limit{})
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
