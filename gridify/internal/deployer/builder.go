// Package deployer for project deployment
package deployer

import (
	"fmt"
	"net"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/gridify/internal/tfplugin"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

var (
	vmFlist      = "https://hub.grid.tf/aelawady.3bot/abdulrahmanelawady-gridify-test-latest.flist"
	vmCPU        = 2
	vmMemory     = 2 // GB
	vmRootfsSize = 5 // GB
	vmEntrypoint = "/init.sh"
	vmPublicIP   = true
	vmPlanetary  = true
)

func buildNodeFilter() types.NodeFilter {
	nodeStatus := "up"
	freeMRU := uint64(vmMemory)
	freeHRU := uint64(vmRootfsSize)
	freeIPs := uint64(0)
	if vmPublicIP {
		freeIPs = 1
	}
	domain := true

	filter := types.NodeFilter{
		FarmIDs: []uint64{1},
		Status:  &nodeStatus,
		FreeMRU: &freeMRU,
		FreeHRU: &freeHRU,
		FreeIPs: &freeIPs,
		Domain:  &domain,
	}
	return filter
}

func findNode(tfPluginClient tfplugin.TFPluginClientInterface) (uint32, error) {
	filter := buildNodeFilter()
	nodes, _, err := tfPluginClient.FilterNodes(filter, types.Limit{})
	if err != nil {
		return 0, err
	}
	if len(nodes) == 0 {
		return 0, fmt.Errorf(
			"no node with free resources available using node filter: farmIDs: %v, mru: %d, hru: %d, freeips: %d, domain: %t",
			filter.FarmIDs,
			*filter.FreeMRU,
			*filter.FreeHRU,
			*filter.FreeIPs,
			*filter.Domain,
		)
	}

	node := uint32(nodes[0].NodeID)
	return node, nil
}

func buildNetwork(projectName string, node uint32) workloads.ZNet {
	networkName := randName(10)
	network := workloads.ZNet{
		Name:  networkName,
		Nodes: []uint32{node},
		IPRange: gridtypes.NewIPNet(net.IPNet{
			IP:   net.IPv4(10, 20, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}),
		SolutionType: projectName,
	}
	return network
}

func buildDeployment(networkName, projectName, repoURL string, node uint32) workloads.Deployment {
	vmName := randName(10)
	vm := workloads.VM{
		Name:       vmName,
		Flist:      vmFlist,
		CPU:        vmCPU,
		PublicIP:   vmPublicIP,
		Planetary:  vmPlanetary,
		Memory:     vmMemory * 1024,
		RootfsSize: vmRootfsSize * 1024,
		Entrypoint: vmEntrypoint,
		EnvVars: map[string]string{
			"REPO_URL": repoURL,
		},
		NetworkName: networkName,
	}

	dl := workloads.NewDeployment(vm.Name, node, projectName, nil, networkName, nil, nil, []workloads.VM{vm}, nil)
	return dl
}

func buildGateway(backend, projectName string, node uint32) workloads.GatewayNameProxy {
	subdomain := randName(10)
	gateway := workloads.GatewayNameProxy{
		NodeID:       node,
		Name:         subdomain,
		Backends:     []zos.Backend{zos.Backend(backend)},
		SolutionType: projectName,
	}
	return gateway
}

func buildPortlessBackend(ip string) string {
	publicIP := strings.Split(ip, "/")[0]
	backend := fmt.Sprintf("http://%s", publicIP)
	return backend
}
