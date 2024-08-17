package convert

import (
	"fmt"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/config"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/generator"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/types"
)

func ConvertToDeploymentData(config *config.Config) (*types.DeploymentData, error) {
	deploymentData := &types.DeploymentData{
		NetworkNodeMap: make(map[string]*types.NetworkData, 0),
		ServicesGraph:  types.NewDRGraph(types.NewDRNode("root", nil)),
	}

	defaultNetName := generator.GenerateDefaultNetworkName(config.Services)

	for serviceName, service := range config.Services {
		svc := service

		var netName string
		if svc.Network == "" {
			netName = defaultNetName
		} else {
			netName = svc.Network
		}

		if _, ok := deploymentData.NetworkNodeMap[netName]; !ok {
			deploymentData.NetworkNodeMap[netName] = &types.NetworkData{
				NodeID:   svc.NodeID,
				Services: make(map[string]*types.Service, 0),
			}
		}

		if deploymentData.NetworkNodeMap[netName].NodeID == 0 && svc.NodeID != 0 {
			deploymentData.NetworkNodeMap[netName].NodeID = svc.NodeID
		}

		if svc.NodeID != 0 && svc.NodeID != deploymentData.NetworkNodeMap[netName].NodeID {
			return nil, fmt.Errorf("service name %s node_id %d should be the same for all or some or left blank for services in the same network", serviceName, svc.NodeID)
		}

		deploymentData.NetworkNodeMap[netName].Services[serviceName] = &svc

		svcNode, ok := deploymentData.ServicesGraph.Nodes[serviceName]
		if !ok {
			svcNode = types.NewDRNode(
				serviceName,
				&svc,
			)

			deploymentData.ServicesGraph.AddNode(serviceName, svcNode)
		}

		svcRootNode := deploymentData.ServicesGraph.Root

		for _, dep := range svc.DependsOn {
			if _, ok := config.Services[dep]; !ok {
				return nil, fmt.Errorf("service %s depends on %s which does not exist", serviceName, dep)
			}

			depNode, ok := deploymentData.ServicesGraph.Nodes[dep]
			if !ok {
				depService := config.Services[dep]
				depNode = types.NewDRNode(dep, &depService)
			}

			svcNode.AddDependency(depNode)
			depNode.Parent = svcNode
			deploymentData.ServicesGraph.AddNode(dep, depNode)
		}

		if svcNode.Parent == nil {
			svcNode.Parent = svcRootNode
			svcRootNode.AddDependency(svcNode)
		}
	}

	// for netName, data := range deploymentData.NetworkNodeMap {
	// 	log.Println(netName)
	// 	log.Println(data.NodeID)

	// 	for svcName := range data.Services {
	// 		log.Println(svcName)
	// 	}

	// }

	// resolvedServices, err := deploymentData.ServicesGraph.ResolveDependencies(deploymentData.ServicesGraph.Root, []*types.DRNode{}, []*types.DRNode{})

	// if err != nil {
	// 	return nil, err
	// }

	// for _, svc := range resolvedServices {
	// 	log.Println(svc.Name)
	// }
	return deploymentData, nil
}
