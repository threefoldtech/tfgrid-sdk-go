package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/NicoNex/echotron/v3"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/graphql"
)

// Monitor struct of parsed configration
type Monitor struct {
	Bot      echotron.API
	Mnemonic string `env:"MNEMONIC"`
	BotToken string `env:"BOT_TOKEN"`
	Network  string `env:"NETWORK"`
	interval int    `env:"INTERVAL"`
}

// NewMonitor creates a new monitor from parsed config/env file
func NewMonitor(envPath string) (Monitor, error) {
	envFile, err := readFile(envPath)
	if err != nil {
		return Monitor{}, err
	}

	mon, err := parseMonitor(envFile)
	if err != nil {
		return Monitor{}, err
	}
	mon.Bot = echotron.NewAPI(mon.BotToken)

	return mon, nil
}

// StartMonitoring starts monitoring the contracts with
// specific mnemonics and notify them every fixed interval
func (mon Monitor) StartMonitoring(tfPluginClient deployer.TFPluginClient, chatID int64) {
	ticker := time.NewTicker(time.Duration(mon.interval) * time.Hour)
	for ; true; <-ticker.C {

		contractsInGracePeriod, err := getContractsInGracePeriod(tfPluginClient)
		if err != nil {
			mon.Bot.SendMessage("Failed to get contracts in grace period", chatID, nil)
			return
		}
		mon.Bot.SendMessage(contractsInGracePeriod, chatID, nil)

		contractsAgainstDownNodes, err := getContractsAgainstDownNodes(tfPluginClient)
		if err != nil {
			mon.Bot.SendMessage("Failed to get contracts against down nodes", chatID, nil)
			return
		}
		mon.Bot.SendMessage(contractsAgainstDownNodes, chatID, nil)
	}
}

func getContractsInGracePeriod(tfPluginClient deployer.TFPluginClient) (string, error) {
	contracts, err := tfPluginClient.ContractsGetter.ListContractsByTwinID([]string{"GracePeriod"})
	if err != nil {
		return "", err
	}

	info := ""
	for _, contract := range contracts.NameContracts {
		info += fmt.Sprintf("- %s\n", contract.ContractID)
	}

	for _, contract := range contracts.NodeContracts {
		info += fmt.Sprintf("- %s\n", contract.ContractID)
	}

	for _, contract := range contracts.RentContracts {
		info += fmt.Sprintf("- %s\n", contract.ContractID)
	}
	if info == "" {
		return "", nil
	}
	return "contracts in grace period:\n" + info, nil
}

func getContractsAgainstDownNodes(tfPluginClient deployer.TFPluginClient) (string, error) {
	contracts, err := tfPluginClient.ContractsGetter.ListContractsByTwinID([]string{"Created"})
	if err != nil {
		return "", err
	}

	nodeContracts := contracts.NodeContracts
	nodeContracts = append(nodeContracts, contracts.RentContracts...)

	contractsIds := ""
	upNodes := make(chan string)

	for _, contract := range nodeContracts {
		go isNodeUp(tfPluginClient, contract, upNodes)
	}

	for range nodeContracts {
		id := <-upNodes
		if id != "" {
			contractsIds += fmt.Sprintf("- %s\n", id)
		}
	}

	if contractsIds == "" {
		return "", nil
	}
	return "contracts against down nodes:\n" + contractsIds, nil
}

func isNodeUp(tfPluginClient deployer.TFPluginClient, contract graphql.Contract, upNodes chan string) {
	cli, err := tfPluginClient.NcPool.GetNodeClient(tfPluginClient.SubstrateConn, contract.NodeID)
	if err != nil {
		upNodes <- contract.ContractID
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	err = cli.IsNodeUp(ctx)
	if err != nil {
		upNodes <- contract.ContractID
		return
	}

	upNodes <- ""
}
