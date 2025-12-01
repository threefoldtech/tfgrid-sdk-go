package grid

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

func GetNodes(ctx context.Context, proxyClient client.Client, filters NodeFilters) ([]types.Node, error) {
	limit := types.Limit{
		Size: 100,
		Page: 1,
	}
	var allNodes []types.Node

	var excludeIDs []uint64
	for _, id := range filters.Exclude {
		excludeIDs = append(excludeIDs, uint64(id))
	}

	for {
		var status []string
		if filters.Status != nil {
			status = []string{*filters.Status}
		}

		filter := types.NodeFilter{
			Status:   status,
			FarmIDs:  filters.FarmIDs,
			NodeIDs:  filters.NodeIDs,
			Excluded: excludeIDs,
		}

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

type NodeFilters struct {
	Status  *string
	FarmIDs []uint64
	NodeIDs []uint64
	Exclude []int
}
