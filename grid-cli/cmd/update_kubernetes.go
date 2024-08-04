// Package cmd for parsing command line arguments
package cmd

import (
	"github.com/spf13/cobra"
)

// deployCmd represents the deploy command
var updatekubernetesCmd = &cobra.Command{
	Use:   "kubernetes",
	Short: "update kubernetes workers",
}

func init() {
	updateCmd.AddCommand(updatekubernetesCmd)
}
