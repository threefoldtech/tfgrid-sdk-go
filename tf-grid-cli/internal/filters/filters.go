package filters

import (
	"fmt"
	"strings"

	"github.com/threefoldtech/grid3-go/workloads"
	"github.com/threefoldtech/grid_proxy_server/pkg/client"
	"github.com/threefoldtech/grid_proxy_server/pkg/types"
)

func GetAvailableNode(client client.Client, filter types.NodeFilter) (uint32, error) {
	nodes, _, err := client.Nodes(filter, types.Limit{})
	if err != nil {
		return 0, err
	}
	if len(nodes) == 0 {
		var filterStringBuilder strings.Builder
		if filter.FarmIDs != nil {
			fmt.Fprintf(&filterStringBuilder, "farmIDs: %v, ", filter.FarmIDs)
		}
		if filter.FreeMRU != nil {
			fmt.Fprintf(&filterStringBuilder, "mru: %d, ", *filter.FreeMRU)
		}
		if filter.FreeSRU != nil {
			fmt.Fprintf(&filterStringBuilder, "sru: %d, ", *filter.FreeSRU)
		}
		if filter.FreeIPs != nil {
			fmt.Fprintf(&filterStringBuilder, "freeips: %d, ", *filter.FreeIPs)
		}
		if filter.Domain != nil {
			fmt.Fprintf(&filterStringBuilder, "domain: %t, ", *filter.Domain)
		}
		filterString := filterStringBuilder.String()
		return 0, fmt.Errorf("no node with free resources available using node filter: %s", filterString[:len(filterString)-2])
	}

	node := uint32(nodes[0].NodeID)
	return node, nil
}

func BuildK8sFilter(k8sNode workloads.K8sNode, farmID uint64, k8sNodesNum uint) types.NodeFilter {
	freeMRUs := uint64(k8sNode.Memory*int(k8sNodesNum)) / 1024
	freeSRUs := uint64(k8sNode.DiskSize * int(k8sNodesNum))
	freeIPs := uint64(0)
	if k8sNode.PublicIP {
		freeIPs = uint64(k8sNodesNum)
	}

	return buildGenericFilter(&freeMRUs, &freeSRUs, &freeIPs, []uint64{farmID}, nil)
}

func BuildVMFilter(vm workloads.VM, disk workloads.Disk, farmID uint64) types.NodeFilter {
	freeMRUs := uint64(vm.Memory) / 1024
	freeSRUs := uint64(vm.RootfsSize) / 1024
	freeIPs := uint64(0)
	if vm.PublicIP {
		freeIPs = 1
	}
	freeSRUs += uint64(disk.SizeGB)
	return buildGenericFilter(&freeMRUs, &freeSRUs, &freeIPs, []uint64{farmID}, nil)
}

func BuildGatewayFilter(farmID uint64) types.NodeFilter {
	domain := true
	return buildGenericFilter(nil, nil, nil, []uint64{farmID}, &domain)
}

func buildGenericFilter(mrus, srus, ips *uint64, farmIDs []uint64, domain *bool) types.NodeFilter {
	status := "up"
	return types.NodeFilter{
		Status:  &status,
		FreeMRU: mrus,
		FreeSRU: srus,
		FreeIPs: ips,
		FarmIDs: farmIDs,
		Domain:  domain,
	}
}
