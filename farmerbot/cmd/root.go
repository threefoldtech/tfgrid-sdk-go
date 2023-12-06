// Package cmd for parsing command line arguments
package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/cosmos/go-bip39"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/constants"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/version"
	"github.com/vedhavyas/go-subkey"
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
	farmerBotCmd.AddCommand(startCmd)
	startCmd.AddCommand(startAllCmd)

	err := farmerBotCmd.Execute()
	if err != nil {
		log.Fatal().Err(err).Send()
	}
}

func init() {
	farmerBotCmd.PersistentFlags().StringP("network", "n", constants.MainNetwork, "the grid network to use")
	farmerBotCmd.PersistentFlags().StringP("mnemonic", "m", "", "the mnemonic of the account of the farmer")
	farmerBotCmd.PersistentFlags().StringP("seed", "s", "", "the hex seed of the account of the farmer")
	farmerBotCmd.PersistentFlags().BoolP("debug", "d", false, "by setting this flag the farmerbot will print debug logs too")
	farmerBotCmd.MarkFlagsMutuallyExclusive("mnemonic", "seed")

	runCmd.Flags().StringP("config", "c", "", "enter your config file that includes your farm, node and power configs. Available format is yml/yaml")

	startCmd.Flags().Uint32("node", 0, "enter the node ID you want to use")

	startAllCmd.Flags().Uint32("farm", 0, "enter the farm ID you want to start your nodes in")
}

func getDefaultFlags(cmd *cobra.Command) (network string, mnemonicOrSeed string, err error) {
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		err = errors.Wrapf(err, "invalid log debug mode input '%v'", debug)
		return
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	network, err = cmd.Flags().GetString("network")
	if err != nil {
		err = errors.Wrapf(err, "invalid network input '%s'", network)
		return
	}

	if !slices.Contains([]string{constants.DevNetwork, constants.QaNetwork, constants.TestNetwork, constants.MainNetwork}, network) {
		err = fmt.Errorf("network must be one of %s, %s, %s, and %s not '%s'", constants.DevNetwork, constants.QaNetwork, constants.TestNetwork, constants.MainNetwork, network)
		return
	}

	mnemonic, err := cmd.Flags().GetString("mnemonic")
	if err != nil {
		err = errors.Wrapf(err, "invalid mnemonic input '%s'", mnemonic)
		return
	}

	if len(strings.TrimSpace(mnemonic)) > 0 {
		if !bip39.IsMnemonicValid(mnemonic) {
			err = fmt.Errorf("invalid mnemonic input '%s'", mnemonic)
			return
		}

		mnemonicOrSeed = mnemonic
		return
	}

	seed, err := cmd.Flags().GetString("seed")
	if err != nil {
		err = errors.Wrapf(err, "invalid seed input '%s'", seed)
		return
	}

	if len(strings.TrimSpace(seed)) == 0 && len(strings.TrimSpace(mnemonic)) == 0 {
		err = errors.New("seed/mnemonic is required")
		return
	}

	_, ok := subkey.DecodeHex(seed)
	if !ok {
		err = fmt.Errorf("invalid seed input '%s'", seed)
		return
	}

	mnemonicOrSeed = seed
	return
}
