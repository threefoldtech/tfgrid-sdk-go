// Package cmd for parsing command line arguments
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/internal/parser"
	deployer "github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/pkg/mass-deployer"
	"golang.org/x/sys/unix"
)

const (
	ymlExt  = ".yml"
	yamlExt = ".yaml"
	jsonExt = ".json"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploy groups of vms in configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		// It doesn't have a subcommand
		if len(cmd.Flags().Args()) != 0 {
			return fmt.Errorf("'deploy' and %v cannot be used together, please use one command at a time", cmd.Flags().Args())
		}

		debug, err := cmd.Flags().GetBool("debug")
		if err != nil {
			return fmt.Errorf("invalid log debug mode input '%v' with error: %w", debug, err)
		}

		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		if debug {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}

		outputPath, err := cmd.Flags().GetString("output")
		if err != nil {
			return fmt.Errorf("error in output file: %w", err)
		}

		outJsonFmt := filepath.Ext(outputPath) == jsonExt
		outYmlFmt := filepath.Ext(outputPath) == yamlExt || filepath.Ext(outputPath) == ymlExt
		if outputPath != "" {
			if !outJsonFmt && !outYmlFmt {
				return fmt.Errorf("unsupported output file format '%s', should be [yaml, yml, json]", outputPath)
			}

			_, err := os.Stat(outputPath)
			// check if output file is writable
			if !errors.Is(err, os.ErrNotExist) && unix.Access(outputPath, unix.W_OK) != nil {
				return fmt.Errorf("output path '%s' is not writable", outputPath)
			}
		}

		configPath, err := cmd.Flags().GetString("config")
		if err != nil {
			return fmt.Errorf("error in configuration file: %w", err)
		}

		if configPath == "" {
			return fmt.Errorf("required configuration file path is empty")
		}

		configFile, err := os.Open(configPath)
		if err != nil {
			return fmt.Errorf("failed to open configuration file '%s' with error: %w", configPath, err)
		}
		defer configFile.Close()

		jsonFmt := filepath.Ext(configPath) == jsonExt
		ymlFmt := filepath.Ext(configPath) == yamlExt || filepath.Ext(configPath) == ymlExt
		if !jsonFmt && !ymlFmt {
			return fmt.Errorf("unsupported configuration file format '%s', should be [yaml, yml, json]", configPath)
		}

		cfg, err := parser.ParseConfig(configFile, jsonFmt)
		if err != nil {
			return fmt.Errorf("failed to parse configuration file '%s' with error: %w", configPath, err)
		}

		tfPluginClient, err := setup(cfg, debug)
		if err != nil {
			return err
		}

		if err = parser.ValidateConfig(cfg, tfPluginClient); err != nil {
			return fmt.Errorf("failed to validate configuration file '%s' with error: %w", configPath, err)
		}

		ctx := context.Background()
		if err = deployer.RunDeployer(ctx, cfg, tfPluginClient, outputPath, debug); err != nil {
			return fmt.Errorf("failed to run the deployer with error: %w", err)
		}

		return nil
	},
}
