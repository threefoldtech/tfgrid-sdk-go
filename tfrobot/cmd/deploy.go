// Package cmd for parsing command line arguments
package cmd

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/tfrobot/internal/parser"
	"github.com/threefoldtech/tfgrid-sdk-go/tfrobot/pkg/deployer"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploy groups of vms in configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		// It doesn't have a subcommand
		if len(cmd.Flags().Args()) != 0 {
			return fmt.Errorf("'deploy' and %v cannot be used together, please use one command at a time", cmd.Flags().Args())
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
		noColor, err := cmd.Flags().GetBool("no-color")
		if err != nil {
			return err
		}

		if err = checkOutputFile(outputPath); err != nil {
			return err
		}

		cfg, err := readConfig(configPath)
		if err != nil {
			return err
		}

		tfPluginClient, err := setup(cfg, debug, noColor)
		if err != nil {
			return err
		}

		if err = parser.ValidateConfig(cfg, tfPluginClient); err != nil {
			return errors.Wrapf(err, "failed to validate configuration file '%s' with error", configPath)
		}

		if errs := deployer.RunDeployer(context.Background(), cfg, tfPluginClient, outputPath, debug); errs != nil {
			return errors.Wrap(err, "failed to run deployer")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().BoolP("debug", "d", false, "allow debug logs")
	deployCmd.Flags().StringP("config", "c", "", "path to config file")
	deployCmd.Flags().StringP("output", "o", "output.yaml", "path to output file")
	deployCmd.Flags().Bool("no-color", false, "disable output styling")
}
