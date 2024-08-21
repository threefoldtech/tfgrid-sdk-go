package indexer

import (
	"context"
	"reflect"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

const (
	indexerCallTimeout     = 30 * time.Second // rmb calls timeout
	flushingBufferInterval = 60 * time.Second // upsert buffer in db if it didn't reach the batch size
	newNodesCheckInterval  = 5 * time.Minute
	batchSize              = 20
)

type Work[T any] interface {
	Finders() map[string]time.Duration
	Get(ctx context.Context, rmb *peer.RpcClient, id uint32) ([]T, error)
	Upsert(ctx context.Context, db db.Database, batch []T) error
}

type Indexer[T any] struct {
	name       string
	work       Work[T]
	dbClient   db.Database
	rmbClient  *peer.RpcClient
	idChan     chan uint32
	resultChan chan T
	batchChan  chan []T
	workerNum  uint
}

func NewIndexer[T any](
	work Work[T],
	name string,
	db db.Database,
	rmb *peer.RpcClient,
	worker uint,
) *Indexer[T] {
	return &Indexer[T]{
		work:       work,
		name:       name,
		dbClient:   db,
		rmbClient:  rmb,
		workerNum:  worker,
		idChan:     make(chan uint32),
		resultChan: make(chan T),
		batchChan:  make(chan []T),
	}
}

func (i *Indexer[T]) Start(ctx context.Context) {
	for name, interval := range i.work.Finders() {
		go finders[name](ctx, interval, i.dbClient, i.idChan)
	}

	for j := uint(0); j < i.workerNum; j++ {
		go i.get(ctx)
	}

	go i.batch(ctx)

	go i.upsert(ctx)

	log.Info().Msgf("%s Indexer started", i.name)
}

func (i *Indexer[T]) get(ctx context.Context) {
	for {
		select {
		case id := <-i.idChan:
			res, err := i.work.Get(ctx, i.rmbClient, id)
			if err != nil {
				log.Debug().Err(err).Str("indexer", i.name).Uint32("twinId", id).Msg("failed to call")
				continue
			}

			for _, item := range res {
				log.Debug().Str("indexer", i.name).Uint32("twinId", id).Msgf("response: %+v", item)
				i.resultChan <- item
			}
		case <-ctx.Done():
			return
		}
	}
}

func (i *Indexer[T]) batch(ctx context.Context) {
	buffer := make([]T, 0, batchSize)

	ticker := time.NewTicker(flushingBufferInterval)
	for {
		select {
		case data := <-i.resultChan:
			// to prevent having multiple data for the same twin from different finders
			if i.isUnique(buffer, data) {
				buffer = append(buffer, data)
			}
			if len(buffer) >= int(batchSize) {
				log.Debug().Str("indexer", i.name).Int("size", len(buffer)).Msg("batching")
				i.batchChan <- buffer
				buffer = nil
			}
		case <-ticker.C:
			if len(buffer) != 0 {
				log.Debug().Str("indexer", i.name).Int("size", len(buffer)).Msg("batching")
				i.batchChan <- buffer
				buffer = nil
			}
		case <-ctx.Done():
			return
		}
	}
}

func (i *Indexer[T]) upsert(ctx context.Context) {
	for {
		select {
		case batch := <-i.batchChan:
			err := i.work.Upsert(ctx, i.dbClient, batch)
			if err != nil {
				log.Error().Err(err).Str("indexer", i.name).Msg("failed to upsert batch")
			}
		case <-ctx.Done():
			return
		}
	}
}

func (i *Indexer[T]) isUnique(buffer []T, data T) bool {
	for _, item := range buffer {
		if reflect.DeepEqual(item, data) {
			return false
		}
	}
	return true
}
