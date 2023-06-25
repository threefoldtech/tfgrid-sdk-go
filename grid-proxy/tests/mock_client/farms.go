package mock

import (
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// Farms returns farms with the given filters and pagination parameters
func (g *GridProxyMockClient) Farms(filter types.FarmFilter, limit types.Limit) ([]types.Farm, int, error) {

	publicIPs := g.getFarmPublicIPs()
	farms, err := g.filterFarms(filter, publicIPs)
	if err != nil {
		return nil, 0, err
	}

	sort.Slice(farms, func(i, j int) bool {
		return farms[i].FarmID < farms[j].FarmID
	})

	res, count := getPage(farms, limit)

	return res, count, nil
}

func (g *GridProxyMockClient) filterFarms(filter types.FarmFilter, publicIPs map[uint32][]types.PublicIP) ([]types.Farm, error) {
	res := []types.Farm{}

	for _, farm := range g.data.Farms {
		satisfies, err := farm.satisfies(&g.data, filter)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to check farm %d eligibility", farm.FarmID)
		}

		if satisfies {
			res = append(res, types.Farm{
				Name:              farm.Name,
				FarmID:            farm.FarmID,
				TwinID:            farm.TwinID,
				PricingPolicyID:   farm.PricingPolicyID,
				StellarAddress:    farm.StellarAddress,
				PublicIps:         publicIPs[farm.FarmID],
				Dedicated:         farm.DedicatedFarm,
				CertificationType: farm.Certification,
			})
		}
	}

	return res, nil
}

func (g *GridProxyMockClient) getFarmPublicIPs() map[uint32][]types.PublicIP {
	publicIPs := make(map[uint32][]types.PublicIP)
	for _, publicIP := range g.data.PublicIPs {
		publicIPs[g.data.FarmIDMap[publicIP.FarmID]] = append(publicIPs[g.data.FarmIDMap[publicIP.FarmID]], types.PublicIP{
			ID:         publicIP.ID,
			IP:         publicIP.IP,
			ContractID: publicIP.ContractID,
			Gateway:    publicIP.Gateway,
		})
	}

	return publicIPs
}

func (farm *DBFarm) satisfies(data *DBData, f types.FarmFilter) (bool, error) {
	if f.FreeIPs != nil && *f.FreeIPs > data.FreeIPs[farm.FarmID] {
		return false, nil
	}

	if f.TotalIPs != nil && *f.TotalIPs > data.TotalIPs[farm.FarmID] {
		return false, nil
	}

	if f.StellarAddress != nil && *f.StellarAddress != farm.StellarAddress {
		return false, nil
	}

	if f.PricingPolicyID != nil && *f.PricingPolicyID != farm.PricingPolicyID {
		return false, nil
	}

	if f.FarmID != nil && *f.FarmID != farm.FarmID {
		return false, nil
	}

	if f.TwinID != nil && *f.TwinID != farm.TwinID {
		return false, nil
	}

	if f.NameContains != nil && *f.NameContains != "" && !stringMatch(farm.Name, *f.NameContains) {
		return false, nil
	}

	if f.Name != nil && *f.Name != "" && !strings.EqualFold(*f.Name, farm.Name) {
		return false, nil
	}

	if f.NameContains != nil && *f.NameContains != "" && !stringMatch(farm.Name, *f.NameContains) {
		return false, nil
	}

	if f.CertificationType != nil && *f.CertificationType != "" && *f.CertificationType != farm.Certification {
		return false, nil
	}

	if f.Dedicated != nil && *f.Dedicated != farm.DedicatedFarm {
		return false, nil
	}

	if f.NodeFreeHRU != nil || f.NodeFreeMRU != nil || f.NodeFreeSRU != nil {
		satisfies, err := farm.satisfyResource(data, f)
		if err != nil {
			return false, err
		}

		if !satisfies {
			return false, nil
		}
	}

	return true, nil
}

func (farm *DBFarm) satisfyResource(data *DBData, f types.FarmFilter) (bool, error) {
	for _, val := range data.Nodes {
		if val.FarmID != farm.FarmID {
			continue
		}

		total := data.NodeTotalResources[val.NodeID]
		used := data.NodeUsedResources[val.NodeID]
		free, err := CalculateFreeResources(total, used)
		if err != nil {
			return false, err
		}

		if f.NodeFreeHRU != nil && free.HRU < *f.NodeFreeHRU {
			continue
		}

		if f.NodeFreeMRU != nil && free.MRU < *f.NodeFreeMRU {
			continue
		}

		if f.NodeFreeSRU != nil && free.SRU < *f.NodeFreeSRU {
			continue
		}

		return true, nil
	}

	return false, nil
}
