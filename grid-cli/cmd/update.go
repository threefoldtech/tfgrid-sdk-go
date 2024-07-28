// Package cmd for parsing command line arguments
package cmd

import (
	"github.com/spf13/cobra"
)

// deployCmd represents the deploy command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "update resources in Threefold grid",
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
