package mock

import (
	"sort"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/nodestatus"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// Farms returns farms with the given filters and pagination parameters
func (g *GridProxyMockClient) Farms(filter types.FarmFilter, limit types.Limit) (res []types.Farm, totalCount int, err error) {
	res = []types.Farm{}
	if limit.Page == 0 {
		limit.Page = 1
	}

	if limit.Size == 0 {
		limit.Size = 50
	}

	publicIPs := make(map[uint64][]types.PublicIP)
	for _, publicIP := range g.data.PublicIPs {
		publicIPs[g.data.FarmIDMap[publicIP.FarmID]] = append(publicIPs[g.data.FarmIDMap[publicIP.FarmID]], types.PublicIP{
			ID:         publicIP.ID,
			IP:         publicIP.IP,
			ContractID: int(publicIP.ContractID),
			Gateway:    publicIP.Gateway,
			FarmID:     publicIP.FarmID,
		})
	}

	for _, farm := range g.data.Farms {
		if farm.satisfies(filter, &g.data) {
			res = append(res, types.Farm{
				Name:              farm.Name,
				FarmID:            int(farm.FarmID),
				TwinID:            int(farm.TwinID),
				PricingPolicyID:   int(farm.PricingPolicyID),
				StellarAddress:    farm.StellarAddress,
				PublicIps:         publicIPs[farm.FarmID],
				Dedicated:         farm.DedicatedFarm,
				CertificationType: farm.Certification,
			})
		}
	}

	if filter.NodeAvailableFor != nil {
		sort.Slice(res, func(i, j int) bool {
			f1 := g.data.FarmHasRentedNode[uint64(res[i].FarmID)]
			f2 := g.data.FarmHasRentedNode[uint64(res[j].FarmID)]
			lessFarmID := res[i].FarmID < res[j].FarmID

			return f1 && !f2 || f1 && f2 && lessFarmID || !f1 && !f2 && lessFarmID
		})
	} else {
		sort.Slice(res, func(i, j int) bool {
			return res[i].FarmID < res[j].FarmID
		})
	}

	res, totalCount = getPage(res, limit)

	return
}

func (f *Farm) satisfies(filter types.FarmFilter, data *DBData) bool {
	if filter.FreeIPs != nil && *filter.FreeIPs > data.FreeIPs[f.FarmID] {
		return false
	}

	if filter.TotalIPs != nil && *filter.TotalIPs > data.TotalIPs[f.FarmID] {
		return false
	}

	if filter.StellarAddress != nil && *filter.StellarAddress != f.StellarAddress {
		return false
	}

	if filter.PricingPolicyID != nil && *filter.PricingPolicyID != f.PricingPolicyID {
		return false
	}

	if filter.FarmID != nil && *filter.FarmID != f.FarmID {
		return false
	}

	if filter.TwinID != nil && *filter.TwinID != f.TwinID {
		return false
	}

	if filter.Name != nil && !strings.EqualFold(*filter.Name, f.Name) {
		return false
	}

	if filter.NameContains != nil && !stringMatch(f.Name, *filter.NameContains) {
		return false
	}

	if filter.CertificationType != nil && *filter.CertificationType != f.Certification {
		return false
	}

	if filter.Dedicated != nil && *filter.Dedicated != f.DedicatedFarm {
		return false
	}

	if !f.satisfyFarmNodesFilter(data, filter) {
		return false
	}

	return true
}

func (f *Farm) satisfyFarmNodesFilter(data *DBData, filter types.FarmFilter) bool {
	for _, node := range data.Nodes {
		if node.FarmID != f.FarmID {
			continue
		}

		free := NodeResourcesTotal{
			HRU: data.NodesCacheMap[node.NodeID].FreeHRU,
			SRU: data.NodesCacheMap[node.NodeID].FreeSRU,
			MRU: data.NodesCacheMap[node.NodeID].FreeMRU,
		}

		if filter.NodeFreeHRU != nil && free.HRU < *filter.NodeFreeHRU {
			continue
		}

		if filter.NodeFreeMRU != nil && free.MRU < *filter.NodeFreeMRU {
			continue
		}

		if filter.NodeFreeSRU != nil && free.SRU < *filter.NodeFreeSRU {
			continue
		}

		if filter.NodeAvailableFor != nil && ((data.NodeRentedBy[node.NodeID] != 0 && data.NodeRentedBy[node.NodeID] != *filter.NodeAvailableFor) ||
			(data.NodeRentedBy[node.NodeID] != *filter.NodeAvailableFor && data.Farms[node.FarmID].DedicatedFarm)) {
			continue
		}

		_, ok := data.GPUs[node.TwinID]
		if filter.NodeHasGPU != nil && ok != *filter.NodeHasGPU {
			continue
		}

		if filter.NodeRentedBy != nil && *filter.NodeRentedBy != data.NodeRentedBy[node.NodeID] {
			continue
		}

		nodePower := types.NodePower{
			State:  node.Power.State,
			Target: node.Power.Target,
		}
		if filter.NodeStatus != nil && *filter.NodeStatus != nodestatus.DecideNodeStatus(nodePower, int64(node.UpdatedAt)) {
			continue
		}

		if filter.NodeCertified != nil && *filter.NodeCertified != (node.Certification == "Certified") {
			continue
		}

		return true
	}
	return false
}
