package app

import (
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/cosmos/go-bip39"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/pkg/parser/config"
)

// App is the main application struct that holds the client and the config data
type App struct {
	Client *deployer.TFPluginClient
	Config *config.Config
}

// NewApp creates a new instance of the application
func NewApp(net, mnemonic, configPath string) (*App, error) {
	if !validateCredentials(mnemonic, net) {
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

	if err := config.Validate(); err != nil {
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

// GetProjectName returns the project name for the given key
func (a *App) GetProjectName(key string) string {
	return fmt.Sprintf("vm/compose/%v/%v", a.Client.TwinID, key)
}

// GetDeploymentName returns the deployment name for the given key
func (a *App) GetDeploymentName(key string) string {
	return fmt.Sprintf("dl_%v", key)
}

// GenerateDefaultNetworkName generates a default network name based on the sorted service names.
func (a *App) GenerateDefaultNetworkName() string {
	var serviceNames []string
	for serviceName := range a.Config.Services {
		serviceNames = append(serviceNames, serviceName)
	}
	sort.Strings(serviceNames)

	var defaultNetName string
	for _, serviceName := range serviceNames {
		defaultNetName += serviceName[:2]
	}

	return fmt.Sprintf("net_%s", defaultNetName)
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

// validateCredentials validates the mnemonics and network values of the user
func validateCredentials(mnemonics, network string) bool {
	return validateMnemonics(mnemonics) && validateNetwork(network)
}

func validateMnemonics(mnemonics string) bool {
	return bip39.IsMnemonicValid(mnemonics)
}

func validateNetwork(network string) bool {
	switch network {
	case "test", "dev", "main", "qa":
		return true
	default:
		return false
	}
}
