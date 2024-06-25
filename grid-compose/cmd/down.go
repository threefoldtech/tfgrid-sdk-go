package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "cancel your project on the grid",
	Run: func(cmd *cobra.Command, args []string) {
		if err := down(); err != nil {
			log.Fatal().Err(err).Send()
		}
	},
}

func down() error {
	for key := range app.Specs.Services {
		projectName := internal.GetProjectName(key, app.Client.TwinID)
		log.Info().Str("projectName", projectName).Msg("canceling deployments")
		if err := app.Client.CancelByProjectName(projectName); err != nil {
			return err
		}
	}
	return nil
}
