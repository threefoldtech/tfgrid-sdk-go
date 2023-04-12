// Package cmd for parsing command line arguments
package cmd

import (
	"math/rand"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gridify",
	Short: "A tool to deploy projects on threefold grid",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).Level(zerolog.InfoLevel).
		With().
		Timestamp().
		Logger()

	rand.Seed(time.Now().UnixNano())

	rootCmd.PersistentFlags().BoolP("debug", "d", false, "show debug level logs")
}
