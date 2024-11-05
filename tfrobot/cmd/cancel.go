// Package cmd for parsing command line arguments
package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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
		configPath, err := cmd.Flags().GetString("config")
		if err != nil {
			return errors.Wrap(err, "error in configuration file")
		}
		debug, err := cmd.Flags().GetBool("debug")
		if err != nil {
			return errors.Wrapf(err, "invalid log debug mode input '%v'", debug)
		}
		noColor, err := cmd.Flags().GetBool("no-color")
		if err != nil {
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

		err = deployer.RunCanceler(cfg, tfPluginClient, debug)
		if err != nil {
			return errors.Wrap(err, "failed to cancel configured deployments")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(cancelCmd)

	cancelCmd.Flags().BoolP("debug", "d", false, "allow debug logs")
	cancelCmd.Flags().StringP("config", "c", "", "path to config file")
	cancelCmd.Flags().BoolP("no-color", "n", false, "disable output styling")
}
