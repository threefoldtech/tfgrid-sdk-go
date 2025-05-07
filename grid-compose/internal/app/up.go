package app

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/app/dependency"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/convert"
)

// Up deploys the services described in the config file
func (a *App) Up(ctx context.Context) error {
	defaultNetName := a.GenerateDefaultNetworkName()
	deploymentData, err := convert.ConvertConfigToDeploymentData(ctx, a.Client, a.Config, defaultNetName)
	if err != nil {
		return err
	}

	networks := buildNetworks(deploymentData.NetworkNodeMap, a.Config.Networks, defaultNetName, a.GetProjectName)

	resolvedServices, err := deploymentData.ServicesGraph.ResolveDependencies(deploymentData.ServicesGraph.Root, []*dependency.DRNode{}, []*dependency.DRNode{})
	if err != nil {
		return err
	}

	// maybe add a deployed field for both services and networks instead of using maps and slices
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

		err = assignMyCeliumKeys(network, vm.MyceliumIPSeed)
		if err != nil {
			return err
		}

		disks, mounts, err := buildStorage(service.Volumes, a.Config.Volumes)
		if err != nil {
			return err
		}
		vm.Mounts = mounts

		if err := vm.Validate(); err != nil {
			return rollback(ctx, a.Client, deployedDls, deployedNets, err)
		}

		if _, ok := deployedNets[network.Name]; !ok {
			deployedNets[network.Name] = network
			log.Info().Str("name", network.Name).Uint32("node_id", network.Nodes[0]).Msg("deploying network...")
			if err := network.Validate(); err != nil {
				return rollback(ctx, a.Client, deployedDls, deployedNets, err)
			}

			if err := a.Client.NetworkDeployer.Deploy(ctx, network); err != nil {
				return rollback(ctx, a.Client, deployedDls, deployedNets, err)
			}
			log.Info().Msg("deployed successfully")
		}

		deployedDl, err := deployVM(ctx, a.Client, vm, disks, network, a.GetDeploymentName(serviceName), service.HealthCheck)
		deployedDls = append(deployedDls, &deployedDl)
		if err != nil {
			return rollback(ctx, a.Client, deployedDls, deployedNets, err)
		}
	}

	log.Info().Msg("all deployments deployed successfully")

	return nil
}
