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
	database        db.Database
	rmbClient       *peer.RpcClient
	nodeTwinIdsChan chan uint32
	resultChan      chan types.HealthReport
	batchChan       chan []types.HealthReport
	indexerInterval time.Duration
	indexerWorkers  uint
	batchSize       uint
}

func NewNodeHealthIndexer(
	rpcClient *peer.RpcClient,
	database db.Database,
	batchSize uint,
	indexerWorkers uint,
	indexerInterval uint,
) *NodeHealthIndexer {
	return &NodeHealthIndexer{
		database:        database,
		rmbClient:       rpcClient,
		nodeTwinIdsChan: make(chan uint32),
		resultChan:      make(chan types.HealthReport),
		batchChan:       make(chan []types.HealthReport),
		batchSize:       batchSize,
		indexerWorkers:  indexerWorkers,
		indexerInterval: time.Duration(indexerInterval) * time.Minute,
	}
}

func (c *NodeHealthIndexer) Start(ctx context.Context) {
	go c.StartNodeFinder(ctx)

	for i := uint(0); i < c.indexerWorkers; i++ {
		go c.StartNodeCaller(ctx)
	}

	for i := uint(0); i < c.indexerWorkers; i++ {
		go c.StartResultBatcher(ctx)
	}

	go c.StartBatchUpserter(ctx)
}

func (c *NodeHealthIndexer) StartNodeFinder(ctx context.Context) {
	ticker := time.NewTicker(c.indexerInterval)

	queryHealthyNodes(ctx, c.database, c.nodeTwinIdsChan) // to revalidate the reports if node went down
	queryUpNodes(ctx, c.database, c.nodeTwinIdsChan)
	for {
		select {
		case <-ticker.C:
			queryHealthyNodes(ctx, c.database, c.nodeTwinIdsChan)
			queryUpNodes(ctx, c.database, c.nodeTwinIdsChan)
		case <-ctx.Done():
			return
		}
	}
}

func (c *NodeHealthIndexer) StartNodeCaller(ctx context.Context) {
	for {
		select {
		case twinId := <-c.nodeTwinIdsChan:
			var response types.HealthReport
			err := callNode(ctx, c.rmbClient, healthCallCmd, nil, twinId, &response)
			c.resultChan <- getHealthReport(response, err, twinId)
		case <-ctx.Done():
			return
		}
	}
}

func (c *NodeHealthIndexer) StartResultBatcher(ctx context.Context) {
	buffer := make([]types.HealthReport, 0, c.batchSize)

	ticker := time.NewTicker(flushingBufferInterval)
	for {
		select {
		case report := <-c.resultChan:
			buffer = append(buffer, report)
			if len(buffer) >= int(c.batchSize) {
				c.batchChan <- buffer
				buffer = nil
			}
		case <-ticker.C:
			if len(buffer) != 0 {
				c.batchChan <- buffer
				buffer = nil
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *NodeHealthIndexer) StartBatchUpserter(ctx context.Context) {
	for {
		select {
		case batch := <-c.batchChan:
			err := c.database.UpsertNodeHealth(ctx, batch)
			if err != nil {
				log.Error().Err(err).Msg("failed to upsert node health")
			}
		case <-ctx.Done():
			return
		}
	}
}

func getHealthReport(response interface{}, err error, twinId uint32) types.HealthReport {
	report := types.HealthReport{
		NodeTwinId: twinId,
		Healthy:    false,
	}

	if err != nil {
		return report
	}

	report.Healthy = true
	return report
}
