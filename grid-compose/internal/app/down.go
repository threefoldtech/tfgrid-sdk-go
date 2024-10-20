package app

import (
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/types"
)

// Down cancels all the deployments on the grid
// TODO: remove known hosts
func (a *App) Down() error {
	if len(a.Config.Networks) == 0 {
		a.Config.Networks[a.GenerateDefaultNetworkName()] = types.Network{}
	}
	for networkName := range a.Config.Networks {
		projectName := a.GetProjectName(networkName)
		log.Info().Str("projectName", projectName).Msg("canceling deployments")
		if err := a.Client.CancelByProjectName(projectName); err != nil {
			return err
		}
	}
	return nil
}
