package mock

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/nodestatus"
	proxytypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

var farmFilterFieldValidator = map[string]func(farm Farm, data *DBData, f proxytypes.FarmFilter) bool{
	"FreeIPs": func(farm Farm, data *DBData, f proxytypes.FarmFilter) bool {
		return f.FreeIPs == nil || *f.FreeIPs <= data.FreeIPs[farm.FarmID]
	},
	"TotalIPs": func(farm Farm, data *DBData, f proxytypes.FarmFilter) bool {
		return f.TotalIPs == nil || *f.TotalIPs <= data.TotalIPs[farm.FarmID]
	},
	"StellarAddress": func(farm Farm, data *DBData, f proxytypes.FarmFilter) bool {
		return f.StellarAddress == nil || *f.StellarAddress == farm.StellarAddress
	},
	"PricingPolicyID": func(farm Farm, data *DBData, f proxytypes.FarmFilter) bool {
		return f.PricingPolicyID == nil || *f.PricingPolicyID == farm.PricingPolicyID
	},
	"FarmID": func(farm Farm, data *DBData, f proxytypes.FarmFilter) bool {
		return f.FarmID == nil || *f.FarmID == farm.FarmID
	},
	"TwinID": func(farm Farm, data *DBData, f proxytypes.FarmFilter) bool {
		return f.TwinID == nil || *f.TwinID == farm.TwinID
	},
	"Name": func(farm Farm, data *DBData, f proxytypes.FarmFilter) bool {
		return f.Name == nil || *f.Name == "" || strings.EqualFold(*f.Name, farm.Name)
	},
	"NameContains": func(farm Farm, data *DBData, f proxytypes.FarmFilter) bool {
		return f.NameContains == nil || *f.NameContains == "" || stringMatch(farm.Name, *f.NameContains)
	},
	"CertificationType": func(farm Farm, data *DBData, f proxytypes.FarmFilter) bool {
		return f.CertificationType == nil || *f.CertificationType == "" || *f.CertificationType == farm.Certification
	},
	"Dedicated": func(farm Farm, data *DBData, f proxytypes.FarmFilter) bool {
		return f.Dedicated == nil || *f.Dedicated == farm.DedicatedFarm
	},
	"NodeFreeMRU": func(farm Farm, data *DBData, f proxytypes.FarmFilter) bool {
		return satisfyFarmResourceFilter(farm, data, f)
	},
	"NodeFreeHRU": func(farm Farm, data *DBData, f proxytypes.FarmFilter) bool {
		return satisfyFarmResourceFilter(farm, data, f)
	},
	"NodeFreeSRU": func(farm Farm, data *DBData, f proxytypes.FarmFilter) bool {
		return satisfyFarmResourceFilter(farm, data, f)
	},
}

// Farms returns farms with the given filters and pagination parameters
func (g *GridProxyMockClient) Farms(filter proxytypes.FarmFilter, limit proxytypes.Limit) (res []proxytypes.Farm, totalCount int, err error) {
	res = []proxytypes.Farm{}
	if limit.Page == 0 {
		limit.Page = 1
	}

	if limit.Size == 0 {
		limit.Size = 50
	}

	publicIPs := make(map[uint64][]proxytypes.PublicIP)
	for _, publicIP := range g.data.PublicIPs {
		publicIPs[g.data.FarmIDMap[publicIP.FarmID]] = append(publicIPs[g.data.FarmIDMap[publicIP.FarmID]], proxytypes.PublicIP{
			ID:         publicIP.ID,
			IP:         publicIP.IP,
			ContractID: int(publicIP.ContractID),
			Gateway:    publicIP.Gateway,
		})
	}

	for _, farm := range g.data.Farms {
		satisfies, err := farmSatisfies(&g.data, farm, filter)
		if err != nil {
			return res, totalCount, err
		}

		if satisfies {
			res = append(res, proxytypes.Farm{
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

func farmSatisfies(data *DBData, farm Farm, f proxytypes.FarmFilter) (bool, error) {
	v := reflect.ValueOf(f)
	for i := 0; i < v.NumField(); i++ {
		valid, ok := farmFilterFieldValidator[v.Type().Field(i).Name]
		if !ok {
			return false, fmt.Errorf("Field %s has no validator", v.Type().Field(i).Name)
		}

		if !valid(farm, data, f) {
			return false, nil
		}
	}

	return true, nil
}

func satisfyFarmResourceFilter(farm Farm, data *DBData, f proxytypes.FarmFilter) bool {
	for _, val := range data.Nodes {
		if val.FarmID != farm.FarmID {
			continue
		}
		total := data.NodeTotalResources[val.NodeID]
		used := data.NodeUsedResources[val.NodeID]
		free := calcFreeResources(total, used)
		if f.NodeFreeHRU != nil && free.HRU < *f.NodeFreeHRU {
			continue
		}
		if f.NodeFreeMRU != nil && free.MRU < *f.NodeFreeMRU {
			continue
		}
		if f.NodeFreeSRU != nil && free.SRU < *f.NodeFreeSRU {
			continue
		}
		if f.NodeAvailableFor != nil && ((data.NodeRentedBy[val.NodeID] != 0 && data.NodeRentedBy[val.NodeID] != *f.NodeAvailableFor) ||
			(data.NodeRentedBy[val.NodeID] != *f.NodeAvailableFor && data.Farms[val.FarmID].DedicatedFarm)) {
			continue
		}

		_, ok := data.GPUs[val.TwinID]
		if f.NodeHasGPU != nil && ok != *f.NodeHasGPU {
			continue
		}

		if f.NodeRentedBy != nil && *f.NodeRentedBy != data.NodeRentedBy[val.NodeID] {
			continue
		}

		nodePower := proxytypes.NodePower{
			State:  val.Power.State,
			Target: val.Power.Target,
		}
		if f.NodeStatus != nil && *f.NodeStatus != nodestatus.DecideNodeStatus(nodePower, int64(val.UpdatedAt)) {
			continue
		}
		return true
	}
	return false
}
