// Package cmd for parsing command line arguments
package cmd

import (
	"encoding/json"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	command "github.com/threefoldtech/zos_sdk_go/grid-cli/internal/cmd"
	"github.com/threefoldtech/zos_sdk_go/grid-cli/internal/config"
	"github.com/threefoldtech/zos_sdk_go/grid-client/deployer"
)

// getGatewayNameCmd represents the get gateway name command
var getGatewayNameCmd = &cobra.Command{
	Use:   "name",
	Short: "Get deployed gateway name",
	Run: func(cmd *cobra.Command, args []string) {
		noColor, err := cmd.Flags().GetBool("no-color")
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		disableSentry, err := cmd.Flags().GetBool("disable-sentry")
		if err != nil {
			log.Fatal().Err(err).Send()
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

		if disableSentry {
			opts = append(opts, deployer.WithDisableSentry())
		}
		t, err := deployer.NewTFPluginClient(cfg.Mnemonics, opts...)
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		gateway, err := command.GetGatewayName(cmd.Context(), t, args[0])
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		s, err := json.MarshalIndent(gateway, "", "\t")
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		log.Info().Msg("gateway name:\n" + string(s))
	},
}

func init() {
	getGatewayCmd.AddCommand(getGatewayNameCmd)
}
