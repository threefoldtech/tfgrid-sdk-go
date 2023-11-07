package mock

import (
	"context"
	"sort"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/nodestatus"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"golang.org/x/exp/slices"
)

func isDedicatedNode(db DBData, node Node) bool {
	return db.Farms[node.FarmID].DedicatedFarm ||
		len(db.NonDeletedContracts[node.NodeID]) == 0 ||
		db.NodeRentedBy[node.NodeID] != 0
}

// Nodes returns nodes with the given filters and pagination parameters
func (g *GridProxyMockClient) Nodes(ctx context.Context, filter types.NodeFilter, limit types.Limit) (res []types.Node, totalCount int, err error) {
	res = []types.Node{}
	if limit.Page == 0 {
		limit.Page = 1
	}
	if limit.Size == 0 {
		limit.Size = 50
	}
	for _, node := range g.data.Nodes {
		if node.satisfies(filter, &g.data) {
			numGPU := len(g.data.GPUs[node.TwinID])

			nodePower := types.NodePower{
				State:  node.Power.State,
				Target: node.Power.Target,
			}
			status := nodestatus.DecideNodeStatus(nodePower, int64(node.UpdatedAt))
			res = append(res, types.Node{
				ID:              node.ID,
				NodeID:          int(node.NodeID),
				FarmID:          int(node.FarmID),
				TwinID:          int(node.TwinID),
				Country:         node.Country,
				City:            node.City,
				GridVersion:     int(node.GridVersion),
				Uptime:          int64(node.Uptime),
				Created:         int64(node.Created),
				FarmingPolicyID: int(node.FarmingPolicyID),
				TotalResources: types.Capacity{
					CRU: g.data.NodeTotalResources[node.NodeID].CRU,
					HRU: gridtypes.Unit(g.data.NodeTotalResources[node.NodeID].HRU),
					MRU: gridtypes.Unit(g.data.NodeTotalResources[node.NodeID].MRU),
					SRU: gridtypes.Unit(g.data.NodeTotalResources[node.NodeID].SRU),
				},
				UsedResources: types.Capacity{
					CRU: g.data.NodeUsedResources[node.NodeID].CRU,
					HRU: gridtypes.Unit(g.data.NodeUsedResources[node.NodeID].HRU),
					MRU: gridtypes.Unit(g.data.NodeUsedResources[node.NodeID].MRU),
					SRU: gridtypes.Unit(g.data.NodeUsedResources[node.NodeID].SRU),
				},
				Location: types.Location{
					Country: node.Country,
					City:    node.City,
				},
				PublicConfig: types.PublicConfig{
					Domain: g.data.PublicConfigs[node.NodeID].Domain,
					Ipv4:   g.data.PublicConfigs[node.NodeID].IPv4,
					Ipv6:   g.data.PublicConfigs[node.NodeID].IPv6,
					Gw4:    g.data.PublicConfigs[node.NodeID].GW4,
					Gw6:    g.data.PublicConfigs[node.NodeID].GW6,
				},
				Status:            status,
				CertificationType: node.Certification,
				UpdatedAt:         int64(node.UpdatedAt),
				Dedicated:         isDedicatedNode(g.data, node),
				RentedByTwinID:    uint(g.data.NodeRentedBy[node.NodeID]),
				RentContractID:    uint(g.data.NodeRentContractID[node.NodeID]),
				SerialNumber:      node.SerialNumber,
				Power: types.NodePower{
					State:  node.Power.State,
					Target: node.Power.Target,
				},
				NumGPU: numGPU,
			})
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].NodeID < res[j].NodeID
	})

	if filter.AvailableFor != nil {
		sort.Slice(res, func(i, j int) bool {

			return g.data.NodeRentContractID[uint64(res[i].NodeID)] != 0
		})
	}

	res, totalCount = getPage(res, limit)

	return
}

func (g *GridProxyMockClient) Node(ctx context.Context, nodeID uint32) (res types.NodeWithNestedCapacity, err error) {
	node := g.data.Nodes[uint64(nodeID)]
	numGPU := len(g.data.GPUs[node.TwinID])

	nodePower := types.NodePower{
		State:  node.Power.State,
		Target: node.Power.Target,
	}
	status := nodestatus.DecideNodeStatus(nodePower, int64(node.UpdatedAt))
	res = types.NodeWithNestedCapacity{
		ID:              node.ID,
		NodeID:          int(node.NodeID),
		FarmID:          int(node.FarmID),
		TwinID:          int(node.TwinID),
		Country:         node.Country,
		City:            node.City,
		GridVersion:     int(node.GridVersion),
		Uptime:          int64(node.Uptime),
		Created:         int64(node.Created),
		FarmingPolicyID: int(node.FarmingPolicyID),
		Capacity: types.CapacityResult{
			Total: types.Capacity{
				CRU: g.data.NodeTotalResources[node.NodeID].CRU,
				HRU: gridtypes.Unit(g.data.NodeTotalResources[node.NodeID].HRU),
				MRU: gridtypes.Unit(g.data.NodeTotalResources[node.NodeID].MRU),
				SRU: gridtypes.Unit(g.data.NodeTotalResources[node.NodeID].SRU),
			},
			Used: types.Capacity{
				CRU: g.data.NodeUsedResources[node.NodeID].CRU,
				HRU: gridtypes.Unit(g.data.NodeUsedResources[node.NodeID].HRU),
				MRU: gridtypes.Unit(g.data.NodeUsedResources[node.NodeID].MRU),
				SRU: gridtypes.Unit(g.data.NodeUsedResources[node.NodeID].SRU),
			},
		},
		Location: types.Location{
			Country: node.Country,
			City:    node.City,
		},
		PublicConfig: types.PublicConfig{
			Domain: g.data.PublicConfigs[node.NodeID].Domain,
			Ipv4:   g.data.PublicConfigs[node.NodeID].IPv4,
			Ipv6:   g.data.PublicConfigs[node.NodeID].IPv6,
			Gw4:    g.data.PublicConfigs[node.NodeID].GW4,
			Gw6:    g.data.PublicConfigs[node.NodeID].GW6,
		},
		Status:            status,
		CertificationType: node.Certification,
		UpdatedAt:         int64(node.UpdatedAt),
		Dedicated:         isDedicatedNode(g.data, node),
		RentedByTwinID:    uint(g.data.NodeRentedBy[node.NodeID]),
		RentContractID:    uint(g.data.NodeRentContractID[node.NodeID]),
		SerialNumber:      node.SerialNumber,
		Power: types.NodePower{
			State:  node.Power.State,
			Target: node.Power.Target,
		},
		NumGPU:   numGPU,
		ExtraFee: node.ExtraFee,
	}
	return
}

func (g *GridProxyMockClient) NodeStatus(ctx context.Context, nodeID uint32) (res types.NodeStatus, err error) {
	node := g.data.Nodes[uint64(nodeID)]
	nodePower := types.NodePower{
		State:  node.Power.State,
		Target: node.Power.Target,
	}
	res.Status = nodestatus.DecideNodeStatus(nodePower, int64(node.UpdatedAt))
	return
}

func (n *Node) satisfies(f types.NodeFilter, data *DBData) bool {
	nodePower := types.NodePower{
		State:  n.Power.State,
		Target: n.Power.Target,
	}

	total := data.NodeTotalResources[n.NodeID]
	used := data.NodeUsedResources[n.NodeID]
	free := calcFreeResources(total, used)

	if f.Status != nil && *f.Status != nodestatus.DecideNodeStatus(nodePower, int64(n.UpdatedAt)) {
		return false
	}

	if f.FreeMRU != nil && *f.FreeMRU > free.MRU {
		return false
	}

	if f.FreeHRU != nil && *f.FreeHRU > free.HRU {
		return false
	}

	if f.FreeSRU != nil && *f.FreeSRU > free.SRU {
		return false
	}

	if f.TotalCRU != nil && *f.TotalCRU > total.CRU {
		return false
	}

	if f.TotalHRU != nil && *f.TotalHRU > total.HRU {
		return false
	}

	if f.TotalMRU != nil && *f.TotalMRU > total.MRU {
		return false
	}

	if f.TotalSRU != nil && *f.TotalSRU > total.SRU {
		return false
	}

	if f.Country != nil && !strings.EqualFold(*f.Country, n.Country) {
		return false
	}

	if f.CountryContains != nil && !stringMatch(n.Country, *f.CountryContains) {
		return false
	}

	if f.City != nil && !strings.EqualFold(*f.City, n.City) {
		return false
	}

	if f.CityContains != nil && !stringMatch(n.City, *f.CityContains) {
		return false
	}

	if f.FarmName != nil && !strings.EqualFold(*f.FarmName, data.Farms[n.FarmID].Name) {
		return false
	}

	if f.FarmNameContains != nil && !stringMatch(data.Farms[n.FarmID].Name, *f.FarmNameContains) {
		return false
	}

	if f.FarmIDs != nil && len(f.FarmIDs) != 0 && !slices.Contains(f.FarmIDs, n.FarmID) {
		return false
	}

	if f.FreeIPs != nil && *f.FreeIPs > data.FreeIPs[n.FarmID] {
		return false
	}

	if f.IPv4 != nil && *f.IPv4 == (data.PublicConfigs[n.NodeID].IPv4 == "") {
		return false
	}

	if f.IPv6 != nil && *f.IPv6 == (data.PublicConfigs[n.NodeID].IPv6 == "") {
		return false
	}

	if f.Domain != nil && *f.Domain == (data.PublicConfigs[n.NodeID].Domain == "") {
		return false
	}

	if f.Dedicated != nil && *f.Dedicated != isDedicatedNode(*data, *n) {
		return false
	}

	rentable := data.NodeRentedBy[n.NodeID] == 0 &&
		(data.Farms[n.FarmID].DedicatedFarm || len(data.NonDeletedContracts[n.NodeID]) == 0)
	if f.Rentable != nil && *f.Rentable != rentable {
		return false
	}

	_, ok := data.NodeRentedBy[n.NodeID]
	if f.Rented != nil && *f.Rented != ok {
		return false
	}

	if f.RentedBy != nil && *f.RentedBy != data.NodeRentedBy[n.NodeID] {
		return false
	}

	renter, ok := data.NodeRentedBy[n.NodeID]
	if f.AvailableFor != nil &&
		((ok && renter != *f.AvailableFor) ||
			(!ok && data.Farms[n.FarmID].DedicatedFarm)) {
		return false
	}

	if f.NodeID != nil && *f.NodeID != n.NodeID {
		return false
	}

	if f.TwinID != nil && *f.TwinID != n.TwinID {
		return false
	}

	if f.CertificationType != nil && *f.CertificationType != n.Certification {
		return false
	}

	foundGpuFilter := f.HasGPU != nil || f.GpuDeviceName != nil || f.GpuVendorName != nil || f.GpuVendorID != nil || f.GpuDeviceID != nil || f.GpuAvailable != nil
	gpus, foundGpuCards := data.GPUs[n.TwinID]

	if !foundGpuCards && foundGpuFilter {
		return false
	}

	if f.HasGPU != nil && *f.HasGPU != foundGpuCards {
		return false
	}

	foundSuitableCard := false
	for _, gpu := range gpus {
		if gpuSatisfied(gpu, f) {
			foundSuitableCard = true
		}
	}

	if !foundSuitableCard && foundGpuFilter {
		return false
	}

	return true
}

func gpuSatisfied(gpu NodeGPU, f types.NodeFilter) bool {
	if f.GpuDeviceName != nil && !contains(gpu.Device, *f.GpuDeviceName) {
		return false
	}

	if f.GpuVendorName != nil && !contains(gpu.Vendor, *f.GpuVendorName) {
		return false
	}

	if f.GpuVendorID != nil && !contains(gpu.ID, *f.GpuVendorID) {
		return false
	}

	if f.GpuDeviceID != nil && !contains(gpu.ID, *f.GpuDeviceID) {
		return false
	}

	if f.GpuAvailable != nil && *f.GpuAvailable != (gpu.Contract == 0) {
		return false
	}

	return true
}

func contains(s string, sub string) bool {
	return strings.Contains(strings.ToLower(s), sub)
}
