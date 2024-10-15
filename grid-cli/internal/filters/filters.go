// Package filters for filtering nodes for needed resources
package filters

import (
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// BuildK8sNodeFilter build a filter for a k8s node
func BuildK8sNodeFilter(k8sNode workloads.K8sNode, farmID uint64) (types.NodeFilter, []uint64, []uint64) {
	freeMRUs := k8sNode.MemoryMB / 1024
	freeSRUs := k8sNode.DiskSizeGB
	freeIPs := uint64(0)

	if k8sNode.PublicIP {
		freeIPs = uint64(1)
	}
	disks := []uint64{*convertGBToBytes(k8sNode.DiskSizeGB)}
	// k8s rootfs is either 2 or 0.5
	rootfss := []uint64{*convertGBToBytes(uint64(2))}

	return buildGenericFilter(&freeMRUs, &freeSRUs, nil, &freeIPs, []uint64{farmID}, nil, false), disks, rootfss
}

// BuildVMFilter build a filter for a vm
func BuildVMFilter(disk workloads.Disk, volume workloads.Volume, farmID, memoryMB, rootfsMB uint64, ipv4, light bool) (types.NodeFilter, []uint64, []uint64) {
	freeMRUs := memoryMB / 1024
	freeSRUs := rootfsMB / 1024
	freeIPs := uint64(0)
	if ipv4 {
		freeIPs = 1
	}

	freeSRUs += disk.SizeGB + volume.SizeGB

	ssd := make([]uint64, 0)
	if disk.SizeGB > 0 {
		ssd = append(ssd, *convertGBToBytes(disk.SizeGB))
	}
	if volume.SizeGB > 0 {
		ssd = append(ssd, *convertGBToBytes(volume.SizeGB))
	}

	rootfss := []uint64{*convertGBToBytes(rootfsMB / 1024)}
	return buildGenericFilter(&freeMRUs, &freeSRUs, nil, &freeIPs, []uint64{farmID}, nil, light), ssd, rootfss
}

// BuildGatewayFilter build a filter for a gateway
func BuildGatewayFilter(farmID uint64) types.NodeFilter {
	domain := true
	return buildGenericFilter(nil, nil, nil, nil, []uint64{farmID}, &domain, false)
}

// BuildZDBFilter build a filter for a zdbs
func BuildZDBFilter(zdb workloads.ZDB, n int, farmID uint64) (types.NodeFilter, []uint64) {
	freeHRUs := zdb.SizeGB * uint64(n)
	return buildGenericFilter(nil, nil, &freeHRUs, nil, []uint64{farmID}, nil, false), []uint64{*convertGBToBytes(freeHRUs)}
}

func buildGenericFilter(mrus, srus, hrus, ips *uint64, farmIDs []uint64, domain *bool, light bool) types.NodeFilter {
	var freeMRUs *uint64
	if mrus != nil {
		freeMRUs = convertGBToBytes(*mrus)
	}
	var freeSRUs *uint64
	if srus != nil {
		freeSRUs = convertGBToBytes(*srus)
	}
	var freeHRUs *uint64
	if hrus != nil {
		freeHRUs = convertGBToBytes(*hrus)
	}
	rented := false

	var features []string
	if light {
		features = append(features, zos.NetworkLightType, zos.ZMachineLightType)
	}

	return types.NodeFilter{
		Status:   []string{"up"},
		FreeMRU:  freeMRUs,
		FreeSRU:  freeSRUs,
		FreeHRU:  freeHRUs,
		FreeIPs:  ips,
		FarmIDs:  farmIDs,
		Domain:   domain,
		Rented:   &rented,
		Features: features,
	}
}

func convertGBToBytes(gb uint64) *uint64 {
	bytes := gb * 1024 * 1024 * 1024
	return &bytes
}
