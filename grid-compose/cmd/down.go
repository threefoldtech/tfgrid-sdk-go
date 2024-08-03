package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "cancel your project on the grid",
	Run: func(cmd *cobra.Command, args []string) {
		if err := app.Down(); err != nil {
			log.Fatal().Err(err).Send()
		}
	},
}
