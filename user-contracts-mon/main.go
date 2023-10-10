package main

import (
	"flag"
	"log"
	"os"

	"github.com/NicoNex/echotron/v3"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	monitor "github.com/threefoldtech/tfgrid-sdk-go/user-contracts-mon/internal"
)

func main() {
	env := flag.String("e", "./.env", "Path to env file")
	flag.Parse()

	mon, err := monitor.NewMonitor(*env)
	if err != nil {
		log.Printf(err.Error())
		os.Exit(1)
	}
	log.Printf("monitoring bot has started and waiting for requests")

	tfPluginClient, err := deployer.NewTFPluginClient(mon.Mnemonic, "sr25519", mon.Network, "", "", "", 0, true)
	if err != nil {
		log.Printf("Failed to establish gird connection")
		os.Exit(1)
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
}
