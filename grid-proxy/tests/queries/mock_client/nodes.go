package mock

import (
	"sort"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/nodestatus"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"golang.org/x/exp/slices"
)

// Nodes returns nodes with the given filters and pagination parameters
func (g *GridProxyMockClient) Nodes(filter types.NodeFilter, limit types.Limit) (res []types.Node, totalCount int, err error) {
	res = []types.Node{}
	if limit.Page == 0 {
		limit.Page = 1
	}
	if limit.Size == 0 {
		limit.Size = 50
	}
	for _, node := range g.data.Nodes {
		if node.satisfies(filter, &g.data) {
			numGPU := 0
			if _, ok := g.data.GPUs[node.TwinID]; ok {
				numGPU = 1
			}

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
					CRU: node.TotalCRU,
					MRU: gridtypes.Unit(node.TotalMRU),
					SRU: gridtypes.Unit(node.TotalSRU),
					HRU: gridtypes.Unit(node.TotalHRU),
				},
				UsedResources: types.Capacity{
					CRU: 0,
					HRU: gridtypes.Unit(node.TotalHRU - g.data.NodesCacheMap[node.NodeID].FreeHRU),
					MRU: gridtypes.Unit(node.TotalMRU - g.data.NodesCacheMap[node.NodeID].FreeMRU),
					SRU: gridtypes.Unit(node.TotalSRU - g.data.NodesCacheMap[node.NodeID].FreeSRU),
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
				UpdatedAt:         int64(int64(node.UpdatedAt)),
				Dedicated:         g.data.Farms[node.FarmID].DedicatedFarm,
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

func (g *GridProxyMockClient) Node(nodeID uint32) (res types.NodeWithNestedCapacity, err error) {
	node := g.data.Nodes[uint64(nodeID)]
	numGPU := 0
	if _, ok := g.data.GPUs[node.TwinID]; ok {
		numGPU = 1
	}
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
				CRU: node.TotalCRU,
				HRU: gridtypes.Unit(node.TotalHRU),
				MRU: gridtypes.Unit(node.TotalMRU),
				SRU: gridtypes.Unit(node.TotalSRU),
			},
			Used: types.Capacity{
				CRU: 0,
				HRU: gridtypes.Unit(node.TotalHRU - g.data.NodesCacheMap[node.NodeID].FreeHRU),
				MRU: gridtypes.Unit(node.TotalMRU - g.data.NodesCacheMap[node.NodeID].FreeMRU),
				SRU: gridtypes.Unit(node.TotalSRU - g.data.NodesCacheMap[node.NodeID].FreeSRU),
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
		UpdatedAt:         int64(int64(node.UpdatedAt)),
		Dedicated:         g.data.Farms[node.FarmID].DedicatedFarm,
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

func (g *GridProxyMockClient) NodeStatus(nodeID uint32) (res types.NodeStatus, err error) {
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
	free := NodeResourcesTotal{
		HRU: data.NodesCacheMap[n.NodeID].FreeHRU,
		SRU: data.NodesCacheMap[n.NodeID].FreeSRU,
		MRU: data.NodesCacheMap[n.NodeID].FreeMRU,
	}

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

	if f.Dedicated != nil && *f.Dedicated != data.Farms[n.FarmID].DedicatedFarm {
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

	gpu, ok := data.GPUs[n.TwinID]
	if f.HasGPU != nil && *f.HasGPU != ok {
		return false
	}

	if f.GpuDeviceName != nil && !strings.Contains(strings.ToLower(gpu.Device), *f.GpuDeviceName) {
		return false
	}

	if f.GpuVendorName != nil && !strings.Contains(strings.ToLower(gpu.Vendor), *f.GpuVendorName) {
		return false
	}

	if f.GpuVendorID != nil && !strings.Contains(strings.ToLower(gpu.ID), *f.GpuVendorID) {
		return false
	}

	if f.GpuDeviceID != nil && !strings.Contains(strings.ToLower(gpu.ID), *f.GpuDeviceID) {
		return false
	}

	if f.GpuAvailable != nil && *f.GpuAvailable != (gpu.Contract == 0) {
		return false
	}

	if !ok && (f.HasGPU != nil || f.GpuDeviceName != nil || f.GpuVendorName != nil || f.GpuVendorID != nil || f.GpuDeviceID != nil || f.GpuAvailable != nil) {
		return false
	}
	return true
}
