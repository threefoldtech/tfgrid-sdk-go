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
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/internal"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/parser"
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
	farmerBotCmd.PersistentFlags().StringP("env", "e", "", "enter your env file that includes your NETWORK and MNEMONIC_OR_SEED")

	farmerBotCmd.PersistentFlags().StringP("network", "n", internal.MainNetwork, fmt.Sprintf("the grid network to use, available networks: %s, %s, %s, and %s", internal.DevNetwork, internal.QaNetwork, internal.TestNetwork, internal.MainNetwork))
	farmerBotCmd.PersistentFlags().StringP("mnemonic", "m", "", "the mnemonic of the account of the farmer")
	farmerBotCmd.PersistentFlags().StringP("seed", "s", "", "the hex seed of the account of the farmer")
	farmerBotCmd.MarkFlagsMutuallyExclusive("mnemonic", "seed")

	farmerBotCmd.MarkFlagsMutuallyExclusive("env", "network")
	farmerBotCmd.MarkFlagsMutuallyExclusive("env", "seed")
	farmerBotCmd.MarkFlagsMutuallyExclusive("env", "mnemonic")

	farmerBotCmd.PersistentFlags().BoolP("debug", "d", false, "by setting this flag the farmerbot will print debug logs too")

	runCmd.Flags().StringP("config", "c", "", "enter your config file that includes your farm, node and power configs. Allowed format is yml/yaml")

	startCmd.Flags().Uint32("node", 0, "enter the node ID you want to use")

	startAllCmd.Flags().Uint32("farm", 0, "enter the farm ID you want to start your nodes in")
}

func getDefaultFlags(cmd *cobra.Command) (network string, mnemonicOrSeed string, err error) {
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		err = fmt.Errorf("invalid log debug mode input '%v' with error: %w", debug, err)
		return
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	envPath, err := cmd.Flags().GetString("env")
	if err != nil {
		err = fmt.Errorf("invalid env path '%s'", envPath)
		return
	}

	if len(envPath) != 0 {
		envContent, err := parser.ReadFile(envPath)
		if err != nil {
			return "", "", err
		}

		return parser.ParseEnv(string(envContent))
	}

	network, err = cmd.Flags().GetString("network")
	if err != nil {
		err = fmt.Errorf("invalid network input '%s' with error: %w", network, err)
		return
	}

	if !slices.Contains([]string{internal.DevNetwork, internal.QaNetwork, internal.TestNetwork, internal.MainNetwork}, network) {
		err = fmt.Errorf("network must be one of %s, %s, %s, and %s not '%s'", internal.DevNetwork, internal.QaNetwork, internal.TestNetwork, internal.MainNetwork, network)
		return
	}

	mnemonic, err := cmd.Flags().GetString("mnemonic")
	if err != nil {
		err = fmt.Errorf("invalid mnemonic input '%s' with error: %w", mnemonic, err)
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
		err = fmt.Errorf("invalid seed input '%s' with error: %w", seed, err)
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
