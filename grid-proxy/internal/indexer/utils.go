package indexer

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

func queryUpNodes(ctx context.Context, database db.Database, nodeTwinIdChan chan uint32) {
	filter := types.NodeFilter{
		Status: []string{"up"},
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

func queryHealthyNodes(ctx context.Context, database db.Database, nodeTwinIdChan chan uint32) {
	ids, err := database.GetHealthyNodeTwinIds(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to query healthy nodes")
	}

	for _, id := range ids {
		nodeTwinIdChan <- id
	}
}

func callNode(ctx context.Context, rmbClient *peer.RpcClient, cmd string, payload interface{}, twinId uint32, result interface{}) error {
	subCtx, cancel := context.WithTimeout(ctx, indexerCallTimeout)
	defer cancel()

	return rmbClient.Call(subCtx, twinId, cmd, payload, result)
}
