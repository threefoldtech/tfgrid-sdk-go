// Package cmd for monitoring cmdline
/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid_monitoring_bot/internal"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tfgrid_monitoring_bot",
	Short: "monitor bot for tfgrid wallets",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

		env, err := cmd.Flags().GetString("env")
		if err != nil {
			log.Error().Err(err).Msg("error in env")
			return
		}

		if env == "" {
			log.Error().Msg("env file is missing")
			return
		}

		wallets, err := cmd.Flags().GetString("wallets")
		if err != nil {
			log.Error().Err(err).Msg("error in env")
			return
		}

		if wallets == "" {
			log.Error().Msg("json addresses file is missing")
			return
		}

		monitor, err := internal.NewMonitor(env, wallets)
		if err != nil {
			log.Error().Err(err).Msg("failed monitoring")
			return
		}

		monitor.Start()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize()

	rootCmd.Flags().StringP("env", "e", "", "Enter your env path")
	rootCmd.Flags().StringP("wallets", "w", "", "Enter your wallets json file path")
}
