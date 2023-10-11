package monitor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NicoNex/echotron/v3"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"golang.org/x/exp/slices"
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
	if len(info) == 0 {
		return "", nil
	}
	return "contracts in grace period:\n" + info, nil
}

func getContractsAgainstDownNodes(tfPluginClient deployer.TFPluginClient) (string, error) {
	contracts, err := tfPluginClient.ContractsGetter.ListContractsByTwinID([]string{"Created"})
	if err != nil {
		return "", err
	}

	downNodes, err := getDownNodes(tfPluginClient)
	if err != nil {
		return "", err
	}

	contractsWithDownNodes := ""
	for _, contract := range contracts.NodeContracts {

		contractNodeId := int(contract.NodeID)
		if slices.Contains(downNodes, contractNodeId) {
			contractsWithDownNodes += fmt.Sprintf("- %s\n", contract.ContractID)
		}
	}

	if len(contractsWithDownNodes) == 0 {
		return "", nil
	}
	return "contracts against down nodes:\n" + contractsWithDownNodes, nil
}

func getDownNodes(tfPluginClient deployer.TFPluginClient) ([]int, error) {
	statusDown := "down"
	minRootfs := []uint64{2 * 1024 * 1024 * 1024}

	nodeFilter := types.NodeFilter{
		Status: &statusDown,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	nodes, err := deployer.FilterNodes(ctx, tfPluginClient, nodeFilter, nil, nil, minRootfs)
	if err != nil {
		return []int{}, errors.New("failed to find down nodes")
	}

	downNodesIds := make([]int, len(nodes))
	for _, node := range nodes {
		downNodesIds = append(downNodesIds, node.NodeID)
	}

	return downNodesIds, nil
}
