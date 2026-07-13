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

// getZDBCmd represents the get zdb command
var getZDBCmd = &cobra.Command{
	Use:   "zdb",
	Short: "Get deployed zdb",
	Args:  cobra.ExactArgs(1),
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

		zdb, err := command.GetDeployment(cmd.Context(), t, args[0])
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		s, err := json.MarshalIndent(zdb, "", "\t")
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		log.Info().Msg("zdb:\n" + string(s))
	},
}

func init() {
	getCmd.AddCommand(getZDBCmd)
}
