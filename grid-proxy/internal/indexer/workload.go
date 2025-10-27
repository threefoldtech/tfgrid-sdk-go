package indexer

import (
	"context"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
	"github.com/threefoldtech/zosbase/pkg/gridtypes"
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

type NodeStatisticsResult struct {
	NodeTwinID     uint32
	Workload       types.NodesWorkloads
	SystemCapacity types.Capacity
}

func (w *WorkloadWork) Get(ctx context.Context, rmb *peer.RpcClient, twinId uint32) ([]NodeStatisticsResult, error) {
	var response types.NodeStatistics

	if err := callNode(ctx, rmb, statsCall, nil, twinId, &response); err != nil {
		return []NodeStatisticsResult{}, err
	}

	now := time.Now().Unix()

	return []NodeStatisticsResult{
		{
			NodeTwinID: twinId,
			Workload: types.NodesWorkloads{
				NodeTwinId:      twinId,
				WorkloadsNumber: uint32(response.Users.Workloads),
				UpdatedAt:       now,
			},
			SystemCapacity: types.Capacity{
				CRU: uint64(response.System.CRU),
				HRU: gridtypes.Unit(response.System.HRU),
				MRU: gridtypes.Unit(response.System.MRU),
				SRU: gridtypes.Unit(response.System.SRU),
			},
		},
	}, nil
}

func (w *WorkloadWork) Upsert(ctx context.Context, db db.Database, batch []NodeStatisticsResult) error {
	// Extract workloads and system overhead usage into separate slices
	workloads := make([]types.NodesWorkloads, len(batch))
	systemOverheads := make([]types.NodeSystemUsage, len(batch))

	for i, data := range batch {
		workloads[i] = data.Workload
		systemOverheads[i] = types.NodeSystemUsage{
			NodeTwinID: data.NodeTwinID,
			SystemCRU:  data.SystemCapacity.CRU + (data.SystemCapacity.CRU / 10),
			SystemHRU:  uint64(data.SystemCapacity.HRU) + uint64(data.SystemCapacity.HRU)/10,
			SystemMRU:  uint64(data.SystemCapacity.MRU) + uint64(data.SystemCapacity.MRU)/10,
			SystemSRU:  uint64(data.SystemCapacity.SRU) + uint64(data.SystemCapacity.SRU)/10,
			UpdatedAt:  data.Workload.UpdatedAt,
		}
	}

	// Upsert both workloads and system overhead usage
	if err := db.UpsertNodeWorkloads(ctx, workloads); err != nil {
		return err
	}

	return db.UpsertNodeSystemResources(ctx, systemOverheads)
}
