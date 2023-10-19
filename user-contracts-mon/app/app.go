package app

import (
	"errors"
	"flag"
	"log"

	"github.com/NicoNex/echotron/v3"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	monitor "github.com/threefoldtech/tfgrid-sdk-go/user-contracts-mon/internal"
)

func Start() error {
	env := ""
	flag.StringVar(&env, "e", "", "Path to env file")
	flag.Parse()

	conf, err := monitor.ParseConfig(env)
	if err != nil {
		return err
	}

	mon := monitor.NewMonitor(conf)
	log.Printf("monitoring bot has started and waiting for requests")

	tfPluginClient, err := deployer.NewTFPluginClient(mon.Mnemonic, "sr25519", mon.Network, "", "", "", 0, true)
	if err != nil {
		return errors.New("failed to establish gird connection")
	}
	log.Printf("grid connection established successfully")

	addChatChan := make(chan int64)
	stopChatChan := make(chan int64)
	go mon.StartMonitoring(tfPluginClient, addChatChan, stopChatChan)

	for update := range echotron.PollingUpdates(mon.BotToken) {
		switch update.Message.Text {
		case "/start":
			addChatChan <- update.ChatID()
			log.Printf("[%s] %s", update.Message.From.Username, update.Message.Text)

		case "/stop":
			stopChatChan <- update.ChatID()
			log.Printf("[%s] %s", update.Message.From.Username, update.Message.Text)

		default:
			_, err = mon.Bot.SendMessage("invalid message or command", update.ChatID(), nil)
			if err != nil {
				return errors.New("failed to respond to the user" + update.Message.From.Username)
			}
		}
	}
	return nil
}
