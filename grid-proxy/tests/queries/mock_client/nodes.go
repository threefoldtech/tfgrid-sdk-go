package mock

import (
	"context"
	"fmt"
	"math"
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

func calculateCU(cru, mru float64) float64 {
	MruUsed1 := float64(mru / 4)
	CruUsed1 := float64(cru / 2)
	cu1 := math.Max(MruUsed1, CruUsed1)

	MruUsed2 := float64(mru / 8)
	CruUsed2 := float64(cru)
	cu2 := math.Max(MruUsed2, CruUsed2)

	MruUsed3 := float64(mru / 2)
	CruUsed3 := float64(cru / 4)
	cu3 := math.Max(MruUsed3, CruUsed3)

	cu := math.Min(cu1, cu2)
	cu = math.Min(cu, cu3)

	return cu
}

func calculateSU(hru, sru float64) float64 {
	return float64(hru/1200 + sru/200)
}

func calcNodePrice(db DBData, node Node) float64 {
	cu := calculateCU(float64(db.NodeTotalResources[node.NodeID].CRU),
		float64(db.NodeTotalResources[node.NodeID].MRU)/(1024*1024*1024))
	su := calculateSU(float64(db.NodeTotalResources[node.NodeID].HRU)/(1024*1024*1024),
		float64(db.NodeTotalResources[node.NodeID].SRU)/(1024*1024*1024))

	pricingPolicy := db.PricingPolicies[uint(db.Farms[node.FarmID].PricingPolicyID)]
	certifiedFactor := float64(1)
	if node.Certification == "Certified" {
		certifiedFactor = 1.25
	}

	costPerMonth := (cu*float64(pricingPolicy.CU.Value) +
		su*float64(pricingPolicy.SU.Value) +
		float64(node.ExtraFee)) *
		certifiedFactor * 24 * 30

	costInUsd := costPerMonth / 1e7
	return math.Round(costInUsd*1000) / 1000
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
				FarmName:        g.data.Farms[node.FarmID].Name,
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
					Country:   node.Country,
					City:      node.City,
					Longitude: g.data.Locations[node.LocationID].Longitude,
					Latitude:  g.data.Locations[node.LocationID].Latitude,
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
				InDedicatedFarm:   g.data.Farms[node.FarmID].DedicatedFarm,
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
				Healthy:  g.data.HealthReports[node.TwinID],
				PriceUsd: calcNodePrice(g.data, node),
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
	node, ok := g.data.Nodes[uint64(nodeID)]
	if !ok {
		return res, fmt.Errorf("node not found")
	}

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
		FarmName:        g.data.Farms[node.FarmID].Name,
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
			Country:   node.Country,
			City:      node.City,
			Longitude: g.data.Locations[node.LocationID].Longitude,
			Latitude:  g.data.Locations[node.LocationID].Latitude,
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
		InDedicatedFarm:   g.data.Farms[node.FarmID].DedicatedFarm,
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
		Healthy:  g.data.HealthReports[node.TwinID],
		PriceUsd: calcNodePrice(g.data, node),
	}
	return
}

func (g *GridProxyMockClient) NodeStatus(ctx context.Context, nodeID uint32) (res types.NodeStatus, err error) {
	node, ok := g.data.Nodes[uint64(nodeID)]
	if !ok {
		return res, fmt.Errorf("node not found")
	}

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
	free := CalcFreeResources(total, used)

	if f.Status != nil && *f.Status != nodestatus.DecideNodeStatus(nodePower, int64(n.UpdatedAt)) {
		return false
	}

	if f.FreeMRU != nil && int64(*f.FreeMRU) > int64(free.MRU) {
		return false
	}

	if f.FreeHRU != nil && int64(*f.FreeHRU) > int64(free.HRU) {
		return false
	}

	if f.Healthy != nil && *f.Healthy != data.HealthReports[n.TwinID] {
		return false
	}

	if f.FreeSRU != nil && int64(*f.FreeSRU) > int64(free.SRU) {
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

	if f.Region != nil && !strings.EqualFold(*f.Region, data.Regions[strings.ToLower(n.Country)]) {
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

	if f.InDedicatedFarm != nil && *f.InDedicatedFarm != data.Farms[n.FarmID].DedicatedFarm {
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

	if f.OwnedBy != nil && *f.OwnedBy != data.Farms[n.FarmID].TwinID {
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
