package indexer

import (
	"context"
	"fmt"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

const (
	gpuListCmd = "zos.gpu.list"
)

type GPUWork struct {
	findersInterval map[string]time.Duration
	db              db.Database
}

func NewGPUWork(interval uint, db db.Database) *GPUWork {
	return &GPUWork{
		findersInterval: map[string]time.Duration{
			"up":  time.Duration(interval) * time.Minute,
			"new": newNodesCheckInterval,
		},
		db: db,
	}
}

func (w *GPUWork) Finders() map[string]time.Duration {
	return w.findersInterval
}

func (w *GPUWork) IndexedTable() string {
	return "node_gpu"
}

func (w *GPUWork) Get(ctx context.Context, rmb *peer.RpcClient, twinId uint32) ([]types.NodeGPU, error) {
	// in case an error returned? return directly we can leave the previously indexed cards
	// in case null returned? we need to clean all previously added cards till now
	// in case cards changed? Upsert() will take care of invalidating the old cards

	var gpus []types.NodeGPU
	if err := callNode(ctx, rmb, gpuListCmd, nil, twinId, &gpus); err != nil {
		return gpus, err
	}

	before := time.Now().Unix()
	if err := w.db.DeleteOldGpus(ctx, []uint32{twinId}, before); err != nil {
		return gpus, fmt.Errorf("failed to remove old GPUs: %w", err)
	}

	for i := 0; i < len(gpus); i++ {
		gpus[i].NodeTwinID = twinId
		gpus[i].UpdatedAt = time.Now().Unix()
	}

	return gpus, nil
}

func (w *GPUWork) Upsert(ctx context.Context, db db.Database, batch []types.NodeGPU) error {
	nodeTwinIds := []uint32{}
	for _, gpu := range batch {
		nodeTwinIds = append(nodeTwinIds, gpu.NodeTwinID)
	}

	// Invalidate old indexed GPUs for the same node, but first check the batch
	// to avoid removing GPUs inserted in the last batch within the same indexer run.
	before := time.Now().Add(-w.findersInterval["up"]).Unix()
	if err := db.DeleteOldGpus(ctx, nodeTwinIds, before); err != nil {
		return fmt.Errorf("failed to remove old GPUs: %w", err)
	}

	if err := db.UpsertNodesGPU(ctx, batch); err != nil {
		return fmt.Errorf("failed to upsert new GPUs: %w", err)
	}

	return nil
}
