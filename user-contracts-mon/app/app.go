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

	mon, err := monitor.NewMonitor(conf)
	if err != nil {
		return err
	}
	log.Printf("monitoring bot has started and waiting for requests")

	tfPluginClient, err := deployer.NewTFPluginClient(mon.Mnemonic, "sr25519", mon.Network, "", "", "", 0, true)
	if err != nil {
		return errors.New("Failed to establish gird connection")
	}
	log.Printf("grid connection established successfully")

	for update := range echotron.PollingUpdates(mon.BotToken) {
		if update.Message.Text == "/start" {
			log.Printf("[%s] %s", update.Message.From.Username, update.Message.Text)
			go mon.StartMonitoring(tfPluginClient, update.ChatID())
		} else {
			mon.Bot.SendMessage("to start monitoring enter /start", update.ChatID(), nil)
		}
	}
	return nil
}
