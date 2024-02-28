// Package cmd for parsing command line arguments
package cmd

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/config"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
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
		t, err := deployer.NewTFPluginClient(cfg.Mnemonics, "sr25519", cfg.Network, "", "", "", 100, false, true)
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
