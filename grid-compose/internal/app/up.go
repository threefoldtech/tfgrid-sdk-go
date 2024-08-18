package app

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/convert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/deploy"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/types"
)

// Up deploys the services described in the config file
func (a *App) Up(ctx context.Context) error {
	deploymentData, err := convert.ConvertConfigToDeploymentData(ctx, a.Client, a.Config)
	if err != nil {
		return err
	}

	networks := deploy.BuildNetworks(deploymentData.NetworkNodeMap, a.Config.Networks, deploy.GenerateDefaultNetworkName(a.Config.Services), a.GetProjectName)

	resolvedServices, err := deploymentData.ServicesGraph.ResolveDependencies(deploymentData.ServicesGraph.Root, []*types.DRNode{}, []*types.DRNode{})
	if err != nil {
		return err
	}

	deployedDls := make([]*workloads.Deployment, 0)
	deployedNets := make([]*workloads.ZNet, 0)
	for _, resService := range resolvedServices {
		if resService.Name == "root" {
			continue
		}

		serviceName := resService.Name
		service := deploymentData.ServicesGraph.Nodes[serviceName].Service
		var network *workloads.ZNet

		if service.Network == "" {
			network = networks[deploy.GenerateDefaultNetworkName(a.Config.Services)]
			service.Network = network.Name
		} else {
			network = networks[service.Network]
		}

		vm := &workloads.VM{
			Name:        serviceName,
			Flist:       service.Flist,
			Entrypoint:  service.Entrypoint,
			CPU:         int(service.Resources.CPU),
			Memory:      int(service.Resources.Memory),
			RootfsSize:  int(service.Resources.Rootfs),
			NetworkName: network.Name,
		}

		if err := deploy.BuildVM(vm, service); err != nil {
			return err
		}

		disks, err := deploy.BuildDisks(vm, service.Volumes, a.Config.Volumes)
		if err != nil {
			return err
		}

		deployedDl, err := deploy.DeployVM(ctx, a.Client, *vm, disks, network, a.GetDeploymentName(serviceName), *service.HealthCheck)

		deployedDls = append(deployedDls, &deployedDl)
		deployedNets = append(deployedNets, network)
		if err != nil {
			log.Info().Msg("an error occurred while deploying the deployment, canceling all deployments")
			log.Info().Msg("canceling networks...")
			for _, network := range deployedNets {
				if err := a.Client.NetworkDeployer.Cancel(ctx, network); err != nil {
					return err
				}
			}

			log.Info().Msg("canceling deployments...")
			for _, deployment := range deployedDls {
				if err := a.Client.DeploymentDeployer.Cancel(ctx, deployment); err != nil {
					return err
				}
			}

			log.Info().Msg("all deployments canceled successfully")
			return err
		}
	}

	log.Info().Msg("all deployments deployed successfully")

	return nil
}
