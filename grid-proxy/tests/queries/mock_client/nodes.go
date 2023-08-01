package mock

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/nodestatus"
	proxytypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"golang.org/x/exp/slices"
)

var nodeValidator = map[string]func(node Node, data *DBData, f proxytypes.NodeFilter) bool{
	"Status": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		nodePower := proxytypes.NodePower{
			State:  node.Power.State,
			Target: node.Power.Target,
		}

		return f.Status == nil || *f.Status == nodestatus.DecideNodeStatus(nodePower, int64(node.UpdatedAt))
	},
	"FreeMRU": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		total := data.NodeTotalResources[node.NodeID]
		used := data.NodeUsedResources[node.NodeID]
		free := calcFreeResources(total, used)
		return f.FreeMRU == nil || *f.FreeMRU <= free.MRU
	},
	"FreeHRU": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		total := data.NodeTotalResources[node.NodeID]
		used := data.NodeUsedResources[node.NodeID]
		free := calcFreeResources(total, used)
		return f.FreeHRU == nil || *f.FreeHRU <= free.HRU
	},
	"FreeSRU": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		total := data.NodeTotalResources[node.NodeID]
		used := data.NodeUsedResources[node.NodeID]
		free := calcFreeResources(total, used)
		return f.FreeSRU == nil || *f.FreeSRU <= free.SRU
	},
	"TotalMRU": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		total := data.NodeTotalResources[node.NodeID]
		return f.TotalMRU == nil || *f.TotalMRU <= total.MRU
	},
	"TotalHRU": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		total := data.NodeTotalResources[node.NodeID]
		return f.TotalHRU == nil || *f.TotalHRU <= total.HRU
	},
	"TotalSRU": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		total := data.NodeTotalResources[node.NodeID]
		return f.TotalSRU == nil || *f.TotalSRU <= total.SRU
	},
	"TotalCRU": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		total := data.NodeTotalResources[node.NodeID]
		return f.TotalCRU == nil || *f.TotalCRU <= total.CRU
	},
	"Country": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		return f.Country == nil || strings.EqualFold(*f.Country, node.Country)
	},
	"CountryContains": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		return f.CountryContains == nil || stringMatch(node.Country, *f.CountryContains)
	},
	"City": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		return f.City == nil || strings.EqualFold(*f.City, node.City)
	},
	"CityContains": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		return f.CityContains == nil || stringMatch(node.City, *f.CityContains)
	},
	"FarmName": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		return f.FarmName == nil || strings.EqualFold(*f.FarmName, data.Farms[node.FarmID].Name)
	},
	"FarmNameContains": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		return f.FarmNameContains == nil || stringMatch(data.Farms[node.FarmID].Name, *f.FarmNameContains)
	},
	"FarmIDs": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		return f.FarmIDs == nil || len(f.FarmIDs) == 0 || slices.Contains(f.FarmIDs, node.FarmID)
	},
	"FreeIPs": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		return f.FreeIPs == nil || *f.FreeIPs <= data.FreeIPs[node.FarmID]
	},
	"IPv4": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		return f.IPv4 == nil || *f.IPv4 != (data.PublicConfigs[node.NodeID].IPv4 == "")
	},
	"IPv6": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		return f.IPv6 == nil || *f.IPv6 != (data.PublicConfigs[node.NodeID].IPv6 == "")
	},
	"Domain": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		return f.Domain == nil || *f.Domain != (data.PublicConfigs[node.NodeID].Domain == "")
	},
	"Dedicated": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		return f.Dedicated == nil || *f.Dedicated == data.Farms[node.FarmID].DedicatedFarm
	},
	"Rentable": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		rentable := data.NodeRentedBy[node.NodeID] == 0 &&
			(data.Farms[node.FarmID].DedicatedFarm || len(data.NonDeletedContracts[node.NodeID]) == 0)
		return f.Rentable == nil || *f.Rentable == rentable
	},
	"Rented": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		_, ok := data.NodeRentedBy[node.NodeID]
		return f.Rented == nil || *f.Rented == ok
	},
	"RentedBy": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		return f.RentedBy == nil || *f.RentedBy == data.NodeRentedBy[node.NodeID]
	},
	"AvailableFor": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		renter := data.NodeRentedBy[node.NodeID]
		return f.AvailableFor == nil || renter == *f.AvailableFor || (renter == 0 && !data.Farms[node.FarmID].DedicatedFarm)
	},
	"NodeID": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		return f.NodeID == nil || *f.NodeID == node.NodeID
	},
	"TwinID": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		return f.TwinID == nil || *f.TwinID == node.TwinID
	},
	"CertificationType": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		return f.CertificationType == nil || *f.CertificationType == node.Certification
	},
	"HasGPU": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		return f.HasGPU == nil || *f.HasGPU == node.HasGPU
	},
	"GpuDeviceName": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		gpu := data.GPUs[node.TwinID]
		return f.GpuDeviceName == nil || strings.Contains(strings.ToLower(gpu.Device), *f.GpuDeviceName)
	},
	"GpuVendorName": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		gpu := data.GPUs[node.TwinID]
		return f.GpuVendorName == nil || strings.Contains(strings.ToLower(gpu.Vendor), *f.GpuVendorName)
	},
	"GpuVendorID": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		gpu := data.GPUs[node.TwinID]
		return f.GpuVendorID == nil || strings.Contains(gpu.ID, *f.GpuVendorID)
	},
	"GpuDeviceID": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		gpu := data.GPUs[node.TwinID]
		return f.GpuDeviceID == nil || strings.Contains(gpu.ID, *f.GpuDeviceID)
	},
	"GpuAvailable": func(node Node, data *DBData, f proxytypes.NodeFilter) bool {
		gpu := data.GPUs[node.TwinID]
		return f.GpuAvailable == nil || gpu.Contract == 0 == *f.GpuAvailable
	},
}

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
		satisfies, err := nodeSatisfies(&g.data, node, filter)
		if err != nil {
			return res, totalCount, err
		}

		if satisfies {
			numGPU := 0
			if _, ok := g.data.GPUs[node.TwinID]; ok {
				numGPU = 1
			}

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

func nodeSatisfies(data *DBData, node Node, f proxytypes.NodeFilter) (bool, error) {
	v := reflect.ValueOf(f)
	for i := 0; i < v.NumField(); i++ {
		valid, ok := nodeValidator[v.Type().Field(i).Name]
		if !ok {
			return false, fmt.Errorf("Field %s has no validator", v.Type().Field(i).Name)
		}

		if !valid(node, data, f) {
			return false, nil
		}
	}

	return true, nil
}

// getNumGPUs should be deleted after removing hasGPU
func getNumGPUs(hasGPU bool) int {

	if hasGPU {
		return 1
	}
	return 0
}
