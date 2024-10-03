// Package cmd for parsing command line arguments
package cmd

import (
	"log"

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
		log.Println(version)
		log.Println(commit)
	},
}
