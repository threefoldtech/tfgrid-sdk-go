package main

import (
	"flag"
	"log"
	"os"

	"github.com/NicoNex/echotron/v3"
	monitor "github.com/threefoldtech/tfgrid-sdk-go/user-contracts-mon/internal"
)

func main() {
	conf := flag.String("cfg", "./.config", "Path to config file")
	flag.Parse()

	mon, err := monitor.NewMonitor(*conf)
	if err != nil {
		log.Printf(err.Error())
		os.Exit(1)
	}

	for update := range echotron.PollingUpdates(mon.BotToken) {
		if update.Message.Text == "/start" {

			log.Printf("[%s] %s", update.Message.From.Username, update.Message.Text)
			err = mon.StartMonitoring(update.ChatID())
			if err != nil {
				log.Printf(err.Error())
				os.Exit(1)
			}
		}
	}
}
