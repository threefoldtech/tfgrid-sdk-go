package mock

import (
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"golang.org/x/exp/slices"
)

// Nodes returns nodes with the given filters and pagination parameters
func (g *GridProxyMockClient) Nodes(filter types.NodeFilter, limit types.Limit) ([]types.Node, int, error) {
	nodes, err := g.filterNodes(filter)
	if err != nil {
		return nil, 0, err
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].NodeID < nodes[j].NodeID
	})

	res, count := getPage(nodes, limit)

	return res, count, nil
}

func (g *GridProxyMockClient) filterNodes(filter types.NodeFilter) ([]types.Node, error) {
	res := []types.Node{}

	for _, node := range g.data.Nodes {
		satisfies, err := node.nodeSatisfies(&g.data, filter)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to check node %d eligibility", node.NodeID)
		}

		if satisfies {
			status := decideNodeStatus(node.Power, node.UpdatedAt)
			res = append(res, types.Node{
				ID:              node.ID,
				NodeID:          node.NodeID,
				FarmID:          node.FarmID,
				TwinID:          node.TwinID,
				Country:         node.Country,
				City:            node.City,
				GridVersion:     node.GridVersion,
				Uptime:          node.Uptime,
				Created:         node.Created,
				FarmingPolicyID: node.FarmingPolicyID,
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
				UpdatedAt:         node.UpdatedAt,
				Dedicated:         g.data.Farms[node.FarmID].DedicatedFarm,
				RentedByTwinID:    g.data.NodeRentedBy[node.NodeID],
				RentContractID:    g.data.NodeRentContractID[node.NodeID],
				SerialNumber:      node.SerialNumber,
				Power: types.NodePower{
					State:  node.Power.State,
					Target: node.Power.Target,
				},
				NumGPU: getNumGPUs(node.HasGPU),
			})
		}
	}

	return res, nil
}

func (g *GridProxyMockClient) Node(nodeID uint32) (res types.NodeWithNestedCapacity, err error) {
	node := g.data.Nodes[nodeID]
	status := decideNodeStatus(node.Power, node.UpdatedAt)
	res = types.NodeWithNestedCapacity{
		ID:              node.ID,
		NodeID:          node.NodeID,
		FarmID:          node.FarmID,
		TwinID:          node.TwinID,
		Country:         node.Country,
		City:            node.City,
		GridVersion:     node.GridVersion,
		Uptime:          node.Uptime,
		Created:         node.Created,
		FarmingPolicyID: node.FarmingPolicyID,
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
		UpdatedAt:         node.UpdatedAt,
		Dedicated:         g.data.Farms[node.FarmID].DedicatedFarm,
		RentedByTwinID:    g.data.NodeRentedBy[node.NodeID],
		RentContractID:    g.data.NodeRentContractID[node.NodeID],
		SerialNumber:      node.SerialNumber,
		Power: types.NodePower{
			State:  node.Power.State,
			Target: node.Power.Target,
		},
		NumGPU:   getNumGPUs(node.HasGPU),
		ExtraFee: node.ExtraFee,
	}

	return
}

func (g *GridProxyMockClient) NodeStatus(nodeID uint32) (res types.NodeStatus, err error) {
	node := g.data.Nodes[nodeID]
	res.Status = decideNodeStatus(node.Power, node.UpdatedAt)
	return
}

func (n *DBNode) nodeSatisfies(data *DBData, f types.NodeFilter) (bool, error) {
	if f.Status != nil && (*f.Status == STATUS_UP) != isUp(n.UpdatedAt) {
		return false, nil
	}

	total := data.NodeTotalResources[data.NodeIDMap[n.ID]]
	used := data.NodeUsedResources[data.NodeIDMap[n.ID]]
	free, err := CalculateFreeResources(total, used)
	if err != nil {
		return false, err
	}
	if f.FreeMRU != nil && *f.FreeMRU > free.MRU {
		return false, nil
	}
	if f.FreeHRU != nil && *f.FreeHRU > free.HRU {
		return false, nil
	}
	if f.FreeSRU != nil && *f.FreeSRU > free.SRU {
		return false, nil
	}
	if f.Country != nil && !strings.EqualFold(*f.Country, n.Country) {
		return false, nil
	}
	if f.CountryContains != nil && !stringMatch(n.Country, *f.CountryContains) {
		return false, nil
	}
	if f.TotalCRU != nil && *f.TotalCRU > total.CRU {
		return false, nil
	}
	if f.TotalHRU != nil && *f.TotalHRU > total.HRU {
		return false, nil
	}
	if f.TotalMRU != nil && *f.TotalMRU > total.MRU {
		return false, nil
	}
	if f.TotalSRU != nil && *f.TotalSRU > total.SRU {
		return false, nil
	}
	if f.NodeID != nil && *f.NodeID != n.NodeID {
		return false, nil
	}
	if f.TwinID != nil && *f.TwinID != n.TwinID {
		return false, nil
	}
	if f.CityContains != nil && !stringMatch(n.City, *f.CityContains) {
		return false, nil
	}
	if f.City != nil && !strings.EqualFold(*f.City, n.City) {
		return false, nil
	}
	if f.FarmNameContains != nil && !stringMatch(data.Farms[n.FarmID].Name, *f.FarmNameContains) {
		return false, nil
	}
	if f.FarmName != nil && !strings.EqualFold(*f.FarmName, data.Farms[n.FarmID].Name) {
		return false, nil
	}
	if f.FarmIDs != nil && !slices.Contains(f.FarmIDs, n.FarmID) {
		return false, nil
	}
	if f.FreeIPs != nil && *f.FreeIPs > data.FreeIPs[n.FarmID] {
		return false, nil
	}
	if f.IPv4 != nil && *f.IPv4 && data.PublicConfigs[n.NodeID].IPv4 == "" {
		return false, nil
	}
	if f.IPv6 != nil && *f.IPv6 && data.PublicConfigs[n.NodeID].IPv6 == "" {
		return false, nil
	}
	if f.Domain != nil && *f.Domain && data.PublicConfigs[n.NodeID].Domain == "" {
		return false, nil
	}
	rentable := data.NodeRentedBy[n.NodeID] == 0 &&
		(data.Farms[n.FarmID].DedicatedFarm || len(data.NonDeletedContracts[n.NodeID]) == 0)
	if f.Rentable != nil && *f.Rentable != rentable {
		return false, nil
	}
	if f.RentedBy != nil && *f.RentedBy != data.NodeRentedBy[n.NodeID] {
		return false, nil
	}
	if f.AvailableFor != nil &&
		((data.NodeRentedBy[n.NodeID] != 0 && data.NodeRentedBy[n.NodeID] != *f.AvailableFor) ||
			(data.NodeRentedBy[n.NodeID] != *f.AvailableFor && data.Farms[n.FarmID].DedicatedFarm)) {
		return false, nil
	}
	if f.Rented != nil {
		_, ok := data.NodeRentedBy[n.NodeID]

		return ok == *f.Rented, nil
	}

	if f.CertificationType != nil && *f.CertificationType != n.Certification {
		return false, nil
	}

	if f.HasGPU != nil && !n.HasGPU {
		return false, nil
	}

	return true, nil
}

func decideNodeStatus(power db.NodePower, updatedAt int64) string {
	if power.Target == "Down" { // off or powering off
		return "standby"
	} else if power.Target == "Up" && power.State == "Down" { // powering on
		return "down"
	} else if updatedAt >= time.Now().Add(nodeUpInterval).Unix() {
		return "up"
	} else {
		return "down"
	}
}

// getNumGPUs should be deleted after removing hasGPU
func getNumGPUs(hasGPU bool) uint8 {
	if hasGPU {
		return 1
	}
	return 0
}
