package internal

import (
	"context"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	zos "github.com/threefoldtech/zos/client"
	"github.com/threefoldtech/zos/pkg"
)

// RMB is an rmb abstract client interface.
type RMB interface {
	Statistics(ctx context.Context, nodeTwin uint32) (stats zos.Counters, err error)
	GetStoragePools(ctx context.Context, nodeTwin uint32) (pools []pkg.PoolMetrics, err error)
	ListGPUs(ctx context.Context, nodeTwin uint32) (gpus []zos.GPU, err error)
}

type RMBNodeClient struct {
	client     rmb.Client
	rmbTimeout time.Duration
}

func NewRmbNodeClient(rmb rmb.Client) *RMBNodeClient {
	return &RMBNodeClient{
		client:     rmb,
		rmbTimeout: timeoutRMBResponse,
	}
}

// Statistics returns some node statistics. Including total and available cpu, memory, storage, etc...
func (r *RMBNodeClient) Statistics(ctx context.Context, nodeTwin uint32) (stats zos.Counters, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.rmbTimeout)
	defer cancel()

	const cmd = "zos.statistics.get"
	return stats, r.client.Call(ctx, nodeTwin, cmd, nil, &stats)
}

// GetStoragePools executes zos system version cmd
func (r *RMBNodeClient) GetStoragePools(ctx context.Context, nodeTwin uint32) (pools []pkg.PoolMetrics, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.rmbTimeout)
	defer cancel()

	const cmd = "zos.storage.pools"
	return pools, r.client.Call(ctx, nodeTwin, cmd, nil, &pools)
}

// ListGPUs return a list of all gpus on the node.
func (r *RMBNodeClient) ListGPUs(ctx context.Context, nodeTwin uint32) (gpus []zos.GPU, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.rmbTimeout)
	defer cancel()

	const cmd = "zos.gpu.list"
	return gpus, r.client.Call(ctx, nodeTwin, cmd, nil, &gpus)
}
