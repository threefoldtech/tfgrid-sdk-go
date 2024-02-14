package cmd

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tfrobot",
	Short: "A tool for deploying groups of vms on Threefold Grid",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(cancelCmd)
	rootCmd.AddCommand(loadCmd)

	err := rootCmd.Execute()
	if err != nil {
		log.Fatal().Err(err).Send()
	}
}

func init() {
	deployCmd.Flags().BoolP("debug", "d", false, "allow debug logs")
	deployCmd.Flags().StringP("config", "c", "", "path to config file")
	deployCmd.Flags().StringP("output", "o", "output.yaml", "path to output file")

	loadCmd.Flags().BoolP("debug", "d", false, "allow debug logs")
	loadCmd.Flags().StringP("config", "c", "", "path to config file")
	loadCmd.Flags().StringP("output", "o", "output.yaml", "path to output file")

	cancelCmd.Flags().BoolP("debug", "d", false, "allow debug logs")
	cancelCmd.Flags().StringP("config", "c", "", "path to config file")
}
