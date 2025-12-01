package grid

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/provision-probe/pkg/config"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

func GetNodes(ctx context.Context, proxyClient client.Client, filters config.NodesConfig) ([]types.Node, error) {
	limit := types.Limit{
		Size: 100,
		Page: 1,
	}
	filter := buildFilters(filters)

	var allNodes []types.Node
	for {
		nodes, total, err := proxyClient.Nodes(ctx, filter, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to query nodes: %w", err)
		}

		allNodes = append(allNodes, nodes...)

		if len(allNodes) >= total || len(nodes) == 0 {
			break
		}

		limit.Page++
	}

	log.Info().Int("count", len(allNodes)).Msg("Found eligible nodes")

	return allNodes, nil
}

func buildFilters(filters config.NodesConfig) types.NodeFilter {
	filter := types.NodeFilter{
		FarmIDs:  filters.Farms,
		NodeIDs:  filters.Nodes,
		Excluded: filters.Exclude,
	}

	switch filters.Status {
	case "up":
		filter.Status = []string{"up"}
	case "healthy":
		filter.Healthy = &[]bool{true}[0]
	}

	return filter
}
