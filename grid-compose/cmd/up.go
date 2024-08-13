package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "deploy application on the grid",
	Run: func(cmd *cobra.Command, args []string) {
		if err := app.Up(cmd.Context()); err != nil {
			log.Fatal().Err(err).Send()
		}
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
}
