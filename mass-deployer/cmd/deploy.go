// Package cmd for parsing command line arguments
package cmd

import (
	"context"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/internal/parser"
	deployer "github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/pkg/mass-deployer"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploy groups of vms in configuration file",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, err := cmd.Flags().GetString("config")
		if err != nil || configPath == "" {
			log.Fatal().Err(err).Msg("error in config file")
		}

		configFile, err := os.Open(configPath)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to open config file: %s", configPath)
		}
		defer configFile.Close()

		cfg, err := parser.ParseConfig(configFile)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to parse config file: %s", configPath)
		}

		ctx := context.Background()
		err = deployer.RunDeployer(cfg, ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to run the deployer")
		}
	},
}

func init() {
	deployCmd.Flags().StringP("config", "c", "", "path to config file")
	rootCmd.AddCommand(deployCmd)
}
