package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/app"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "deploy application on the grid",
	Run: func(cmd *cobra.Command, args []string) {
		app, ok := cmd.Context().Value("app").(*app.App)
		if !ok {
			log.Fatal().Msg("app not found in context")
		}

		if err := app.Up(cmd.Context()); err != nil {
			log.Fatal().Err(err).Send()
		}
	},
}
