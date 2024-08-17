package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/app"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "cancel your project on the grid",
	Run: func(cmd *cobra.Command, args []string) {
		app := cmd.Context().Value("app").(*app.App)
		if err := app.Down(); err != nil {
			log.Fatal().Err(err).Send()
		}
	},
}
