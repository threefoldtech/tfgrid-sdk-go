package grid

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

// RevertDeployment cleans up a deployment and network
func RevertDeployment(ctx context.Context, tfPlugin deployer.TFPluginClient, dl *workloads.Deployment, network *workloads.ZNet, deleteVM bool) {
	if deleteVM {
		log.Debug().Msg("Cleaning up deployment")
		if err := tfPlugin.DeploymentDeployer.Cancel(ctx, dl); err != nil {
			log.Error().Err(err).Msg("Failed to cancel deployment")
		}
	}
	if err := tfPlugin.NetworkDeployer.Cancel(ctx, network); err != nil {
		log.Error().Err(err).Msg("Failed to cancel network")
	}
}

// RevertDeploymentLight cleans up a light deployment and network
func RevertDeploymentLight(ctx context.Context, tfPlugin deployer.TFPluginClient, dl *workloads.Deployment, network *workloads.ZNetLight, deleteVM bool) {
	if deleteVM {
		log.Debug().Msg("Cleaning up deployment light")
		if err := tfPlugin.DeploymentDeployer.Cancel(ctx, dl); err != nil {
			log.Error().Err(err).Msg("Failed to cancel deployment")
		}
	}
	if err := tfPlugin.NetworkDeployer.Cancel(ctx, network); err != nil {
		log.Error().Err(err).Msg("Failed to cancel network")
	}
}
