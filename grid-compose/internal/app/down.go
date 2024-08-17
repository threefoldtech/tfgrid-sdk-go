package app

import (
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/generator"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/types"
)

// Down cancels all the deployments
func (a *App) Down() error {
	if a.Config.Networks == nil {
		a.Config.Networks[generator.GenerateDefaultNetworkName(a.Config.Services)] = types.Network{}
	}
	for networkName := range a.Config.Networks {
		projectName := a.getProjectName(networkName)
		log.Info().Str("projectName", projectName).Msg("canceling deployments")
		if err := a.Client.CancelByProjectName(projectName); err != nil {
			return err
		}
	}
	return nil
}
