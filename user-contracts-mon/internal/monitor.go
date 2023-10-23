package monitor

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	tgapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/graphql"
)

// Monitor struct of parsed configration
type Monitor struct {
	Bot      *tgapi.BotAPI
	BotToken string
	interval int
}

type User struct {
	ChatID   int64
	mnemonic string
	network  string
}

// NewMonitor creates a new monitor from parsed config/env file
func NewMonitor(conf Config) (Monitor, error) {
	mon := Monitor{}
	mon.interval = conf.interval
	mon.BotToken = conf.botToken

	bot, err := tgapi.NewBotAPI(mon.BotToken)
	if err != nil {
		return Monitor{}, err
	}

	mon.Bot = bot

	return mon, nil
}

// NewUser parses the user message to mnemonic and network and validate them
func NewUser(msg tgapi.Update) (User, error) {
	user := User{}
	info := strings.Split(msg.Message.Text, "\n")

	log.Println(info)
	if !strings.Contains(info[0], "network=") || !strings.Contains(info[1], "mnemonic=") {
		return user, errors.New("invalid format")
	}
	network := strings.Split(info[0], "=")[1]
	mnemonic := strings.Split(info[1], "=")[1]

	tfPluginClient, err := deployer.NewTFPluginClient(mnemonic, "sr25519", network, "", "", "", 0, true)
	if err != nil {
		return user, errors.New("failed to establish gird connection")
	}
	tfPluginClient.Close()

	user.ChatID = msg.FromChat().ID
	user.mnemonic = mnemonic
	user.network = network

	return user, nil
}

// StartMonitoring starts monitoring the contracts with
// specific mnemonics and notify subscribed chats every fixed interval
func (mon Monitor) StartMonitoring(addChatChan chan User, stopChatChan chan int64) {
	users := map[int64]User{}
	ticker := time.NewTicker(time.Duration(mon.interval) * time.Hour)

	for {
		select {
		case user := <-addChatChan:
			users[user.ChatID] = user
			contractsInGracePeriod, contractsAgainstDownNodes, err := runMonitor(user.mnemonic, user.network)
			err = mon.sendResponse(user.ChatID, contractsInGracePeriod, contractsAgainstDownNodes, err)
			if err != nil {
				log.Println(err)
			}

		case chatID := <-stopChatChan:
			users[chatID] = User{}

		case <-ticker.C:
			for chatID, user := range users {
				if user.mnemonic != "" {
					contractsInGracePeriod, contractsAgainstDownNodes, err := runMonitor(user.mnemonic, user.network)
					err = mon.sendResponse(chatID, contractsInGracePeriod, contractsAgainstDownNodes, err)
					if err != nil {
						log.Println(err)
					}
				}
			}
		}
	}
}

func runMonitor(mnemonic, network string) (string, string, error) {
	tfPluginClient, err := deployer.NewTFPluginClient(mnemonic, "sr25519", network, "", "", "", 0, true)
	if err != nil {
		return "", "", errors.New("failed to establish gird connection")
	}
	defer tfPluginClient.Close()

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
	contractsStruct, err := tfPluginClient.ContractsGetter.ListContractsByTwinID([]string{"GracePeriod"})
	if err != nil {
		return "", err
	}

	allContracts := contractsStruct.NameContracts
	allContracts = append(allContracts, contractsStruct.NodeContracts...)
	allContracts = append(allContracts, contractsStruct.RentContracts...)

	info := ""
	for _, contract := range allContracts {
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
	downNodes := make(chan string)

	for _, contract := range nodeContracts {
		go isNodeDown(tfPluginClient, contract, downNodes)
	}

	for range nodeContracts {
		id := <-downNodes
		if id != "" {
			contractsIds += fmt.Sprintf("- %s\n", id)
		}
	}

	if contractsIds == "" {
		return "", nil
	}
	return "contracts against down nodes:\n" + contractsIds, nil
}

func isNodeDown(tfPluginClient deployer.TFPluginClient, contract graphql.Contract, downNodes chan string) {
	cli, err := tfPluginClient.NcPool.GetNodeClient(tfPluginClient.SubstrateConn, contract.NodeID)
	if err != nil {
		downNodes <- contract.ContractID
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	err = cli.IsNodeUp(ctx)
	if err != nil {
		downNodes <- contract.ContractID
		return
	}

	downNodes <- ""
}

func (mon Monitor) sendResponse(chatID int64, contractsInGracePeriod, contractsAgainstDownNodes string, err error) error {
	if err != nil {
		msg := tgapi.NewMessage(chatID, "Failed to load contracts")
		_, err := mon.Bot.Send(msg)
		if err != nil {
			return err
		}
		return err
	}

	if contractsInGracePeriod != "" {
		msg := tgapi.NewMessage(chatID, contractsInGracePeriod)
		_, err := mon.Bot.Send(msg)
		if err != nil {
			return err
		}
	}

	if contractsAgainstDownNodes != "" {
		msg := tgapi.NewMessage(chatID, contractsAgainstDownNodes)
		_, err := mon.Bot.Send(msg)
		if err != nil {
			return err
		}
	}
	return nil
}
