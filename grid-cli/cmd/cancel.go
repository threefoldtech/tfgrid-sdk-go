// Package cmd for parsing command line arguments
package cmd

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/zos_sdk_go/grid-cli/internal/config"
	"github.com/threefoldtech/zos_sdk_go/grid-client/deployer"
)

// cancelCmd represents the cancel command
var cancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel resources on Threefold grid",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.GetUserConfig()
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		disableSentry, err := cmd.Flags().GetBool("disable-sentry")
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		opts := []deployer.PluginOpt{
			deployer.WithNetwork(cfg.Network),
			deployer.WithRMBTimeout(100),
		}

		if disableSentry {
			opts = append(opts, deployer.WithDisableSentry())
		}

		t, err := deployer.NewTFPluginClient(cfg.Mnemonics, opts...)
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		err = t.CancelByProjectName(fmt.Sprintf("vm/%s", args[0]))
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		err = t.CancelByProjectName(fmt.Sprintf("kubernetes/%s", args[0]))
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		err = t.CancelByProjectName(args[0])
		if err != nil {
			log.Fatal().Err(err).Send()
		}
	},
}

func init() {
	rootCmd.AddCommand(cancelCmd)
}
