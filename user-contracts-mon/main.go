package main

import (
	"flag"
	"log"
	"os"

	"github.com/NicoNex/echotron/v3"
	monitor "github.com/threefoldtech/tfgrid-sdk-go/user-contracts-mon/internal"
)

func main() {
	conf := flag.String("e", "./.env", "Path to config file")
	flag.Parse()

	mon, err := monitor.NewMonitor(*conf)
	if err != nil {
		log.Printf(err.Error())
		os.Exit(1)
	}
	log.Printf("bot has started and waiting for requests")

	for update := range echotron.PollingUpdates(mon.BotToken) {
		if update.Message.Text == "/start" {
			log.Printf("[%s] %s", update.Message.From.Username, update.Message.Text)

			okChan := make(chan bool, 0)
			go mon.StartMonitoring(update.ChatID(), okChan)

			ok := <-okChan

			if !ok {
				log.Printf("Failed to start monitoring")
				os.Exit(1)
			}

		}
	}
}
