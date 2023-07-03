// Package cmd to make it cmd app
/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/activation-service/app"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "activation-service",
	Short: "activation-service activates new TFChain wallet addresses by depositing a minimal amount of TFT (currently 1 TFT).",
	RunE: func(cmd *cobra.Command, args []string) error {
		configFile, err := cmd.Flags().GetString("config")
		if err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}

		app, err := app.NewApp(cmd.Context(), configFile)
		if err != nil {
			return fmt.Errorf("failed to create new app: %w", err)
		}

		err = app.Start(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to start app: %w", err)
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	err := rootCmd.Execute()
	if err != nil {
		log.Err(err).Send()
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringP("config", "c", "./.env", "Enter your env configurations path")
}
