// Package cmd to make it cmd app
/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Commit set at build time
var Commit string

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Get current build commit",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Commit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
