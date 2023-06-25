package mock

import "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"

func (g *GridProxyMockClient) Counters(filter types.StatsFilter) (res types.Counters, err error) {
	res.Farms = uint64(len(g.data.Farms))
	res.Twins = uint64(len(g.data.Twins))
	res.PublicIPs = uint64(len(g.data.PublicIPs))
	res.Contracts = uint64(len(g.data.RentContracts))
	res.Contracts += uint64(len(g.data.NodeContracts))
	res.Contracts += uint64(len(g.data.NameContracts))

	distribution := map[string]uint64{}
	var gpus uint64
	for _, node := range g.data.Nodes {
		if filter.Status == nil || (*filter.Status == STATUS_UP && isUp(node.UpdatedAt)) {
			res.Nodes++
			distribution[node.Country] += 1
			res.TotalCRU += uint64(g.data.NodeTotalResources[node.NodeID].CRU)
			res.TotalMRU += uint64(g.data.NodeTotalResources[node.NodeID].MRU)
			res.TotalSRU += uint64(g.data.NodeTotalResources[node.NodeID].SRU)
			res.TotalHRU += uint64(g.data.NodeTotalResources[node.NodeID].HRU)
			if g.data.PublicConfigs[node.NodeID].IPv4 != "" || g.data.PublicConfigs[node.NodeID].IPv6 != "" {
				res.AccessNodes++
				if g.data.PublicConfigs[node.NodeID].Domain != "" {
					res.Gateways++
				}
			}
			if node.HasGPU {
				gpus++
			}
		}
	}

	res.Countries = uint64(len(distribution))
	res.NodesDistribution = distribution
	res.GPUs = gpus

	return
}
