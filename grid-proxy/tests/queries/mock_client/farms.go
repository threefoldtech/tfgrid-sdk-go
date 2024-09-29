package mock

import (
	"context"
	"sort"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/nodestatus"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"golang.org/x/exp/slices"
)

// Farms returns farms with the given filters and pagination parameters
func (g *GridProxyMockClient) Farms(ctx context.Context, filter types.FarmFilter, limit types.Limit) (res []types.Farm, totalCount int, err error) {
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
			ContractID: publicIP.ContractID,
			Gateway:    publicIP.Gateway,
		})
	}

	for _, farm := range g.data.Farms {
		if farm.satisfies(filter, &g.data) {
			ips := []types.PublicIP{}
			if pubIPs, ok := publicIPs[farm.FarmID]; ok {
				ips = pubIPs
			}
			res = append(res, types.Farm{
				Name:              farm.Name,
				FarmID:            int(farm.FarmID),
				TwinID:            int(farm.TwinID),
				PricingPolicyID:   int(farm.PricingPolicyID),
				StellarAddress:    farm.StellarAddress,
				PublicIps:         ips,
				Dedicated:         farm.DedicatedFarm,
				CertificationType: farm.Certification,
			})
		}
	}

	if filter.NodeAvailableFor != nil {
		sort.Slice(res, func(i, j int) bool {
			f1 := g.data.FarmHasRentedNode[uint64(res[i].FarmID)][*filter.NodeAvailableFor]
			f2 := g.data.FarmHasRentedNode[uint64(res[j].FarmID)][*filter.NodeAvailableFor]
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

	if filter.NodeAvailableFor != nil || filter.NodeCertified != nil ||
		filter.NodeFreeHRU != nil || filter.NodeFreeMRU != nil ||
		filter.NodeFreeSRU != nil || filter.NodeHasGPU != nil ||
		filter.NodeRentedBy != nil || len(filter.NodeStatus) != 0 ||
		filter.Country != nil || filter.Region != nil ||
		filter.NodeTotalCRU != nil || filter.NodeHasIpv6 != nil ||
		filter.NodeWGSupported != nil || filter.NodeYggSupported != nil ||
		filter.NodePubIpSupported != nil {
		if !f.satisfyFarmNodesFilter(data, filter) {
			return false
		}
	}

	return true
}

func (f *Farm) satisfyFarmNodesFilter(data *DBData, filter types.FarmFilter) bool {
	for _, node := range data.Nodes {
		if node.FarmID != f.FarmID {
			continue
		}

		total := data.NodeTotalResources[node.NodeID]
		used := data.NodeUsedResources[node.NodeID]
		free := CalcFreeResources(total, used)
		if filter.NodeFreeHRU != nil && int64(free.HRU) < int64(*filter.NodeFreeHRU) {
			continue
		}

		if filter.NodeFreeMRU != nil && int64(free.MRU) < int64(*filter.NodeFreeMRU) {
			continue
		}

		if filter.NodeFreeSRU != nil && int64(free.SRU) < int64(*filter.NodeFreeSRU) {
			continue
		}

		if filter.NodeTotalCRU != nil && total.CRU < *filter.NodeTotalCRU {
			continue
		}

		if filter.NodeAvailableFor != nil && ((data.NodeRentedBy[node.NodeID] != 0 && data.NodeRentedBy[node.NodeID] != *filter.NodeAvailableFor) ||
			(data.NodeRentedBy[node.NodeID] != *filter.NodeAvailableFor && (data.Farms[node.FarmID].DedicatedFarm || node.ExtraFee != 0))) {
			continue
		}

		_, ok := data.GPUs[uint32(node.TwinID)]
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
		nodeStatus := nodestatus.DecideNodeStatus(nodePower, int64(node.UpdatedAt))
		if len(filter.NodeStatus) != 0 && !slices.Contains(filter.NodeStatus, nodeStatus) {
			continue
		}

		if filter.NodeCertified != nil && *filter.NodeCertified != (node.Certification == "Certified") {
			continue
		}

		if filter.Country != nil && !strings.EqualFold(*filter.Country, node.Country) {
			continue
		}

		if filter.Region != nil && !strings.EqualFold(*filter.Region, data.Regions[strings.ToLower(node.Country)]) {
			continue
		}

		if filter.NodeHasIpv6 != nil && *filter.NodeHasIpv6 != data.NodeIpv6[uint32(node.TwinID)] {
			continue
		}

		if filter.NodeWGSupported != nil && *filter.NodeWGSupported == data.NodeLight[uint32(node.TwinID)] {
			continue
		}
		if filter.NodeYggSupported != nil && *filter.NodeYggSupported == data.NodeLight[uint32(node.TwinID)] {
			continue
		}
		if filter.NodePubIpSupported != nil && *filter.NodePubIpSupported == data.NodeLight[uint32(node.TwinID)] {
			continue
		}

		return true
	}
	return false
}
