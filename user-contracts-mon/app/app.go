package app

import (
	"flag"
	"fmt"
	"log"

	tgapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

	addChatChan := make(chan monitor.User)
	stopChatChan := make(chan int64)
	go mon.StartMonitoring(addChatChan, stopChatChan)

	u := tgapi.NewUpdate(0)
	u.Timeout = 60
	updates := mon.Bot.GetUpdatesChan(u)

	for update := range updates {
		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
		switch update.Message.Text {
		case "/start":

			msg := tgapi.NewMessage(update.FromChat().ID, fmt.Sprintf("Please send your network and mnemonic in the form\nnetwork=<network>\nmnemonic=<mnemonic>"))
			_, err := mon.Bot.Send(msg)
			if err != nil {
				log.Println(err)
			}

		case "/stop":
			stopChatChan <- update.FromChat().ID

		default:
			user, err := monitor.NewUser(update)
			if err != nil {
				msg := tgapi.NewMessage(update.FromChat().ID, err.Error())
				_, err := mon.Bot.Send(msg)
				if err != nil {
					log.Println(err)
				}
			}
			addChatChan <- user
		}
	}
	return nil
}
