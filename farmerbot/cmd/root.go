// Package cmd for parsing command line arguments
package cmd

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/constants"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/version"
)

// farmerBotCmd represents the root base command when called without any subcommands
var farmerBotCmd = &cobra.Command{
	Use:   "farmerbot",
	Short: "Run farmerbot to manage your farms",
	Long:  fmt.Sprintf(`Welcome to the farmerbot (%v). The farmerbot is a service that a farmer can run allowing him to automatically manage the nodes of his farm.`, version.Version),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	farmerBotCmd.AddCommand(versionCmd)
	farmerBotCmd.AddCommand(runCmd)

	err := farmerBotCmd.Execute()
	if err != nil {
		log.Fatal().Err(err).Send()
	}
}

func init() {
	farmerBotCmd.Flags().StringP("network", "n", constants.MainNetwork, "the grid network to use")
	farmerBotCmd.Flags().StringP("mnemonic", "m", "", "the mnemonic of the account of the farmer")
	farmerBotCmd.Flags().StringP("seed", "s", "", "the hex seed of the account of the farmer")
	farmerBotCmd.Flags().BoolP("debug", "d", false, "by setting this flag the farmerbot will print debug logs too")
	farmerBotCmd.MarkFlagsMutuallyExclusive("mnemonic", "seed")

	runCmd.Flags().StringP("config", "c", "", "enter your config file that includes your farm, node and power configs. Available formats are [json, yml, toml]")

	startCmd.Flags().Uint32("node", 0, "enter the node ID you want to use")
}
