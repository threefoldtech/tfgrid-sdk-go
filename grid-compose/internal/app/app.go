package app

import (
	"fmt"
	"os"
	"strconv"

	"github.com/cosmos/go-bip39"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/config"
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

func (a *App) GetProjectName(key string) string {
	return fmt.Sprintf("compose/%v/%v", a.Client.TwinID, key)
}

func (a *App) GetDeploymentName(key string) string {
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

// validateCredentials validates the mnemonic and network
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
