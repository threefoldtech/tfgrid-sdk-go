package indexer

import (
	"context"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

const (
	statsCall = "zos.statistics.get"
)

type WorkloadWork struct {
	findersInterval map[string]time.Duration
}

func NewWorkloadWork(interval uint) *WorkloadWork {
	return &WorkloadWork{
		findersInterval: map[string]time.Duration{
			"up": time.Duration(interval) * time.Minute,
		},
	}
}

func (w *WorkloadWork) Finders() map[string]time.Duration {
	return w.findersInterval
}

func (w *WorkloadWork) Get(ctx context.Context, rmb *peer.RpcClient, twinId uint32) ([]types.NodesWorkloads, error) {
	var response struct {
		Users struct {
			Workloads uint32 `json:"workloads"`
		} `json:"users"`
	}

	if err := callNode(ctx, rmb, statsCall, nil, twinId, &response); err != nil {
		return []types.NodesWorkloads{}, err
	}

	return []types.NodesWorkloads{
		{
			NodeTwinId:      twinId,
			WorkloadsNumber: response.Users.Workloads,
		},
	}, nil
}

func (w *WorkloadWork) Upsert(ctx context.Context, db db.Database, batch []types.NodesWorkloads) error {
	return db.UpsertNodeWorkloads(ctx, batch)
}
