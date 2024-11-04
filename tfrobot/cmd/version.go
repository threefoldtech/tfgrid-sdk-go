// Package cmd for parsing command line arguments
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// set at build time
var (
	commit  string
	version string
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Get latest build tag",
	RunE: func(cmd *cobra.Command, args []string) error {
		// It doesn't have a subcommand
		if len(cmd.Flags().Args()) != 0 {
			return fmt.Errorf("'version' and %v cannot be used together, please use one command at a time", cmd.Flags().Args())
		}

		fmt.Printf("version: %s", version)
		fmt.Printf("commit: %s", commit)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
