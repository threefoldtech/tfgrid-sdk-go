package indexer

import (
	"context"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

const (
	healthCallCmd = "zos.system.version"
)

type HealthWork struct {
	findersInterval map[string]time.Duration
}

func NewHealthWork(interval uint) *HealthWork {
	return &HealthWork{
		findersInterval: map[string]time.Duration{
			"up":      time.Duration(interval) * time.Minute,
			"healthy": time.Duration(interval) * time.Minute,
		},
	}
}

func (w *HealthWork) Finders() map[string]time.Duration {
	return w.findersInterval
}

func (w *HealthWork) Get(ctx context.Context, rmb *peer.RpcClient, twinId uint32) ([]types.HealthReport, error) {
	var response types.HealthReport
	err := callNode(ctx, rmb, healthCallCmd, nil, twinId, &response)

	res := getHealthReport(response, err, twinId)
	return []types.HealthReport{res}, nil
}

func (w *HealthWork) Upsert(ctx context.Context, db db.Database, batch []types.HealthReport) error {
	return db.UpsertNodeHealth(ctx, batch)
}

// TODO: use diagnostics call instead
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
