package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/internal"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/parser"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run farmerbot to manage your farm",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(cmd.Flags().Args()) != 0 {
			return fmt.Errorf("'run' and %v cannot be used together, please use one command at a time", cmd.Flags().Args())
		}

		network, mnemonicOrSeed, keyType, err := getDefaultFlags(cmd)
		if err != nil {
			return err
		}

		configPath, err := cmd.Flags().GetString("config")
		if err != nil {
			return fmt.Errorf("invalid config path '%s'", configPath)
		}

		continueOnPoweringOnErr, err := cmd.Flags().GetBool("continue-power-on-error")
		if err != nil {
			return fmt.Errorf("invalid `continue-power-on-error` flag %v", continueOnPoweringOnErr)
		}

		fileContent, err := parser.ReadFile(configPath)
		if err != nil {
			return err
		}

		config, err := parser.ParseIntoConfig(fileContent)
		if err != nil {
			return err
		}
		sub, err := substrate.NewManager(internal.SubstrateURLs[network]...).Substrate()
		if err != nil {
			return err
		}
		defer sub.Close()
		err = parser.ValidateInput(&config, sub)
		if err != nil {
			return err
		}

		config.ContinueOnPoweringOnErr = continueOnPoweringOnErr

		farmerBot, err := internal.NewFarmerBot(cmd.Context(), config, network, mnemonicOrSeed, keyType)
		if err != nil {
			return err
		}

		return farmerBot.Run(cmd.Context())
	},
}
