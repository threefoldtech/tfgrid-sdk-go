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
	"golang.org/x/exp/slices"
)

// Monitor struct of parsed configration
type Monitor struct {
	Bot      *tgapi.BotAPI
	BotToken string
	interval int
}

type User struct {
	ChatID   int64
	network  string
	mnemonic string
}

var validNetworks = []string{"dev", "qa", "test", "main"}

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

	if len(info) != 2 || !strings.Contains(info[0], "network=") || !strings.Contains(info[1], "mnemonic=") {
		return user, errors.New("invalid format")
	}

	if len(strings.Split(info[0], "=")) != 2 || len(strings.Split(info[1], "=")) != 2 {
		return user, errors.New("invalid mnemonic or network")
	}
	network := strings.Split(info[0], "=")[1]
	mnemonic := strings.Split(info[1], "=")[1]

	if !slices.Contains(validNetworks, network) {
		return user, errors.New("network must be one of dev, qa, test, and main")
	}

	user.ChatID = msg.FromChat().ID
	user.network = network
	user.mnemonic = mnemonic

	return user, nil
}

// StartMonitoring starts monitoring the contracts with
// specific mnemonics and notify subscribed chats every fixed interval
func (mon Monitor) StartMonitoring(addChatChan chan User, stopChatChan chan int64) {
	users := map[int64]deployer.TFPluginClient{}
	ticker := time.NewTicker(time.Duration(mon.interval) * time.Hour)

	for {
		select {

		case chatID := <-stopChatChan:
			if tfPluginClient, ok := users[chatID]; ok {
				tfPluginClient.Close()
				delete(users, chatID)
			}

		case <-ticker.C:
			for chatID, tfPluginClient := range users {
				contractsInGracePeriod, contractsAgainstDownNodes, err := runMonitor(tfPluginClient)
				err = mon.sendResponse(chatID, contractsInGracePeriod, contractsAgainstDownNodes, err)
				if err != nil {
					log.Println(err)
				}
			}

		case user := <-addChatChan:
			if tfPluginClient, ok := users[user.ChatID]; ok {
				tfPluginClient.Close()
			}

			tfPluginClient, err := deployer.NewTFPluginClient(user.mnemonic, "sr25519", user.network, "", "", "", 0, true)
			if err != nil {
				log.Println(err)

				msg := tgapi.NewMessage(user.ChatID, "failed to connect to the grid, please make sure to use valid mnemonic")
				_, err := mon.Bot.Send(msg)
				if err != nil {
					log.Println(err)
				}
				continue
			}
			users[user.ChatID] = tfPluginClient

			contractsInGracePeriod, contractsAgainstDownNodes, err := runMonitor(tfPluginClient)
			err = mon.sendResponse(user.ChatID, contractsInGracePeriod, contractsAgainstDownNodes, err)
			if err != nil {
				log.Println(err)
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
