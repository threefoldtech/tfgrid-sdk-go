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
}

func NewGPUWork(interval uint) *GPUWork {
	return &GPUWork{
		findersInterval: map[string]time.Duration{
			"up":  time.Duration(interval) * time.Minute,
			"new": newNodesCheckInterval,
		},
	}
}

func (w *GPUWork) Finders() map[string]time.Duration {
	return w.findersInterval
}

func (w *GPUWork) Get(ctx context.Context, rmb *peer.RpcClient, twinId uint32) ([]types.NodeGPU, error) {
	var gpus []types.NodeGPU
	err := callNode(ctx, rmb, gpuListCmd, nil, twinId, &gpus)
	if err != nil {
		return gpus, err
	}

	for i := 0; i < len(gpus); i++ {
		gpus[i].NodeTwinID = twinId
		gpus[i].UpdatedAt = time.Now().Unix()
	}

	return gpus, nil
}

func (w *GPUWork) Upsert(ctx context.Context, db db.Database, batch []types.NodeGPU) error {
	expirationInterval := w.findersInterval["up"]
	err := discardOldGpus(ctx, db, expirationInterval, batch)
	if err != nil {
		return fmt.Errorf("failed to remove old GPUs: %w", err)
	}

	err = db.UpsertNodesGPU(ctx, batch)
	if err != nil {
		return fmt.Errorf("failed to upsert new GPUs: %w", err)
	}

	return nil
}

func discardOldGpus(ctx context.Context, database db.Database, interval time.Duration, gpuBatch []types.NodeGPU) error {
	// invalidate the old indexed GPUs for the same node,
	// but check the batch first to ensure it does not contain related GPUs to node twin it from the last batch.
	nodeTwinIds := []uint32{}
	for _, gpu := range gpuBatch {
		nodeTwinIds = append(nodeTwinIds, gpu.NodeTwinID)
	}

	expiration := time.Now().Unix() - int64(interval.Seconds())
	return database.DeleteOldGpus(ctx, nodeTwinIds, expiration)
}
