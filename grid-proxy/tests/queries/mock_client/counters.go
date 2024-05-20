package mock

import (
	"context"
	"slices"

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
	dedicatedNodesCount := int64(0)
	workloadsNumber := uint32(0)
	var gpus int64
	for _, node := range g.data.Nodes {
		nodePower := types.NodePower{
			State:  node.Power.State,
			Target: node.Power.Target,
		}
		st := nodestatus.DecideNodeStatus(nodePower, int64(node.UpdatedAt))
		if filter.Status == nil || len(filter.Status) == 0 || slices.Contains(filter.Status, st) {
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
			if _, ok := g.data.GPUs[uint32(node.TwinID)]; ok {
				gpus++
			}
			if isDedicatedNode(g.data, node) {
				dedicatedNodesCount++
			}
			workloadsNumber += g.data.WorkloadsNumbers[uint32(node.TwinID)]
		}
	}
	res.Countries = int64(len(distribution))
	res.NodesDistribution = distribution
	res.GPUs = gpus
	res.DedicatedNodes = dedicatedNodesCount
	res.WorkloadsNumber = workloadsNumber
	return
}
