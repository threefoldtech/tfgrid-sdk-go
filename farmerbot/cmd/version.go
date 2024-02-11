// Package cmd for farmerbot commands
/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/version"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "get farmerbot latest version and commit",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.Version)
	},
}
