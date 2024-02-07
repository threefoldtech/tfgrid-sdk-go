package indexer

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

const (
	perfTestCallCmd = "zos.perf.get"
	testName        = "iperf"
)

type SpeedWatcher struct {
	database        db.Database
	rmbClient       *peer.RpcClient
	nodeTwinIdsChan chan uint32
	resultChan      chan types.NetworkTestResult
	interval        time.Duration
	workers         uint
	batchSize       uint
}

func NewSpeedWatcher(
	ctx context.Context,
	database db.Database,
	rmbClient *peer.RpcClient,
	interval uint,
	workers uint,
	batchSize uint,
) *SpeedWatcher {
	return &SpeedWatcher{
		database:        database,
		rmbClient:       rmbClient,
		nodeTwinIdsChan: make(chan uint32),
		resultChan:      make(chan types.NetworkTestResult),
		interval:        time.Duration(interval) * time.Minute,
		workers:         workers,
		batchSize:       batchSize,
	}
}

func (w *SpeedWatcher) Start(ctx context.Context) {
	go w.startNodeQuerier(ctx)

	for i := uint(0); i < w.workers; i++ {
		go w.startNodeCaller(ctx)
	}

	go w.startUpserter(ctx, w.database)
}

// TODO: not only on interval but also on any node goes from down>up or newly added nodes
func (w *SpeedWatcher) startNodeQuerier(ctx context.Context) {
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

func (w *SpeedWatcher) startNodeCaller(ctx context.Context) {
	for {
		select {
		case twinId := <-w.nodeTwinIdsChan:
			response, err := w.callNode(ctx, twinId)
			if err != nil {
				continue
			}
			parsed := parse(response, twinId)
			log.Info().Msgf("got: %+v", parsed)
			w.resultChan <- parsed
		case <-ctx.Done():
			return
		}
	}
}

// TODO: make it generic and then assert the result in each watcher
func (w *SpeedWatcher) callNode(ctx context.Context, twinId uint32) (types.PerfResult, error) {
	var result types.PerfResult
	subCtx, cancel := context.WithTimeout(ctx, indexerCallTimeout)
	defer cancel()

	payload := struct {
		Name string
	}{
		Name: testName,
	}
	err := w.rmbClient.Call(subCtx, twinId, perfTestCallCmd, payload, &result)
	if err != nil {
		log.Error().Err(err).Uint32("twinId", twinId).Msg("failed to call node")
	}

	return result, err
}

func (w *SpeedWatcher) startUpserter(ctx context.Context, database db.Database) {
	buffer := make([]types.NetworkTestResult, w.batchSize)

	ticker := time.NewTicker(flushingInterval)
	for {
		select {
		case report := <-w.resultChan:
			buffer = append(buffer, report)
			if len(buffer) >= int(w.batchSize) {
				err := w.database.UpsertNetworkSpeed(ctx, buffer)
				if err != nil {
					log.Error().Err(err)
				}
				buffer = nil
			}
		case <-ticker.C:
			if len(buffer) != 0 {
				err := w.database.UpsertNetworkSpeed(ctx, buffer)
				if err != nil {
					log.Error().Err(err)
				}
				buffer = nil
			}
		case <-ctx.Done():
			return
		}
	}
}

func parse(res types.PerfResult, twinId uint32) types.NetworkTestResult {
	// TODO: better parsing
	// we have four speeds tcp/udp for ipv4/ipv6.
	// now, we just pick the first non-zero
	for _, report := range res.Result {
		if report.DownloadSpeed != 0 {
			report.NodeTwinId = twinId
			return report
		}
	}
	return types.NetworkTestResult{}
}
