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
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("version: %s", version)
		fmt.Printf("commit: %s", commit)
	},
}
