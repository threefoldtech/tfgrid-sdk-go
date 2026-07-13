// Package cmd for farmerbot commands
/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/threefoldtech/zos_sdk_go/farmerbot/version"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "get farmerbot latest version and commit",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(cmd.Flags().Args()) != 0 {
			return fmt.Errorf("'run' and %v cannot be used together, please use one command at a time", cmd.Flags().Args())
		}

		fmt.Println(version.Version)
		return nil
	},
}
