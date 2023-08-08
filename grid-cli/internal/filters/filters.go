// Package filters for filtering nodes for needed resources
package filters

import (
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

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

// BuildZDBFilter build a filter for a zdbs
func BuildZDBFilter(zdb workloads.ZDB, n int, farmID uint64) types.NodeFilter {
	freeHRUs := uint64(zdb.Size*n) / 1024
	status := "up"
	return types.NodeFilter{
		Status:  &status,
		FreeHRU: convertGBToBytes(&freeHRUs),
		FarmIDs: []uint64{farmID},
	}
}

func buildGenericFilter(mrus, srus, ips *uint64, farmIDs []uint64, domain *bool) types.NodeFilter {
	status := "up"
	return types.NodeFilter{
		Status:  &status,
		FreeMRU: convertGBToBytes(mrus),
		FreeSRU: convertGBToBytes(srus),
		FreeIPs: ips,
		FarmIDs: farmIDs,
		Domain:  domain,
	}
}

func convertGBToBytes(gb *uint64) *uint64 {
	bytes := (*gb) * 1024 * 1024 * 1024
	return &bytes
}
