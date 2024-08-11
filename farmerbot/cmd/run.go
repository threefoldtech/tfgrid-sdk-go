package cmd

import (
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
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
		autoApprove, err := cmd.Flags().GetBool("auto-approve")
		if err != nil {
			return err
		}
		if !autoApprove {
			answer, err := disclaimerPrompt()
			if err != nil {
				return err
			}
			if answer == "no" {
				log.Info().Msg("Farmer bot is exiting ....")
				os.Exit(0)
			}

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

		if err := parser.ValidateConfig(config, network); err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}

		config.ContinueOnPoweringOnErr = continueOnPoweringOnErr

		farmerBot, err := internal.NewFarmerBot(cmd.Context(), config, network, mnemonicOrSeed, keyType)
		if err != nil {
			return err
		}

		return farmerBot.Run(cmd.Context())
	},
}

func disclaimerPrompt() (string, error) {
	prompt := promptui.Select{
		Label: "\033[33mWarning: The Farmerbot is an optional feature developed by ThreeFold. Use at your own risk. While ThreeFold will do its best to fix any issues with the Farmerbot and minting, if minting is affected by the use of the Farmerbot, ThreeFold cannot be held responsible. Do you wish to proceed? (yes/no)\033[0m",
		Items: []string{"yes", "no"},
	}
	_, result, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return result, nil
}
