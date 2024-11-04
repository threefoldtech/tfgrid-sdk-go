// Package cmd for parsing command line arguments
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/tfrobot/internal/parser"
	"github.com/threefoldtech/tfgrid-sdk-go/tfrobot/pkg/deployer"
)

var cancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "cancel all deployments of configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		// It doesn't have a subcommand
		if len(cmd.Flags().Args()) != 0 {
			return fmt.Errorf("'cancel' and %v cannot be used together, please use one command at a time", cmd.Flags().Args())
		}

		debug, err := cmd.Flags().GetBool("debug")
		if err != nil {
			return fmt.Errorf("invalid log debug mode input '%v' with error: %w", debug, err)
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

		err = deployer.RunCanceler(cfg, tfPluginClient, debug)
		if err != nil {
			return fmt.Errorf("failed to cancel configured deployments with error: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(cancelCmd)

	cancelCmd.Flags().BoolP("debug", "d", false, "allow debug logs")
	cancelCmd.Flags().StringP("config", "c", "", "path to config file")
}
