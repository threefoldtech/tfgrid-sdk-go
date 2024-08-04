package cmd

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal"
)

var (
	app        *internal.App
	configPath string
	network    string
	mnemonic   string
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Send()
	}
}

var rootCmd = &cobra.Command{
	Use:   "grid-compose",
	Short: "Grid-Compose is a tool for running multi-vm applications on TFGrid defined using a Yaml formatted file.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		var err error
		app, err = internal.NewApp(network, mnemonic, configPath)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
	},
}

func init() {
	network = os.Getenv("NETWORK")
	mnemonic = os.Getenv("MNEMONIC")
	rootCmd.PersistentFlags().StringVarP(&configPath, "file", "f", "./grid-compose.yaml", "the grid-compose configuration file")
	psCmd.PersistentFlags().BoolP("verbose", "v", false, "all information about deployed services")
	psCmd.PersistentFlags().StringP("output", "o", "", "output result to a file")

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(psCmd)
	rootCmd.AddCommand(downCmd)
}
