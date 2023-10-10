package monitor

import (
	"fmt"
	"time"

	"github.com/NicoNex/echotron/v3"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/graphql"
)

// Monitor struct of parsed configration
type Monitor struct {
	Bot      echotron.API
	Mnemonic string `env:"MNEMONIC"`
	BotToken string `env:"BOT_TOKEN"`
	Network  string `env:"NETWORK"`
	interval int    `env:"INTERVAL"`
}

// NewMonitor creates a new monitor from parsed config/env file
func NewMonitor(envPath string) (Monitor, error) {
	envContent, err := parseFile(envPath)
	if err != nil {
		return Monitor{}, err
	}

	mon, err := parseMonitor(envContent)
	if err != nil {
		return Monitor{}, err
	}
	mon.Bot = echotron.NewAPI(mon.BotToken)

	return mon, nil
}

// StartMonitoring starts monitoring the contracts with
// specific mnemonics and notify them every fixed interval
func (mon Monitor) StartMonitoring(chatID int64) error {
	log.Debug().Msgf("mnemonics: %s", mon.Mnemonic)
	log.Debug().Msgf("network: %s", mon.Network)

	tfPluginClient, err := deployer.NewTFPluginClient(mon.Mnemonic, "sr25519", mon.Network, "", "", "", 0, true)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Duration(mon.interval) * time.Second)

	for ; true; <-ticker.C {
		contracts, err := tfPluginClient.ContractsGetter.ListContractsByTwinID([]string{"GracePeriod"})
		if err != nil {
			return err
		}
		contractsInfo := formatContracts(contracts)

		mon.Bot.SendMessage(contractsInfo, chatID, nil)
	}
	return nil
}

func formatContracts(contracts graphql.Contracts) string {
	info := "contracts in grace period:\n"
	for _, contract := range contracts.NameContracts {
		info += fmt.Sprintf("- %s\n", contract.ContractID)
	}

	for _, contract := range contracts.NodeContracts {
		info += fmt.Sprintf("- %s\n", contract.ContractID)
	}

	for _, contract := range contracts.RentContracts {
		info += fmt.Sprintf("- %s\n", contract.ContractID)
	}
	if len(info) == 0 {
		info = "- None"
	}
	return info
}
