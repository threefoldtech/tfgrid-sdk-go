package main

import (
	"log"

	monitor "github.com/Eslam-Nawara/user-contract-mon/internal"
	"github.com/NicoNex/echotron/v3"
)

func main() {
	mon, err := monitor.NewMonitor("./.config")
	if err != nil {
		panic("failed to load configrations")
	}

	for update := range echotron.PollingUpdates(mon.BotToken) {
		if update.Message.Text == "/start" {

			log.Printf("[%s] %s", update.Message.From.Username, update.Message.Text)
			err = mon.StartMonitoring(update.ChatID())
			if err != nil {
				panic(err)
			}
		}
	}
}
