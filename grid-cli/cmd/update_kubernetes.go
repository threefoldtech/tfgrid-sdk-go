// Package cmd for parsing command line arguments
package cmd

import (
	"github.com/spf13/cobra"
)

// updateKubernetesCmd represents the update kubernetes command
var updateKubernetesCmd = &cobra.Command{
	Use:   "kubernetes",
	Short: "update kubernetes workers",
}

func init() {
	updateCmd.AddCommand(updateKubernetesCmd)
}
