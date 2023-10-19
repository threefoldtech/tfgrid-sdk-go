package monitor

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/NicoNex/echotron/v3"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/graphql"
)

// Monitor struct of parsed configration
type Monitor struct {
	Bot      echotron.API
	Mnemonic string
	BotToken string
	Network  string
	interval int
}

// NewMonitor creates a new monitor from parsed config/env file
func NewMonitor(conf Config) Monitor {
	mon := Monitor{}
	mon.Mnemonic = conf.mnemonic
	mon.Network = conf.network
	mon.interval = conf.interval
	mon.BotToken = conf.botToken
	mon.Bot = echotron.NewAPI(mon.BotToken)

	return mon
}

// StartMonitoring starts monitoring the contracts with
// specific mnemonics and notify subscribed chats every fixed interval
func (mon Monitor) StartMonitoring(tfPluginClient deployer.TFPluginClient, addChatChan chan int64, stopChatChan chan int64) {
	chatIDs := map[int64]bool{}
	ticker := time.NewTicker(time.Duration(mon.interval) * time.Hour)

	for {
		select {
		case chatID := <-addChatChan:
			chatIDs[chatID] = true

		case chatID := <-stopChatChan:
			chatIDs[chatID] = false

		case <-ticker.C:
			contractsInGracePeriod, contractsAgainstDownNodes, err := runMonitor(tfPluginClient)
			err = mon.sendResponse(chatIDs, contractsInGracePeriod, contractsAgainstDownNodes, err)
			if err != nil {
				return
			}
		}
	}
}

func runMonitor(tfPluginClient deployer.TFPluginClient) (string, string, error) {
	contractsInGracePeriod, err := getContractsInGracePeriod(tfPluginClient)
	if err != nil {
		return "", "", err
	}

	contractsAgainstDownNodes, err := getContractsAgainstDownNodes(tfPluginClient)
	if err != nil {
		return "", "", err
	}

	return contractsInGracePeriod, contractsAgainstDownNodes, nil
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

func (mon Monitor) sendResponse(chatIDs map[int64]bool, contractsInGracePeriod, contractsAgainstDownNodes string, err error) error {
	if err != nil {
		for chatID, ok := range chatIDs {
			if ok {
				_, err := mon.Bot.SendMessage("Failed to load contracts", chatID, nil)
				if err != nil {
					log.Println(err)
				}
			}
		}

		log.Println(err)
		return err
	}

	for chatID, ok := range chatIDs {
		if ok {
			if contractsInGracePeriod != "" {
				_, err := mon.Bot.SendMessage(contractsInGracePeriod, chatID, nil)
				if err != nil {
					log.Println(err)
				}
			}

			if contractsAgainstDownNodes != "" {
				_, err := mon.Bot.SendMessage(contractsAgainstDownNodes, chatID, nil)
				if err != nil {
					log.Println(err)
				}
			}
		}
	}
	return nil
}
