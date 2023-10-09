package monitor

import (
	"fmt"
	"strconv"
	"time"

	"github.com/NicoNex/echotron/v3"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
)

type Monitor struct {
	Bot      echotron.API
	Mnemonic string `env:"MNEMONIC"`
	BotToken string `env:"BOT_TOKEN"`
	Network  string `env:"NETWORK"`
	interval int    `env:"INTERVAL"`
}

func NewMonitor(envPath string) (Monitor, error) {
	mon := Monitor{}
	envMap, err := parseConfig(envPath)
	if err != nil {
		return mon, err
	}
	for key, value := range envMap {
		switch key {
		case "MNEMONIC":
			mon.Mnemonic = value
		case "BOT_TOKEN":
			mon.BotToken = value
		case "INTERVAL":
			value, err := strconv.Atoi(value)
			if err != nil {
				return Monitor{}, err
			}
			mon.interval = value
		case "NETWORK":
			mon.Network = value
		}
	}
	mon.Bot = echotron.NewAPI(mon.BotToken)
	return mon, nil
}

func (mon Monitor) StartMonitoring(chatID int64) error {
	log.Debug().Msgf("mnemonics: %s", mon.Mnemonic)
	log.Debug().Msgf("network: %s", mon.Network)

	tfPluginClient, err := deployer.NewTFPluginClient(mon.Mnemonic, "sr25519", mon.Network, "", "", "", 0, true)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Duration(mon.interval) * time.Second)
	for range ticker.C {
		contracts, err := tfPluginClient.ContractsGetter.ListContractsByTwinID([]string{"GracePeriod"})
		if err != nil {
			return err
		}

		mon.Bot.SendMessage(fmt.Sprintf("%v", contracts), chatID, nil)
	}
	return nil
}
