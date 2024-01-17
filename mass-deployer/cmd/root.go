package cmd

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/internal/parser"
	deployer "github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/pkg/mass-deployer"
)

var rootCmd = &cobra.Command{
	Use:   "mass-deployer",
	Short: "A tool for deploying groups of vms on Threefold Grid",

	Run: func(cmd *cobra.Command, args []string) {
		configPath, err := cmd.Flags().GetString("config")
		if err != nil || configPath == "" {
			log.Fatal().Err(err).Msg("error in config file")
			return
		}

		configFile, err := os.ReadFile(configPath)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to open config file")
		}

		cfg, err := parser.ParseConfig(configFile)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to parse config file")
		}

		err = deployer.RunDeployer(cfg)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to run the deployer")
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	rootCmd.Flags().StringP("config", "c", "", "path to config file")
}
