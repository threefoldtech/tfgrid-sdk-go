package indexer

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

const (
	indexerCallTimeout = 10 * time.Second
	indexerCallCommand = "zos.system.version"
)

type NodeHealthIndexer struct {
	db              db.Database
	relayClient     *peer.RpcClient
	nodeTwinIdsChan chan uint32
	indexerInterval time.Duration
	indexerWorkers  uint
}

func NewNodeHealthIndexer(
	ctx context.Context,
	rpcClient *peer.RpcClient,
	db db.Database,
	indexerWorkers uint,
	indexerInterval uint,
) *NodeHealthIndexer {
	return &NodeHealthIndexer{
		db:              db,
		relayClient:     rpcClient,
		nodeTwinIdsChan: make(chan uint32),
		indexerWorkers:  indexerWorkers,
		indexerInterval: time.Duration(indexerInterval) * time.Minute,
	}
}

func (c *NodeHealthIndexer) Start(ctx context.Context) {

	// start the node querier, push twin-ids into chan
	go c.startNodeQuerier(ctx)

	// start the health indexer workers, pop from twin-ids chan and update the db
	for i := uint(0); i < c.indexerWorkers; i++ {
		go c.checkNodeHealth(ctx)
	}

}

func (c *NodeHealthIndexer) startNodeQuerier(ctx context.Context) {
	ticker := time.NewTicker(c.indexerInterval)
	c.queryHealthyNodes(ctx)
	c.queryGridNodes(ctx)
	for {
		select {
		case <-ticker.C:
			c.queryHealthyNodes(ctx)
			c.queryGridNodes(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// to revalidate the reports
func (c *NodeHealthIndexer) queryHealthyNodes(ctx context.Context) {
	ids, err := c.db.GetHealthyNodeTwinIds(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to query healthy nodes")
	}

	for _, id := range ids {
		c.nodeTwinIdsChan <- uint32(id)
	}
}

func (c *NodeHealthIndexer) queryGridNodes(ctx context.Context) {
	status := "up"
	filter := types.NodeFilter{
		Status: &status,
	}

	limit := types.Limit{
		Size:     100,
		RetCount: true,
		Page:     1,
	}

	hasNext := true
	for hasNext {
		nodes, _, err := c.db.GetNodes(ctx, filter, limit)
		if err != nil {
			log.Error().Err(err).Msg("failed to query grid nodes")
		}

		if len(nodes) < int(limit.Size) {
			hasNext = false
		}

		for _, node := range nodes {
			c.nodeTwinIdsChan <- uint32(node.TwinID)
		}

		limit.Page++
	}

}

func (c *NodeHealthIndexer) checkNodeHealth(ctx context.Context) {
	var result interface{}
	for {
		select {
		case twinId := <-c.nodeTwinIdsChan:
			subCtx, cancel := context.WithTimeout(ctx, indexerCallTimeout)
			err := c.relayClient.Call(subCtx, twinId, indexerCallCommand, nil, &result)
			cancel()

			log.Debug().Msgf("health indexer: %+v", result)

			healthReport := types.HealthReport{
				NodeTwinId: twinId,
				Healthy:    isHealthy(err),
			}
			err = c.db.UpsertNodeHealth(ctx, healthReport)
			if err != nil {
				log.Error().Err(err).Msgf("failed to update health report for node with twin id %d", twinId)
			}
		case <-ctx.Done():
			return
		}
	}
}

func isHealthy(err error) bool {
	return err == nil
}

// func generateSessionId() string {
// 	return fmt.Sprintf("node-health-indexer-%s", strings.Split(uuid.NewString(), "-")[0])
// }
