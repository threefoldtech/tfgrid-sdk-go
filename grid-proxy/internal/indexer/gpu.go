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
	gpuListCmd = "zos.gpu.list"
)

type NodeGPUIndexer struct {
	database        db.Database
	rmbClient       *peer.RpcClient
	interval        time.Duration
	workers         uint
	batchSize       uint
	nodeTwinIdsChan chan uint32
	resultChan      chan types.NodeGPU
	batchChan       chan []types.NodeGPU
}

func NewGPUIndexer(
	rmbClient *peer.RpcClient,
	database db.Database,
	batchSize uint,
	interval uint,
	workers uint,
) *NodeGPUIndexer {
	return &NodeGPUIndexer{
		database:        database,
		rmbClient:       rmbClient,
		batchSize:       batchSize,
		workers:         workers,
		interval:        time.Duration(interval) * time.Minute,
		nodeTwinIdsChan: make(chan uint32),
		resultChan:      make(chan types.NodeGPU),
		batchChan:       make(chan []types.NodeGPU),
	}
}

func (n *NodeGPUIndexer) Start(ctx context.Context) {
	go n.StartNodeFinder(ctx)
	go n.startNodeTableWatcher(ctx)

	for i := uint(0); i < n.workers; i++ {
		go n.StartNodeCaller(ctx)
	}

	for i := uint(0); i < n.workers; i++ {
		go n.StartResultBatcher(ctx)
	}

	go n.StartBatchUpserter(ctx)
}

func (n *NodeGPUIndexer) StartNodeFinder(ctx context.Context) {
	ticker := time.NewTicker(n.interval)
	queryUpNodes(ctx, n.database, n.nodeTwinIdsChan)
	for {
		select {
		case <-ticker.C:
			queryUpNodes(ctx, n.database, n.nodeTwinIdsChan)
		case <-ctx.Done():
			return
		}
	}
}

func (n *NodeGPUIndexer) startNodeTableWatcher(ctx context.Context) {
	ticker := time.NewTicker(newNodesCheckInterval)
	latestCheckedID, err := n.database.GetLastNodeTwinID(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get last node twin id")
	}

	for {
		select {
		case <-ticker.C:
			newIDs, err := n.database.GetNodeTwinIDsAfter(ctx, latestCheckedID)
			if err != nil {
				log.Error().Err(err).Msgf("failed to get node twin ids after %d", latestCheckedID)
				continue
			}
			if len(newIDs) == 0 {
				continue
			}

			latestCheckedID = newIDs[0]
			for _, id := range newIDs {
				n.nodeTwinIdsChan <- id
			}
		case <-ctx.Done():
			return
		}
	}
}

func (n *NodeGPUIndexer) StartNodeCaller(ctx context.Context) {
	for {
		select {
		case twinId := <-n.nodeTwinIdsChan:
			var gpus []types.NodeGPU
			err := callNode(ctx, n.rmbClient, gpuListCmd, nil, twinId, &gpus)
			if err != nil {
				continue
			}

			for i := 0; i < len(gpus); i++ {
				gpus[i].NodeTwinID = twinId
				gpus[i].UpdatedAt = time.Now().Unix()
				n.resultChan <- gpus[i]
			}
		case <-ctx.Done():
			return
		}
	}
}

func (n *NodeGPUIndexer) StartResultBatcher(ctx context.Context) {
	buffer := make([]types.NodeGPU, 0, n.batchSize)

	ticker := time.NewTicker(flushingBufferInterval)
	for {
		select {
		case gpus := <-n.resultChan:
			buffer = append(buffer, gpus)
			if len(buffer) >= int(n.batchSize) {
				n.batchChan <- buffer
				buffer = nil
			}
		case <-ticker.C:
			if len(buffer) != 0 {
				n.batchChan <- buffer
				buffer = nil
			}
		case <-ctx.Done():
			return
		}
	}
}
func (n *NodeGPUIndexer) StartBatchUpserter(ctx context.Context) {
	for {
		select {
		case batch := <-n.batchChan:
			err := discardOldGpus(ctx, n.database, n.interval, batch)
			if err != nil {
				log.Error().Err(err).Msg("failed to remove old GPUs")
			}

			err = n.database.UpsertNodesGPU(ctx, batch)
			if err != nil {
				log.Error().Err(err).Msg("failed to upsert new GPUs")
			}
		case <-ctx.Done():
			return
		}
	}
}

func discardOldGpus(ctx context.Context, database db.Database, interval time.Duration, gpuBatch []types.NodeGPU) error {
	// invalidate the old indexed GPUs for the same node,
	// but check the batch first to ensure it does not contain related GPUs to node twin it from the last batch.
	// TODO: if timestamp > 1
	nodeTwinIds := []uint32{}
	for _, gpu := range gpuBatch {
		nodeTwinIds = append(nodeTwinIds, gpu.NodeTwinID)
	}

	expiration := time.Now().Unix() - int64(interval.Seconds())
	return database.DeleteOldGpus(ctx, nodeTwinIds, expiration)
}
