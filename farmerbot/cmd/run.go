package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/internal"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/parser"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run farmerbot to manage your farm",
	RunE: func(cmd *cobra.Command, args []string) error {
		network, mnemonicOrSeed, keyType, err := getDefaultFlags(cmd)
		if err != nil {
			return err
		}

		configPath, err := cmd.Flags().GetString("config")
		if err != nil {
			return fmt.Errorf("invalid config path '%s'", configPath)
		}

		fileContent, err := parser.ReadFile(configPath)
		if err != nil {
			return err
		}

		config, err := parser.ParseIntoConfig(fileContent)
		if err != nil {
			return err
		}
		continueOnPoweringOnError, err := cmd.Flags().GetBool("continue-power-on-error")
		farmerBot, err := internal.NewFarmerBot(cmd.Context(), config, network, mnemonicOrSeed, keyType, continueOnPoweringOnError)
		if err != nil {
			return err
		}

		return farmerBot.Run(cmd.Context())
	},
}
