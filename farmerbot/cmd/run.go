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
		network, mnemonicOrSeed, err := getDefaultFlags(cmd)
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

		farmerBot, err := internal.NewFarmerBot(cmd.Context(), config, network, mnemonicOrSeed)
		if err != nil {
			return err
		}

		return farmerBot.Run(cmd.Context())
	},
}
