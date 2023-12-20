package cmd

import (
	"os"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/monitoring-bot/internal"
)

var rootCmd = &cobra.Command{
	Use:   "monitoring-bot",
	Short: "monitor bot for tfgrid wallets",
	RunE: func(cmd *cobra.Command, args []string) error {
		debug, err := cmd.Flags().GetBool("debug")
		if err != nil {
			return errors.Wrap(err, "invalid debug flag")
		}

		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		if debug {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

		envPath, err := cmd.Flags().GetString("env")
		if err != nil {
			return errors.Wrap(err, "invalid env file")
		}

		if len(envPath) == 0 {
			return errors.Wrap(err, "env file is missing")
		}

		walletsPath, err := cmd.Flags().GetString("wallets")
		if err != nil {
			return errors.Wrap(err, "invalid wallets json file")
		}

		if len(walletsPath) == 0 {
			return errors.Wrap(err, "wallets json file is missing")
		}

		envContent, err := internal.ReadFile(envPath)
		if err != nil {
			return errors.Wrap(err, "failed to read env file")
		}

		env, err := internal.ParseEnv(string(envContent))
		if err != nil {
			return errors.Wrap(err, "failed to parse env content")
		}

		walletsContent, err := internal.ReadFile(walletsPath)
		if err != nil {
			return errors.Wrap(err, "failed to read wallets file")
		}

		wallets, err := internal.ParseJSONIntoWallets(walletsContent)
		if err != nil {
			return errors.Wrap(err, "failed to parse wallets content")
		}

		monitor, err := internal.NewMonitor(env, wallets)
		if err != nil {
			return errors.Wrap(err, "failed to create a new monitor")
		}

		return monitor.Start(cmd.Context())
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Send()
	}
}

func init() {
	cobra.OnInitialize()

	rootCmd.Flags().StringP("env", "e", ".env", "Enter your env path")
	rootCmd.Flags().StringP("wallets", "w", "wallets.json", "Enter your wallets json file path")
	rootCmd.Flags().BoolP("debug", "d", false, "Enable if you want to debug")
}
