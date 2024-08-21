package app

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/app/dependency"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/convert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/deploy"
)

// Up deploys the services described in the config file
func (a *App) Up(ctx context.Context) error {
	deploymentData, err := convert.ConvertConfigToDeploymentData(ctx, a.Client, a.Config)
	if err != nil {
		return err
	}

	defaultNetName := deploy.GenerateDefaultNetworkName(a.Config.Services)
	networks := deploy.BuildNetworks(deploymentData.NetworkNodeMap, a.Config.Networks, defaultNetName, a.GetProjectName)

	resolvedServices, err := deploymentData.ServicesGraph.ResolveDependencies(deploymentData.ServicesGraph.Root, []*dependency.DRNode{}, []*dependency.DRNode{})
	if err != nil {
		return err
	}

	deployedDls := make([]*workloads.Deployment, 0)
	deployedNets := make(map[string]*workloads.ZNet, 0)
	for _, resService := range resolvedServices {
		serviceName := resService.Name
		if serviceName == "root" {
			continue
		}

		service := a.Config.Services[serviceName]

		var network *workloads.ZNet
		if service.Network == "" {
			network = networks[defaultNetName]
			service.Network = network.Name
		} else {
			network = networks[service.Network]
		}

		vm, err := convert.ConvertServiceToVM(&service, serviceName, network.Name)
		if err != nil {
			return err
		}

		err = deploy.AssignMyCeliumKeys(network, vm.MyceliumIPSeed)
		if err != nil {
			return err
		}

		disks, mounts, err := deploy.BuildStorage(service.Volumes, a.Config.Volumes)
		if err != nil {
			return err
		}
		vm.Mounts = mounts

		if _, ok := deployedNets[network.Name]; !ok {
			deployedNets[network.Name] = network
			log.Info().Str("name", network.Name).Uint32("node_id", network.Nodes[0]).Msg("deploying network...")
			if err := a.Client.NetworkDeployer.Deploy(ctx, network); err != nil {
				return rollback(ctx, a.Client, deployedDls, deployedNets, err)
			}
			log.Info().Msg("deployed successfully")
		}

		deployedDl, err := deploy.DeployVM(ctx, a.Client, vm, disks, network, a.GetDeploymentName(serviceName), service.HealthCheck)
		deployedDls = append(deployedDls, &deployedDl)
		if err != nil {
			return rollback(ctx, a.Client, deployedDls, deployedNets, err)
		}
	}

	log.Info().Msg("all deployments deployed successfully")

	return nil
}

func rollback(ctx context.Context, client *deployer.TFPluginClient, deployedDls []*workloads.Deployment, deployedNets map[string]*workloads.ZNet, err error) error {
	log.Info().Msg("an error occurred while deploying, canceling all deployments")
	log.Info().Msg("canceling networks...")
	for _, network := range deployedNets {
		if err := client.NetworkDeployer.Cancel(ctx, network); err != nil {
			return err
		}
	}

	log.Info().Msg("canceling deployments...")
	for _, deployment := range deployedDls {
		if err := client.DeploymentDeployer.Cancel(ctx, deployment); err != nil {
			return err
		}
	}

	log.Info().Msg("all deployments canceled successfully")
	return err
}
