package indexer

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos_sdk_go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/zos_sdk_go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos_sdk_go/rmb-sdk-go/peer"
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

func removeDuplicates[T any, K comparable](items []T, keyFunc func(T) K) (result []T) {
	seen := make(map[K]bool)
	for _, item := range items {
		key := keyFunc(item)
		if _, ok := seen[key]; !ok {
			seen[key] = true
			result = append(result, item)
		}
	}
	return
}
