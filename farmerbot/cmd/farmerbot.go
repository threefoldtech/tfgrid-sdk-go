package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/cosmos/go-bip39"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/stellar/go/support/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/constants"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/models"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/parser"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/server"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/version"
	"github.com/vedhavyas/go-subkey/v2"
)

// farmerBotCmd represents the root base command when called without any subcommands
var farmerBotCmd = &cobra.Command{
	Use:   "farmerbot",
	Short: "Run farmerbot to manage your farms",
	Long:  fmt.Sprintf(`Welcome to the farmerbot (%v). The farmerbot is a service that a farmer can run allowing him to automatically manage the nodes of his farm.`, version.Version),
	RunE: func(cmd *cobra.Command, args []string) error {
		network, mnemonic, err := getDefaultFlags(cmd)
		if err != nil {
			return err
		}

		configPath, err := cmd.Flags().GetString("config")
		if err != nil {
			return fmt.Errorf("invalid config path '%s'", configPath)
		}

		inputs, err := readInputConfigs(configPath)
		if err != nil {
			return err
		}

		farmerBot, err := server.NewFarmerBot(cmd.Context(), inputs, network, mnemonic)
		if err != nil {
			return err
		}

		return farmerBot.Run(cmd.Context())
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	farmerBotCmd.AddCommand(versionCmd)

	err := farmerBotCmd.Execute()
	if err != nil {
		log.Fatal().Err(err).Send()
	}
}

func init() {
	farmerBotCmd.Flags().StringP("config", "c", "", "enter your config file that includes your farm, node and power configs. Available formats are [json, yml, toml]")
	farmerBotCmd.Flags().StringP("network", "n", constants.MainNetwork, "the grid network to use")
	farmerBotCmd.Flags().StringP("mnemonic", "m", "", "the mnemonic of the account of the farmer")
	farmerBotCmd.Flags().StringP("seed", "s", "", "the hex seed of the account of the farmer")
	farmerBotCmd.Flags().BoolP("debug", "d", false, "by setting this flag the farmerbot will print debug logs too")
	farmerBotCmd.MarkFlagsMutuallyExclusive("mnemonic", "seed")
}

func getDefaultFlags(cmd *cobra.Command) (network string, mnemonic string, err error) {
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

	mnemonic, err = cmd.Flags().GetString("mnemonic")
	if err != nil {
		err = errors.Wrapf(err, "invalid mnemonic input '%s'", mnemonic)
		return
	}

	if len(strings.TrimSpace(mnemonic)) > 0 {
		if !bip39.IsMnemonicValid(mnemonic) {
			err = fmt.Errorf("invalid mnemonic input '%s'", mnemonic)
			return
		}
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

	mnemonic = seed
	return
}

func readInputConfigs(configPath string) (models.InputConfig, error) {
	content, format, err := parser.ReadFile(configPath)
	if err != nil {
		return models.InputConfig{}, err
	}

	return parser.ParseIntoInputConfig(content, format)
}
