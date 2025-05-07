package convert

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/app/dependency"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/types"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/pkg/parser/config"
	proxy_types "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

const (
	minCPU    = 1
	minMemory = 2048
	minRootfs = 2048
)

// ConvertConfigToDeploymentData converts the config to deployment data that will be used to deploy the services
func ConvertConfigToDeploymentData(ctx context.Context, client *deployer.TFPluginClient, config *config.Config, defaultNetName string) (*types.DeploymentData, error) {
	deploymentData := &types.DeploymentData{
		NetworkNodeMap: make(map[string]*types.NetworkData, 0),
		ServicesGraph:  dependency.NewDRGraph(dependency.NewDRNode("root")),
	}

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
			svcNode = dependency.NewDRNode(
				serviceName,
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
				depNode = dependency.NewDRNode(dep)
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

	if err := getMissingNodes(ctx, deploymentData.NetworkNodeMap, client); err != nil {
		return nil, err
	}

	return deploymentData, nil
}

// ConvertServiceToVM converts the service to a the VM workload that will be used to deploy a virtual machine on the grid
func ConvertServiceToVM(service *types.Service, serviceName, networkName string) (workloads.VM, error) {
	vm := workloads.VM{
		Name:        serviceName,
		Flist:       service.Flist,
		Entrypoint:  service.Entrypoint,
		CPU:         int(service.Resources.CPU),
		Memory:      int(service.Resources.Memory),
		RootfsSize:  int(service.Resources.Rootfs),
		NetworkName: networkName,
	}

	if vm.RootfsSize == 0 {
		vm.RootfsSize = minRootfs
	}
	if vm.CPU == 0 {
		vm.CPU = minCPU
	}
	if vm.Memory == 0 {
		vm.Memory = minMemory
	}

	if err := assignNetworksTypes(&vm, service.IPTypes); err != nil {
		return workloads.VM{}, err
	}
	return vm, nil
}

// getMissingNodes gets the missing nodes for the deployment data.
// It filters the nodes based on the resources required by the services in one network.
// TODO: Calculate total MRU and SRU while populating the deployment data
func getMissingNodes(ctx context.Context, networkNodeMap map[string]*types.NetworkData, client *deployer.TFPluginClient) error {
	for _, deploymentData := range networkNodeMap {
		if deploymentData.NodeID != 0 {
			continue
		}

		// freeCRU is not in NodeFilter?
		var freeMRU, freeSRU uint64

		for _, service := range deploymentData.Services {
			freeMRU += service.Resources.Memory
			freeSRU += service.Resources.Rootfs
		}

		filter := proxy_types.NodeFilter{
			Status:  []string{"up"},
			FreeSRU: &freeSRU,
			FreeMRU: &freeMRU,
		}

		nodes, _, err := client.GridProxyClient.Nodes(ctx, filter, proxy_types.Limit{})
		if err != nil {
			return err
		}

		if len(nodes) == 0 || (len(nodes) == 1 && nodes[0].NodeID == 1) {
			return fmt.Errorf("no available nodes")
		}

		// TODO: still need to agree on logic to select the node
		for _, node := range nodes {
			if node.NodeID != 1 {
				deploymentData.NodeID = uint32(node.NodeID)
				break
			}
		}
	}

	return nil
}

func assignNetworksTypes(vm *workloads.VM, ipTypes []string) error {
	for _, ipType := range ipTypes {
		switch ipType {
		case "ipv4":
			vm.PublicIP = true
		case "ipv6":
			vm.PublicIP6 = true
		case "ygg":
			vm.Planetary = true
		case "myc":
			seed, err := getRandomMyceliumIPSeed()
			if err != nil {
				return fmt.Errorf("failed to get mycelium seed %w", err)
			}
			vm.MyceliumIPSeed = seed
		}
	}

	return nil
}

func getRandomMyceliumIPSeed() ([]byte, error) {
	key := make([]byte, zos.MyceliumIPSeedLen)
	_, err := rand.Read(key)
	return key, err
}
