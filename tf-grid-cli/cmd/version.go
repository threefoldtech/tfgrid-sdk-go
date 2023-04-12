// Package cmd for parsing command line arguments
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// set at build time
var commit string
var version string

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Get latest build tag",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
		fmt.Println(commit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
