package gpuindexer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/direct"
	rmbTypes "github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/direct/types"
)

type NodeGPUIndexer struct {
	db                    db.Database
	relayClient           *direct.Client
	nodesGPUChan          chan []types.NodeGPU
	checkInterval         time.Duration
	workerCleanupInterval time.Duration
	workerBatchSize       int
}

func NewNodeGPUIndexer(
	ctx context.Context,
	relayURL,
	mnemonics string,
	sub *substrate.Substrate,
	db db.Database,
	indexerCheckIntervalMins int,
	workerBatchSize int) (*NodeGPUIndexer, error) {
	indexer := &NodeGPUIndexer{
		db:                    db,
		nodesGPUChan:          make(chan []types.NodeGPU),
		checkInterval:         time.Duration(indexerCheckIntervalMins) * time.Minute,
		workerCleanupInterval: time.Duration(indexerCheckIntervalMins/4) * time.Minute,
		workerBatchSize:       workerBatchSize,
	}

	client, err := direct.NewClient(ctx, direct.KeyTypeSr25519, mnemonics, relayURL, "tfgrid_proxy_indexer", sub, true, indexer.relayCallback)
	if err != nil {
		return nil, fmt.Errorf("failed to create direct RMB client: %w", err)
	}
	indexer.relayClient = client

	return indexer, nil
}

func (n *NodeGPUIndexer) worker(ctx context.Context) {
	var nodesGPUBuffer []types.NodeGPU
	ticker := time.NewTicker(n.workerCleanupInterval)
	for {
		select {
		case nodesGPU := <-n.nodesGPUChan:
			nodesGPUBuffer = append(nodesGPUBuffer, nodesGPU...)
			if len(nodesGPUBuffer) >= n.workerBatchSize {
				err := n.db.UpsertNodesGPU(nodesGPUBuffer)
				if err != nil {
					log.Error().Err(err).Msg("failed to update GPU info in GPU indexer")
					continue
				}
				// Push the periodic check for leftovers data backwards
				ticker.Reset(n.workerCleanupInterval)
				nodesGPUBuffer = nil
			}
		// This case should only be triggered when there leftovers data in the buffer, it stores them and exit
		// It runs periodically according to the indexer interval, needs to be delayed as much as possible
		// as to not intervene with the batch logic above
		case <-ticker.C:
			if len(nodesGPUBuffer) != 0 {
				err := n.db.UpsertNodesGPU(nodesGPUBuffer)
				if err != nil {
					log.Error().Err(err).Msg("failed to update GPU info in GPU indexer")
					continue
				}
			}
			ticker.Stop()
			return
		case <-ctx.Done():
			log.Error().Err(ctx.Err()).Msg("worker exited")
			return

		}
	}
}

func (n *NodeGPUIndexer) relayCallback(response *rmbTypes.Envelope, callBackErr error) {
	errResp := response.GetError()

	if errResp != nil {
		log.Error().Msg(errResp.Message)
		return
	}

	resp := response.GetResponse()
	if resp == nil {
		log.Error().Msg("received a non response envelope")
		return
	}

	if response.Schema == nil || *response.Schema != rmb.DefaultSchema {
		log.Error().Msgf("invalid schema received expected '%s'", rmb.DefaultSchema)
		return
	}

	output := response.Payload.(*rmbTypes.Envelope_Plain).Plain

	var nodesGPU []types.NodeGPU
	err := json.Unmarshal(output, &nodesGPU)
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal GPU information response")
		return

	}
	for i := range nodesGPU {
		nodesGPU[i].NodeTwinID = response.Source.Twin
	}
	// Will be using only one worker giving the current amount of nodes supporting GPUs
	if len(nodesGPU) != 0 {
		n.nodesGPUChan <- nodesGPU
	}

}

func (n *NodeGPUIndexer) Start(ctx context.Context) {
	ticker := time.NewTicker(n.checkInterval)
	n.run(ctx)
	for {
		select {
		case <-ticker.C:
			n.run(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (n *NodeGPUIndexer) run(ctx context.Context) {
	log.Info().Msg("GPU indexer started")
	status := "up"
	filter := types.NodeFilter{
		Status: &status,
	}

	limit := types.Limit{
		Size:     100,
		RetCount: true,
		Page:     1,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		n.worker(ctx)
	}()

	hasNext := true
	for hasNext {
		nodes, _, err := n.db.GetNodes(filter, limit)
		if err != nil {
			log.Error().Err(err).Msg("unable to query nodes in GPU indexer")
			return
		}
		if len(nodes) < int(limit.Size) {
			hasNext = false
		}

		for _, node := range nodes {
			id := uuid.NewString()
			err = n.relayClient.Call(ctx, id, uint32(node.TwinID), "zos.gpu.list", nil)
			if err != nil {
				log.Error().Err(err).Msgf("failed to send get GPU info request from relay in GPU indexer for node %d", node.NodeID)
			}
		}

		limit.Page++
	}
	wg.Wait()

	log.Info().Msg("GPU indexer finished")
}
