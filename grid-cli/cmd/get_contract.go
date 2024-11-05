package cmd

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/config"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
)

// getContractCmd represents the get contract command
var getContractCmd = &cobra.Command{
	Use:   "contract",
	Short: "Get twin contract",
	Run: func(cmd *cobra.Command, args []string) {
		noColor, err := cmd.Flags().GetBool("no-color")
		if err != nil {
			return
		}

		cfg, err := config.GetUserConfig()
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		opts := []deployer.PluginOpt{
			deployer.WithNetwork(cfg.Network),
			deployer.WithRMBTimeout(100),
		}

		if noColor {
			opts = append(opts, deployer.WithNoColorLogs())
		}

		t, err := deployer.NewTFPluginClient(cfg.Mnemonics, opts...)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		if len(args) == 0 {
			log.Info().Msg("please specify a contract")
			return
		}
		contractID, err := strconv.ParseUint(args[0], 10, 32)
		if err != nil {
			log.Fatal().Err(err).Msg("not a valid contract id")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		contract, err := t.GridProxyClient.Contract(ctx, uint32(contractID))
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to get contract %d", contractID)
		}
		s, err := json.MarshalIndent(contract, "", "\t")
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		log.Info().Msg("contract:\n" + string(s))
	},
}

func init() {
	getCmd.AddCommand(getContractCmd)
	getContractCmd.Flags().BoolP("no-color", "n", false, "disable output styling")
}
