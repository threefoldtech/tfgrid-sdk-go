package mock

import (
	"context"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/nodestatus"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

func (g *GridProxyMockClient) Stats(ctx context.Context, filter types.StatsFilter) (res types.Stats, err error) {
	res.Farms = int64(len(g.data.Farms))
	res.Twins = int64(len(g.data.Twins))
	res.PublicIPs = int64(len(g.data.PublicIPs))
	res.Contracts = int64(len(g.data.RentContracts))
	res.Contracts += int64(len(g.data.NodeContracts))
	res.Contracts += int64(len(g.data.NameContracts))
	distribution := map[string]int64{}
	var gpus int64
	for _, node := range g.data.Nodes {
		nodePower := types.NodePower{
			State:  node.Power.State,
			Target: node.Power.Target,
		}
		if filter.Status == nil || *filter.Status == nodestatus.DecideNodeStatus(nodePower, int64(node.UpdatedAt)) {
			res.Nodes++
			distribution[node.Country] += 1
			res.TotalCRU += int64(g.data.NodeTotalResources[node.NodeID].CRU)
			res.TotalMRU += int64(g.data.NodeTotalResources[node.NodeID].MRU)
			res.TotalSRU += int64(g.data.NodeTotalResources[node.NodeID].SRU)
			res.TotalHRU += int64(g.data.NodeTotalResources[node.NodeID].HRU)
			if g.data.PublicConfigs[node.NodeID].IPv4 != "" || g.data.PublicConfigs[node.NodeID].IPv6 != "" {
				res.AccessNodes++
				if g.data.PublicConfigs[node.NodeID].Domain != "" {
					res.Gateways++
				}
			}
			if _, ok := g.data.GPUs[node.TwinID]; ok {
				gpus++
			}
		}
	}
	res.Countries = int64(len(distribution))
	res.NodesDistribution = distribution
	res.GPUs = gpus

	return
}
