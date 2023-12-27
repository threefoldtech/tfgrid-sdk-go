package gpuindexer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
	rmbTypes "github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer/types"
)

const (
	resultsBatcherCleanupInterval = 10 * time.Second
	minListenerReconnectInterval  = 10 * time.Second
	lingerBatch                   = 10 * time.Second
	newNodesCheckInterval         = 5 * time.Minute
)

type NodeGPUIndexer struct {
	db                     db.Database
	relayPeer              *peer.Peer
	checkInterval          time.Duration
	batchSize              int
	nodesGPUResultsChan    chan []types.NodeGPU
	nodesGPUBatchesChan    chan []types.NodeGPU
	newNodeTwinIDChan      chan []uint32
	nodesGPUResultsWorkers int
	nodesGPUBufferWorkers  int
}

func NewNodeGPUIndexer(
	ctx context.Context,
	relayURL,
	mnemonics string,
	subManager substrate.Manager,
	db db.Database,
	indexerCheckIntervalMins,
	batchSize,
	nodesGPUResultsWorkers,
	nodesGPUBufferWorkers int) (*NodeGPUIndexer, error) {
	indexer := &NodeGPUIndexer{
		db:                     db,
		nodesGPUResultsChan:    make(chan []types.NodeGPU),
		nodesGPUBatchesChan:    make(chan []types.NodeGPU),
		newNodeTwinIDChan:      make(chan []uint32),
		checkInterval:          time.Duration(indexerCheckIntervalMins) * time.Minute,
		batchSize:              batchSize,
		nodesGPUResultsWorkers: nodesGPUResultsWorkers,
		nodesGPUBufferWorkers:  nodesGPUBufferWorkers,
	}

	sessionId := fmt.Sprintf("tfgrid_proxy_indexer-%d", os.Getpid())
	client, err := peer.NewPeer(
		ctx,
		mnemonics,
		subManager,
		indexer.relayCallback,
		peer.WithRelay(relayURL),
		peer.WithSession(sessionId),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create direct RMB client: %w", err)
	}
	indexer.relayPeer = client

	return indexer, nil
}

func (n *NodeGPUIndexer) queryGridNodes(ctx context.Context) {
	ticker := time.NewTicker(n.checkInterval)
	n.runQueryGridNodes(ctx)
	for {
		select {
		case <-ticker.C:
			n.runQueryGridNodes(ctx)
		case twinIDs := <-n.newNodeTwinIDChan:
			n.queryNewNodes(ctx, twinIDs)
		case <-ctx.Done():
			return
		}
	}
}

func (n *NodeGPUIndexer) queryNewNodes(ctx context.Context, twinIDs []uint32) {
	for _, twinID := range twinIDs {
		err := n.getNodeGPUInfo(ctx, twinID)
		log.Error().Err(err).Msgf("failed to send get GPU info request from relay in GPU indexer for node %d", twinID)
	}
}

func (n *NodeGPUIndexer) runQueryGridNodes(ctx context.Context) {
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
		nodes, err := n.getNodes(ctx, filter, limit)
		if err != nil {
			log.Error().Err(err).Msg("unable to query nodes in GPU indexer")
			return
		}

		if len(nodes) < int(limit.Size) {
			hasNext = false
		}

		for _, node := range nodes {
			if err := n.getNodeGPUInfo(ctx, uint32(node.TwinID)); err != nil {
				log.Error().Err(err).Msgf("failed to send get GPU info request from relay in GPU indexer for node %d", node.NodeID)
			}
		}

		limit.Page++
	}
}

func (n *NodeGPUIndexer) getNodeGPUInfo(ctx context.Context, nodeTwinID uint32) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	id := uuid.NewString()
	return n.relayPeer.SendRequest(ctx, id, nodeTwinID, nil, "zos.gpu.list", nil)
}

func (n *NodeGPUIndexer) getNodes(ctx context.Context, filter types.NodeFilter, limit types.Limit) ([]db.Node, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	nodes, _, err := n.db.GetNodes(ctx, filter, limit)
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func (n *NodeGPUIndexer) gpuBatchesDBUpserter(ctx context.Context) {
	for {
		select {
		case gpuBatch := <-n.nodesGPUBatchesChan:
			err := n.db.UpsertNodesGPU(ctx, gpuBatch)
			if err != nil {
				log.Error().Err(err).Msg("failed to update GPU info in GPU indexer")
				continue
			}
		case <-ctx.Done():
			log.Error().Err(ctx.Err()).Msg("Nodes GPU DB Upserter exited")
			return
		}
	}
}

func (n *NodeGPUIndexer) gpuNodeResultsBatcher(ctx context.Context) {
	nodesGPUBuffer := make([]types.NodeGPU, 0, n.batchSize)
	ticker := time.NewTicker(lingerBatch)
	for {
		select {
		case nodesGPU := <-n.nodesGPUResultsChan:
			nodesGPUBuffer = append(nodesGPUBuffer, nodesGPU...)
			if len(nodesGPUBuffer) >= n.batchSize {
				log.Debug().Msg("flushing gpu indexer buffer")
				n.nodesGPUBatchesChan <- nodesGPUBuffer
				nodesGPUBuffer = nil
			}
		// This case covers flushing data when the limit for the batch wasn't met
		case <-ticker.C:
			if len(nodesGPUBuffer) != 0 {
				log.Debug().Msg("cleaning up gpu indexer buffer")
				n.nodesGPUBatchesChan <- nodesGPUBuffer
				nodesGPUBuffer = nil
			}
		case <-ctx.Done():
			log.Error().Err(ctx.Err()).Msg("Node GPU results batcher exited")
			return
		}
	}
}

func (n *NodeGPUIndexer) Start(ctx context.Context) {
	for i := 0; i < n.nodesGPUResultsWorkers; i++ {
		go n.gpuNodeResultsBatcher(ctx)
	}

	for i := 0; i < n.nodesGPUBufferWorkers; i++ {
		go n.gpuBatchesDBUpserter(ctx)
	}

	go n.queryGridNodes(ctx)

	go n.watchNodeTable(ctx)

	log.Info().Msg("GPU indexer started")

}

func (n *NodeGPUIndexer) watchNodeTable(ctx context.Context) {
	ticker := time.NewTicker(newNodesCheckInterval)
	latestCheckedID, err := n.db.GetLastNodeTwinID(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get last node twin id")
	}
	for {
		select {
		case <-ticker.C:
			newIDs, err := n.db.GetNodeTwinIDsAfter(ctx, latestCheckedID)
			if err != nil {
				log.Error().Err(err).Msgf("failed to get node twin ids after %d", latestCheckedID)
				continue
			}
			if len(newIDs) == 0 {
				continue
			}
			nodeTwinIDs := make([]uint32, 0)
			for _, id := range newIDs {
				nodeTwinIDs = append(nodeTwinIDs, uint32(id))
			}

			n.newNodeTwinIDChan <- nodeTwinIDs
			latestCheckedID = int64(nodeTwinIDs[0])
		case <-ctx.Done():
			return
		}
	}
}

func (n *NodeGPUIndexer) relayCallback(ctx context.Context, p peer.Peer, response *rmbTypes.Envelope, callBackErr error) {
	output, err := peer.Json(response, callBackErr)
	if err != nil {
		log.Error().Err(err)
		return
	}

	var nodesGPU []types.NodeGPU
	err = json.Unmarshal(output, &nodesGPU)
	if err != nil {
		log.Error().Err(err).RawJSON("data", output).Msg("failed to unmarshal GPU information response")
		return

	}
	for i := range nodesGPU {
		nodesGPU[i].NodeTwinID = response.Source.Twin
	}
	if len(nodesGPU) != 0 {
		n.nodesGPUResultsChan <- nodesGPU
	}
}
