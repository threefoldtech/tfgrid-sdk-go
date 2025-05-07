package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/app"
)

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "list deployments on the grid",
	Run: func(cmd *cobra.Command, args []string) {
		verbose, err := cmd.Flags().GetBool("verbose")
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		app, ok := cmd.Context().Value("app").(*app.App)
		if !ok {
			log.Fatal().Msg("app not found in context")
		}

		if err := app.Ps(cmd.Context(), verbose); err != nil {
			log.Fatal().Err(err).Send()
		}
	},
}
