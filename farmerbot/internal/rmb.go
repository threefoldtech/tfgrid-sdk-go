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
	rmb        rmb.Client
	rmbTimeout time.Duration
}

func NewRmbNodeClient(rmb rmb.Client) *RMBNodeClient {
	return &RMBNodeClient{
		rmb:        rmb,
		rmbTimeout: timeoutRMBResponse,
	}
}

// Statistics returns some node statistics. Including total and available cpu, memory, storage, etc...
func (n *RMBNodeClient) Statistics(ctx context.Context, nodeTwin uint32) (stats zos.Counters, err error) {
	ctx, cancel := context.WithTimeout(ctx, n.rmbTimeout)
	defer cancel()

	const cmd = "zos.statistics.get"
	return stats, n.rmb.Call(ctx, nodeTwin, cmd, nil, &stats)
}

// GetStoragePools executes zos system version cmd
func (n *RMBNodeClient) GetStoragePools(ctx context.Context, nodeTwin uint32) (pools []pkg.PoolMetrics, err error) {
	ctx, cancel := context.WithTimeout(ctx, n.rmbTimeout)
	defer cancel()

	const cmd = "zos.storage.pools"
	return pools, n.rmb.Call(ctx, nodeTwin, cmd, nil, &pools)
}

// ListGPUs return a list of all gpus on the node.
func (n *RMBNodeClient) ListGPUs(ctx context.Context, nodeTwin uint32) (gpus []zos.GPU, err error) {
	ctx, cancel := context.WithTimeout(ctx, n.rmbTimeout)
	defer cancel()

	const cmd = "zos.gpu.list"
	return gpus, n.rmb.Call(ctx, nodeTwin, cmd, nil, &gpus)
}
