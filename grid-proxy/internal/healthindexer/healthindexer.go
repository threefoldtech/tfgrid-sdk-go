package healthindexer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

const (
	indexerCallTimeout = 10 * time.Second
	indexerCallCommand = "zos.system.version"
)

type NodeHealthIndexer struct {
	db              db.Database
	relayClient     rmb.Client
	nodeTwinIdsChan chan uint32
	indexerInterval time.Duration
	indexerWorkers  uint
}

func NewNodeHealthIndexer(
	ctx context.Context,
	db db.Database,
	subManager substrate.Manager,
	mnemonic string,
	relayUrl string,
	indexerWorkers uint,
	indexerInterval uint,
) (*NodeHealthIndexer, error) {
	sessionId := generateSessionId()
	rpcClient, err := peer.NewRpcClient(ctx, peer.KeyTypeSr25519, mnemonic, relayUrl, sessionId, subManager, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create rmb client: %w", err)
	}

	return &NodeHealthIndexer{
		db:              db,
		relayClient:     rpcClient,
		nodeTwinIdsChan: make(chan uint32),
		indexerWorkers:  indexerWorkers,
		indexerInterval: time.Duration(indexerInterval) * time.Minute,
	}, nil
}

func (c *NodeHealthIndexer) Start(ctx context.Context) {

	// start the node querier, push twin-ids into chan
	go c.startNodeQuerier(ctx)

	// start the health indexer workers, pop from twin-ids chan and update the db
	for i := uint(0); i < c.indexerWorkers; i++ {
		go c.checkNodeHealth(ctx)
	}

	log.Info().Msg("Node health indexer started")

}

func (c *NodeHealthIndexer) startNodeQuerier(ctx context.Context) {
	ticker := time.NewTicker(c.indexerInterval)
	c.queryGridNodes(ctx)
	for {
		select {
		case <-ticker.C:
			c.queryGridNodes(ctx)
		case <-ctx.Done():
			return
		}
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
	}

}

func (c *NodeHealthIndexer) checkNodeHealth(ctx context.Context) {
	var result interface{}
	for twinId := range c.nodeTwinIdsChan {
		ctx, cancel := context.WithTimeout(ctx, indexerCallTimeout)

		err := c.relayClient.Call(ctx, twinId, indexerCallCommand, nil, &result)
		if isHealthy(err) {
			healthReport := types.HealthReport{
				NodeTwinId: twinId,
				Healthy:    true,
			}
			err := c.db.UpsertNodeHealth(ctx, healthReport)
			if err != nil {
				log.Error().Err(err).Msgf("failed to mark node with twin id %d as healthy", twinId)
			}
		}
		cancel()
	}
}

func isHealthy(err error) bool {
	return err == nil
}

func generateSessionId() string {
	return fmt.Sprintf("node-health-indexer-%s", strings.Split(uuid.NewString(), "-")[0])
}
