package indexer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"

	zosPerfPkg "github.com/threefoldtech/zos/pkg/perf"
	zosIPerfPkg "github.com/threefoldtech/zos/pkg/perf/iperf"
)

const (
	perfTestCallCmd = "zos.perf.get"
	testName        = "iperf"
)

type SpeedIndexer struct {
	database        db.Database
	rmbClient       *peer.RpcClient
	interval        time.Duration
	workers         uint
	batchSize       uint
	nodeTwinIdsChan chan uint32
	resultChan      chan types.Speed
	batchChan       chan []types.Speed
}

func NewSpeedIndexer(
	rmbClient *peer.RpcClient,
	database db.Database,
	batchSize uint,
	interval uint,
	workers uint,
) *SpeedIndexer {
	return &SpeedIndexer{
		database:        database,
		rmbClient:       rmbClient,
		batchSize:       batchSize,
		interval:        time.Duration(interval) * time.Minute,
		workers:         workers,
		nodeTwinIdsChan: make(chan uint32),
		resultChan:      make(chan types.Speed),
		batchChan:       make(chan []types.Speed),
	}
}

func (w *SpeedIndexer) Start(ctx context.Context) {
	go w.StartNodeFinder(ctx)

	for i := uint(0); i < w.workers; i++ {
		go w.StartNodeCaller(ctx)
	}

	for i := uint(0); i < w.workers; i++ {
		go w.StartResultBatcher(ctx)
	}

	go w.StartBatchUpserter(ctx)
}

func (w *SpeedIndexer) StartNodeFinder(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	queryUpNodes(ctx, w.database, w.nodeTwinIdsChan)
	for {
		select {
		case <-ticker.C:
			queryUpNodes(ctx, w.database, w.nodeTwinIdsChan)
		case <-ctx.Done():
			return
		}
	}
}

func (w *SpeedIndexer) StartNodeCaller(ctx context.Context) {
	for {
		select {
		case twinId := <-w.nodeTwinIdsChan:
			payload := struct {
				Name string
			}{
				Name: testName,
			}
			var response zosPerfPkg.TaskResult
			if err := callNode(ctx, w.rmbClient, perfTestCallCmd, payload, twinId, &response); err != nil {
				continue
			}

			speedReport, err := parseSpeed(response, twinId)
			if err != nil {
				continue
			}

			w.resultChan <- speedReport
		case <-ctx.Done():
			return
		}
	}
}

func (w *SpeedIndexer) StartResultBatcher(ctx context.Context) {
	buffer := make([]types.Speed, 0, w.batchSize)

	ticker := time.NewTicker(flushingBufferInterval)
	for {
		select {
		case report := <-w.resultChan:
			buffer = append(buffer, report)
			if len(buffer) >= int(w.batchSize) {
				w.batchChan <- buffer
				buffer = nil
			}
		case <-ticker.C:
			if len(buffer) != 0 {
				w.batchChan <- buffer
				buffer = nil
			}
		case <-ctx.Done():
			return
		}
	}
}

func (w *SpeedIndexer) StartBatchUpserter(ctx context.Context) {
	for {
		select {
		case batch := <-w.batchChan:
			err := w.database.UpsertNetworkSpeed(ctx, batch)
			if err != nil {
				log.Error().Err(err).Msg("failed to upsert network speed")
			}
		case <-ctx.Done():
			return
		}
	}
}

func parseSpeed(res zosPerfPkg.TaskResult, twinId uint32) (types.Speed, error) {
	speed := types.Speed{
		NodeTwinId: twinId,
	}

	iperfResultBytes, err := json.Marshal(res.Result)
	if err != nil {
		return speed, err
	}

	var iperfResults []zosIPerfPkg.IperfResult
	if err := json.Unmarshal(iperfResultBytes, &iperfResults); err != nil {
		return speed, err
	}

	// TODO: better parsing
	// we have four speeds tcp/udp for ipv4/ipv6.
	// now, we just pick the first non-zero
	for _, report := range iperfResults {
		if report.DownloadSpeed != 0 {
			speed.Download = report.DownloadSpeed
			speed.Upload = report.UploadSpeed
			return speed, nil
		}
	}

	return speed, nil
}
