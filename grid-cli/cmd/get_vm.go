// Package cmd for parsing command line arguments
package cmd

import (
	"encoding/json"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	command "github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/cmd"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-cli/internal/config"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
)

// getVMCmd represents the get vm command
var getVMCmd = &cobra.Command{
	Use:   "vm",
	Short: "Get deployed vm",
	Args:  cobra.ExactArgs(1),
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

		vm, err := command.GetVM(cmd.Context(), t, args[0])
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		s, err := json.MarshalIndent(vm, "", "\t")
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		log.Info().Msg("vm:\n" + string(s))
	},
}

func init() {
	getCmd.AddCommand(getVMCmd)
	getVMCmd.Flags().BoolP("no-color", "n", false, "disable output styling")
}
