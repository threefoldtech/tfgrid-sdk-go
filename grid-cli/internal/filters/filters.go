// Package filters for filtering nodes for needed resources
package filters

import (
	"context"
	"fmt"
	"math/rand"
	"strings"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// GetAvailableNode returns a node with available resources based on provided filter
func GetAvailableNode(t *deployer.TFPluginClient, filter types.NodeFilter) (uint32, error) {
	nodes, _, err := t.GridProxyClient.Nodes(filter, types.Limit{})
	if err != nil {
		return 0, err
	}
	if len(nodes) == 0 {
		return 0, fmt.Errorf("no node with free resources available using node filter: %s", serializeFilter(filter))
	}

	// shuffle nodes
	for i := range nodes {
		j := rand.Intn(i + 1)
		nodes[i], nodes[j] = nodes[j], nodes[i]
	}
	if filter.FreeSRU == nil {
		return uint32(nodes[0].NodeID), nil
	}

	for _, node := range nodes {
		client, err := t.NcPool.GetNodeClient(t.SubstrateConn, uint32(node.NodeID))
		if err != nil {
			return 0, errors.Wrapf(err, "failed to get node %d client", node.NodeID)
		}
		pools, err := client.Pools(context.Background())
		if err != nil {
			return 0, errors.Wrapf(err, "failed to get node %d pools", node.NodeID)
		}
		if hasEnoughStorage(pools, *filter.FreeSRU) {
			return uint32(node.NodeID), nil
		}

	}

	return 0, fmt.Errorf("no node with free resources available using node filter: %s", serializeFilter(filter))
}

// BuildK8sFilter build a filter for a k8s node
func BuildK8sFilter(k8sNode workloads.K8sNode, farmID uint64, k8sNodesNum uint) types.NodeFilter {
	freeMRUs := uint64(k8sNode.Memory*int(k8sNodesNum)) / 1024
	freeSRUs := uint64(k8sNode.DiskSize * int(k8sNodesNum))
	freeIPs := uint64(0)
	if k8sNode.PublicIP {
		freeIPs = uint64(k8sNodesNum)
	}

	return buildGenericFilter(&freeMRUs, &freeSRUs, &freeIPs, []uint64{farmID}, nil)
}

// BuildVMFilter build a filter for a vm
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

// BuildGatewayFilter build a filter for a gateway
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

func hasEnoughStorage(pools []client.PoolMetrics, storage uint64) bool {
	for _, pool := range pools {
		if pool.Size-pool.Used > gridtypes.Unit(storage*1024*1024*1024) {
			return true
		}
	}
	return false
}

func serializeFilter(filter types.NodeFilter) string {
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
	return filterString[:len(filterString)-2]
}
