// Package cmd for parsing command line arguments
package cmd

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/tfrobot/pkg/deployer"
)

var loadCmd = &cobra.Command{
	Use:   "load",
	Short: "load deployments of configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		// It doesn't have a subcommand
		if len(cmd.Flags().Args()) != 0 {
			return fmt.Errorf("'load' and %v cannot be used together, please use one command at a time", cmd.Flags().Args())
		}

		configPath, err := cmd.Flags().GetString("config")
		if err != nil {
			return errors.Wrap(err, "error in configuration file")
		}
		debug, err := cmd.Flags().GetBool("debug")
		if err != nil {
			return errors.Wrapf(err, "invalid log debug mode input '%v'", debug)
		}
		outputPath, err := cmd.Flags().GetString("output")
		if err != nil {
			return errors.Wrap(err, "error in output file")
		}

		if err = checkOutputFile(outputPath); err != nil {
			return err
		}

		cfg, err := readConfig(configPath)
		if err != nil {
			return err
		}

		tfPluginClient, err := setup(cfg, debug)
		if err != nil {
			return err
		}

		if err := deployer.RunLoader(context.Background(), cfg, tfPluginClient, debug, outputPath); err != nil {
			return errors.Wrap(err, "failed to load configured deployments")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(loadCmd)

	loadCmd.Flags().BoolP("debug", "d", false, "allow debug logs")
	loadCmd.Flags().StringP("config", "c", "", "path to config file")
	loadCmd.Flags().StringP("output", "o", "output.yaml", "path to output file")
}
