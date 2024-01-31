package indexer

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

const (
	DmiCallCmd = "zos.system.dmi"
)

type DmiWatcher struct {
	database        db.Database
	rmbClient       *peer.RpcClient
	nodeTwinIdsChan chan uint32
	resultChan      chan DMI
	interval        time.Duration
	workers         uint
}

func NewDmiWatcher(
	ctx context.Context,
	database db.Database,
	rmbClient *peer.RpcClient,
	interval uint,
	workers uint,
) *DmiWatcher {
	return &DmiWatcher{
		database:        database,
		rmbClient:       rmbClient,
		nodeTwinIdsChan: make(chan uint32),
		resultChan:      make(chan DMI),
		interval:        time.Duration(interval) * time.Minute,
		workers:         workers,
	}
}

func (w *DmiWatcher) Start(ctx context.Context) {
	go w.startNodeQuerier(ctx)

	for i := uint(0); i < w.workers; i++ {
		go w.startNodeCaller(ctx)
	}

	go w.startUpserter(ctx, w.database)
}

func (w *DmiWatcher) startNodeQuerier(ctx context.Context) {
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

func (w *DmiWatcher) startNodeCaller(ctx context.Context) {
	for {
		select {
		case twinId := <-w.nodeTwinIdsChan:
			w.resultChan <- w.callNode(ctx, twinId)
		case <-ctx.Done():
			return
		}
	}
}

// TODO: make it generic and then assert the result in each watcher
func (w *DmiWatcher) callNode(ctx context.Context, twinId uint32) DMI {
	var result DMI
	subCtx, cancel := context.WithTimeout(ctx, indexerCallTimeout)
	defer cancel()

	err := w.rmbClient.Call(subCtx, twinId, DmiCallCmd, nil, &result)
	if err != nil {
		log.Error().Err(err).Uint32("twinId", twinId).Msg("failed to call node")
	}

	return result
}

func (w *DmiWatcher) startUpserter(ctx context.Context, database db.Database) {
	for {
		select {
		case dmiData := <-w.resultChan:
			log.Debug().Msgf("received: %+v", dmiData)
			// collect in batch
			// upsert in db
		case <-ctx.Done():
			return
		}
	}
}
