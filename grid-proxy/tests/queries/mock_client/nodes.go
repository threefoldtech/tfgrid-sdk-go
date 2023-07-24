package mock

import (
	"sort"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/nodestatus"
	proxytypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// Nodes returns nodes with the given filters and pagination parameters
func (g *GridProxyMockClient) Nodes(filter proxytypes.NodeFilter, limit proxytypes.Limit) (res []proxytypes.Node, totalCount int, err error) {
	res = []proxytypes.Node{}
	if limit.Page == 0 {
		limit.Page = 1
	}
	if limit.Size == 0 {
		limit.Size = 50
	}
	for _, node := range g.data.Nodes {
		if nodeSatisfies(&g.data, node, filter) {
			nodePower := proxytypes.NodePower{
				State:  node.Power.State,
				Target: node.Power.Target,
			}
			status := nodestatus.DecideNodeStatus(nodePower, int64(node.UpdatedAt))
			res = append(res, proxytypes.Node{
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
				TotalResources: proxytypes.Capacity{
					CRU: g.data.NodeTotalResources[node.NodeID].CRU,
					HRU: gridtypes.Unit(g.data.NodeTotalResources[node.NodeID].HRU),
					MRU: gridtypes.Unit(g.data.NodeTotalResources[node.NodeID].MRU),
					SRU: gridtypes.Unit(g.data.NodeTotalResources[node.NodeID].SRU),
				},
				UsedResources: proxytypes.Capacity{
					CRU: g.data.NodeUsedResources[node.NodeID].CRU,
					HRU: gridtypes.Unit(g.data.NodeUsedResources[node.NodeID].HRU),
					MRU: gridtypes.Unit(g.data.NodeUsedResources[node.NodeID].MRU),
					SRU: gridtypes.Unit(g.data.NodeUsedResources[node.NodeID].SRU),
				},
				Location: proxytypes.Location{
					Country: node.Country,
					City:    node.City,
				},
				PublicConfig: proxytypes.PublicConfig{
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
				Power: proxytypes.NodePower{
					State:  node.Power.State,
					Target: node.Power.Target,
				},
				NumGPU: getNumGPUs(node.HasGPU),
			})
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].NodeID < res[j].NodeID
	})
	if filter.AvailableFor != nil {
		sort.Slice(res, func(i, j int) bool {

			return g.data.NodeRentContractID[uint64(res[i].NodeID)] != 0

			// return res[i].NodeID < res[j].NodeID
		})
	}

	res, totalCount = getPage(res, limit)

	return
}

func (g *GridProxyMockClient) Node(nodeID uint32) (res proxytypes.NodeWithNestedCapacity, err error) {
	node := g.data.Nodes[uint64(nodeID)]
	numGPU := 0
	if _, ok := g.data.GPUs[node.TwinID]; ok {
		numGPU = 1
	}
	nodePower := proxytypes.NodePower{
		State:  node.Power.State,
		Target: node.Power.Target,
	}
	status := nodestatus.DecideNodeStatus(nodePower, int64(node.UpdatedAt))
	res = proxytypes.NodeWithNestedCapacity{
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
		Capacity: proxytypes.CapacityResult{
			Total: proxytypes.Capacity{
				CRU: g.data.NodeTotalResources[node.NodeID].CRU,
				HRU: gridtypes.Unit(g.data.NodeTotalResources[node.NodeID].HRU),
				MRU: gridtypes.Unit(g.data.NodeTotalResources[node.NodeID].MRU),
				SRU: gridtypes.Unit(g.data.NodeTotalResources[node.NodeID].SRU),
			},
			Used: proxytypes.Capacity{
				CRU: g.data.NodeUsedResources[node.NodeID].CRU,
				HRU: gridtypes.Unit(g.data.NodeUsedResources[node.NodeID].HRU),
				MRU: gridtypes.Unit(g.data.NodeUsedResources[node.NodeID].MRU),
				SRU: gridtypes.Unit(g.data.NodeUsedResources[node.NodeID].SRU),
			},
		},
		Location: proxytypes.Location{
			Country: node.Country,
			City:    node.City,
		},
		PublicConfig: proxytypes.PublicConfig{
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
		Power: proxytypes.NodePower{
			State:  node.Power.State,
			Target: node.Power.Target,
		},
		NumGPU:   numGPU,
		ExtraFee: node.ExtraFee,
	}
	return
}

func (g *GridProxyMockClient) NodeStatus(nodeID uint32) (res proxytypes.NodeStatus, err error) {
	node := g.data.Nodes[uint64(nodeID)]
	nodePower := proxytypes.NodePower{
		State:  node.Power.State,
		Target: node.Power.Target,
	}
	res.Status = nodestatus.DecideNodeStatus(nodePower, int64(node.UpdatedAt))
	return
}

func nodeSatisfies(data *DBData, node Node, f proxytypes.NodeFilter) bool {
	nodePower := proxytypes.NodePower{
		State:  node.Power.State,
		Target: node.Power.Target,
	}
	if f.Status != nil && *f.Status != nodestatus.DecideNodeStatus(nodePower, int64(node.UpdatedAt)) {
		return false
	}
	total := data.NodeTotalResources[node.NodeID]
	used := data.NodeUsedResources[node.NodeID]
	free := calcFreeResources(total, used)
	if f.FreeMRU != nil && *f.FreeMRU > free.MRU {
		return false
	}
	if f.FreeHRU != nil && *f.FreeHRU > free.HRU {
		return false
	}
	if f.FreeSRU != nil && *f.FreeSRU > free.SRU {
		return false
	}
	if f.Country != nil && !strings.EqualFold(*f.Country, node.Country) {
		return false
	}
	if f.CountryContains != nil && !stringMatch(node.Country, *f.CountryContains) {
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
	if f.NodeID != nil && *f.NodeID != node.NodeID {
		return false
	}
	if f.TwinID != nil && *f.TwinID != node.TwinID {
		return false
	}
	if f.CityContains != nil && !stringMatch(node.City, *f.CityContains) {
		return false
	}
	if f.City != nil && !strings.EqualFold(*f.City, node.City) {
		return false
	}
	if f.FarmNameContains != nil && !stringMatch(data.Farms[node.FarmID].Name, *f.FarmNameContains) {
		return false
	}
	if f.FarmName != nil && !strings.EqualFold(*f.FarmName, data.Farms[node.FarmID].Name) {
		return false
	}
	if f.FarmIDs != nil && !isIn(f.FarmIDs, node.FarmID) {
		return false
	}
	if f.FreeIPs != nil && *f.FreeIPs > data.FreeIPs[node.FarmID] {
		return false
	}
	if f.IPv4 != nil && *f.IPv4 && data.PublicConfigs[node.NodeID].IPv4 == "" {
		return false
	}
	if f.IPv6 != nil && *f.IPv6 && data.PublicConfigs[node.NodeID].IPv6 == "" {
		return false
	}
	if f.Domain != nil && *f.Domain && data.PublicConfigs[node.NodeID].Domain == "" {
		return false
	}
	rentable := data.NodeRentedBy[node.NodeID] == 0 &&
		(data.Farms[node.FarmID].DedicatedFarm || len(data.NonDeletedContracts[node.NodeID]) == 0)
	if f.Rentable != nil && *f.Rentable != rentable {
		return false
	}
	if f.RentedBy != nil && *f.RentedBy != data.NodeRentedBy[node.NodeID] {
		return false
	}
	if f.AvailableFor != nil &&
		((data.NodeRentedBy[node.NodeID] != 0 && data.NodeRentedBy[node.NodeID] != *f.AvailableFor) ||
			(data.NodeRentedBy[node.NodeID] != *f.AvailableFor && data.Farms[node.FarmID].DedicatedFarm)) {
		return false
	}
	if f.Rented != nil {
		_, ok := data.NodeRentedBy[node.NodeID]
		return ok == *f.Rented
	}
	if f.HasGPU != nil && *f.HasGPU != node.HasGPU {
		return false
	}

	if !gpuSatisfies(data, node, f) {
		return false
	}

	return true
}

// getNumGPUs should be deleted after removing hasGPU
func getNumGPUs(hasGPU bool) int {

	if hasGPU {
		return 1
	}
	return 0
}

func gpuSatisfies(data *DBData, node Node, f proxytypes.NodeFilter) bool {
	gpu := data.GPUs[node.TwinID]

	if f.GpuDeviceName != nil {
		if !strings.Contains(strings.ToLower(gpu.Device), *f.GpuDeviceName) {
			return false
		}
	}

	if f.GpuVendorName != nil {
		if !strings.Contains(strings.ToLower(gpu.Vendor), *f.GpuVendorName) {
			return false
		}
	}

	if f.GpuVendorID != nil {
		if !strings.Contains(gpu.ID, *f.GpuVendorID) {
			return false
		}
	}

	if f.GpuDeviceID != nil {
		if !strings.Contains(gpu.ID, *f.GpuDeviceID) {
			return false
		}
	}

	if f.GpuAvailable != nil {
		if gpu.Contract == 0 != *f.GpuAvailable {
			return false
		}
	}
	return true
}
