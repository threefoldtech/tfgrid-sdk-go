package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "list containers",
	Run: func(cmd *cobra.Command, args []string) {
		if err := app.Ps(cmd.Context()); err != nil {
			log.Fatal().Err(err).Send()
		}
	},
}
