package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/cosmos/go-bip39"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/stellar/go/support/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/constants"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/internal"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/parser"
	"github.com/vedhavyas/go-subkey"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run farmerbot to manage your farm",
	RunE: func(cmd *cobra.Command, args []string) error {
		network, mnemonic, err := getDefaultFlags(cmd)
		if err != nil {
			return err
		}

		configPath, err := cmd.Flags().GetString("config")
		if err != nil {
			return fmt.Errorf("invalid config path '%s'", configPath)
		}

		inputs, err := readConfigs(configPath)
		if err != nil {
			return err
		}

		farmerBot, err := internal.NewFarmerBot(cmd.Context(), inputs, network, mnemonic)
		if err != nil {
			return err
		}

		return farmerBot.Run(cmd.Context())
	},
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

func readConfigs(configPath string) (Config, error) {
	content, format, err := parser.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}

	return parser.ParseIntoConfig(content, format)
}
