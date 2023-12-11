package cmd

import (
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/config"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
)

// cancelContracts represents the cancel contracts command
var cancelContracts = &cobra.Command{
	Use:   "contracts",
	Short: "Cancel twin contracts",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.GetUserConfig()
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		t, err := deployer.NewTFPluginClient(cfg.Mnemonics, "sr25519", cfg.Network, "", "", "", 100, false)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		var contracts []uint64
		for _, arg := range args {
			contract, err := strconv.ParseUint(arg, 10, 64)
			if err != nil {
				log.Fatal().Err(err).Msgf("%s not a valid contract id", arg)
			}
			contracts = append(contracts, contract)
		}
		err = t.BatchCancelContract(contracts)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to cancel contracts")
		}
		log.Info().Msg("contracts canceled successfully")
	},
}

func init() {
	cancelCmd.AddCommand(cancelContracts)

}
