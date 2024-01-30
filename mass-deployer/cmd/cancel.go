// Package cmd for parsing command line arguments
package cmd

import (
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/internal/parser"
	deployer "github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/pkg/mass-deployer"
)

var cancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "cancel all deployments of configuration file",
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
		jsonFmt := filepath.Ext(configPath) == ".json"

		cfg, err := parser.ParseConfig(configFile, jsonFmt)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to parse config file: %s", configPath)
		}

		err = deployer.RunCanceler(cfg)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to run the canceler")
		}
	},
}

func init() {
	cancelCmd.Flags().StringP("config", "c", "", "path to config file")
	rootCmd.AddCommand(cancelCmd)
}
