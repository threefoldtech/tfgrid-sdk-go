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
	healthCallCmd = "zos.system.version"
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
	queryUpNodes(ctx, c.db, c.nodeTwinIdsChan)
	for {
		select {
		case <-ticker.C:
			c.queryHealthyNodes(ctx)
			queryUpNodes(ctx, c.db, c.nodeTwinIdsChan)
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

func (c *NodeHealthIndexer) checkNodeHealth(ctx context.Context) {
	var result interface{}
	for {
		select {
		case twinId := <-c.nodeTwinIdsChan:
			subCtx, cancel := context.WithTimeout(ctx, indexerCallTimeout)
			err := c.relayClient.Call(subCtx, twinId, healthCallCmd, nil, &result)
			cancel()

			healthReport := types.HealthReport{
				NodeTwinId: twinId,
				Healthy:    isHealthy(err),
			}
			// TODO: separate this on a different channel
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
