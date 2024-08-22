package cmd

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/app"
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Send()
	}
}

// TODO: validate command line arguments
var rootCmd = &cobra.Command{
	Use:   "grid-compose",
	Short: "Grid-Compose is a tool for running multi-vm applications on TFGrid defined using a Yaml formatted file.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		network := os.Getenv("NETWORK")
		mnemonic := os.Getenv("MNEMONIC")
		configPath, _ := cmd.Flags().GetString("file")

		app, err := app.NewApp(network, mnemonic, configPath)

		if err != nil {
			log.Fatal().Err(err).Send()
		}

		ctx := context.WithValue(cmd.Context(), "app", app)
		cmd.SetContext(ctx)
	},
}

func init() {
	rootCmd.PersistentFlags().StringP("file", "f", "./grid-compose.yml", "the grid-compose configuration file")

	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(versionCmd)

	psCmd.PersistentFlags().BoolP("verbose", "v", false, "all information about deployed services")
	rootCmd.AddCommand(psCmd)

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}
