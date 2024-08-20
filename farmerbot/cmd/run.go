package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"

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
				log.Info().Msg("Farmerbot is exiting ....")
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

// disclaimerPrompt reads user input and return string, error
func disclaimerPrompt() (string, error) {
	var answer string
	disclaimer := "\033[33mWarning: The Farmerbot is an optional feature developed by ThreeFold. Use at your own risk. While ThreeFold will do its best to fix any issues with the Farmerbot and minting, if minting is affected by the use of the Farmerbot, ThreeFold cannot be held responsible. Do you wish to proceed? (yes/no)\n\033[0m"
	_, err := fmt.Print(disclaimer)
	if err != nil {
		return "", err
	}
	for i := 0; i < 3; i++ {
		if _, err := fmt.Scanf("%s", &answer); err != nil {
			return "", err
		}

		answer = strings.ToLower(answer)
		if slices.Contains([]string{"yes", "no"}, answer) {
			break
		}
		if i < 2 {
			if _, err := fmt.Print("\033[33mPlease enter yes or no only: \033[0m"); err != nil {
				return "", err
			}
		}
	}
	if !slices.Contains([]string{"yes", "no"}, answer) {
		return "", fmt.Errorf("disclaimer expects yes or no, found : %s", answer)
	}
	return answer, nil

}
