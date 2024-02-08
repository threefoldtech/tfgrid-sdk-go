package indexer

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

func queryUpNodes(ctx context.Context, database db.Database, nodeTwinIdChan chan uint32) {
	status := "up"
	filter := types.NodeFilter{
		Status: &status,
	}
	limit := types.Limit{Size: 100, Page: 1}
	hasNext := true
	for hasNext {
		nodes, _, err := database.GetNodes(ctx, filter, limit)
		if err != nil {
			log.Error().Err(err).Msg("failed to query grid nodes")
		}

		if len(nodes) < int(limit.Size) {
			hasNext = false
		}

		for _, node := range nodes {
			nodeTwinIdChan <- uint32(node.TwinID)
		}

		limit.Page++
	}
}
