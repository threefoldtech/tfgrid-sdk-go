package indexer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

type CpuBenchmarkWork struct {
	findersInterval map[string]time.Duration
}

type CpuBenchmarkResult struct {
	SingleThreaded float64 `json:"single"`
	MultiThreaded  float64 `json:"multi"`
	Threads        int     `json:"threads"`
	Workloads      int     `json:"workloads"`
}

func NewCpuBenchmarkWork(interval uint) *CpuBenchmarkWork {
	return &CpuBenchmarkWork{
		findersInterval: map[string]time.Duration{
			"up": time.Duration(interval) * time.Minute,
		},
	}
}

func (w *CpuBenchmarkWork) Finders() map[string]time.Duration {
	return w.findersInterval
}

func (w *CpuBenchmarkWork) Get(ctx context.Context, rmb *peer.RpcClient, twinId uint32) ([]types.CpuBenchmark, error) {
	payload := struct {
		Name string
	}{
		Name: "cpu-benchmark",
	}
	var response TaskResult
	if err := callNode(ctx, rmb, perfTestCallCmd, payload, twinId, &response); err != nil {
		return []types.CpuBenchmark{}, err
	}

	cpuBenchmarkReport, err := parseCpuBenchmark(response, twinId)
	if err != nil {
		return []types.CpuBenchmark{}, err
	}

	return []types.CpuBenchmark{cpuBenchmarkReport}, nil
}

func (w *CpuBenchmarkWork) Upsert(ctx context.Context, db db.Database, batch []types.CpuBenchmark) error {
	return db.UpsertCpuBenchmark(ctx, batch)
}

func parseCpuBenchmark(res TaskResult, twinId uint32) (types.CpuBenchmark, error) {
	cpuBenchmark := types.CpuBenchmark{
		NodeTwinId: twinId,
	}

	iperfResultBytes, err := json.Marshal(res.Result)
	if err != nil {
		return cpuBenchmark, err
	}

	var cpuBenchmarkResults CpuBenchmarkResult
	if err := json.Unmarshal(iperfResultBytes, &cpuBenchmarkResults); err != nil {
		return cpuBenchmark, err
	}

	cpuBenchmark.SingleThreaded = cpuBenchmarkResults.SingleThreaded
	cpuBenchmark.MultiThreaded = cpuBenchmarkResults.MultiThreaded
	cpuBenchmark.Threads = cpuBenchmarkResults.Threads
	cpuBenchmark.Workloads = cpuBenchmarkResults.Workloads
	cpuBenchmark.UpdatedAt = time.Now().Unix()

	return cpuBenchmark, nil
}
