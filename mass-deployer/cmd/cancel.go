// Package cmd for parsing command line arguments
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/internal/parser"
	deployer "github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/pkg/mass-deployer"
)

var cancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "cancel all deployments of configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		debug, err := cmd.Flags().GetBool("debug")
		if err != nil {
			return fmt.Errorf("invalid log debug mode input '%v' with error: %w", debug, err)
		}

		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		if debug {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}

		configPath, err := cmd.Flags().GetString("config")
		if err != nil {
			return fmt.Errorf("error in configuration file: %w", err)
		}

		if configPath == "" {
			return fmt.Errorf("configuration file path is empty")
		}

		configFile, err := os.Open(configPath)
		if err != nil {
			return fmt.Errorf("failed to open configuration file '%s' with error: %w", configPath, err)
		}
		defer configFile.Close()

		jsonFmt := filepath.Ext(configPath) == ".json"
		cfg, err := parser.ParseConfig(configFile, jsonFmt)
		if err != nil {
			return fmt.Errorf("failed to parse configuration file '%s' with error: %w", configPath, err)
		}

		err = deployer.RunCanceler(cfg)
		if err != nil {
			return fmt.Errorf("failed to cancel configured deployments with error: %w", err)
		}

		return nil
	},
}
