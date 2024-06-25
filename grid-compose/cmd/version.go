package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "v0.0.1"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "get current version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}
