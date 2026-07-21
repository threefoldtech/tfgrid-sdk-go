package indexer

import (
	"context"
	"reflect"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos_sdk_go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/zos_sdk_go/rmb-sdk-go/peer"
)

const (
	indexerCallTimeout     = 30 * time.Second // rmb calls timeout
	flushingBufferInterval = 60 * time.Second // upsert buffer in db if it didn't reach the batch size
	newNodesCheckInterval  = 5 * time.Minute
	batchSize              = 20

	retryInitialBackoff = 30 * time.Second
	retryMaxBackoff     = 5 * time.Minute
	retryTimeout        = 20 * time.Minute // how long a new node keeps being retried
)

// task is a twin id to call. Only tasks from the "new" finder are retryable: a freshly
// registered node is usually not answerable over rmb yet and has no other chance to be
// indexed before the next full sweep, while nodes found by the "up"/"healthy" finders are
// already re-queried every time those finders tick.
type task struct {
	id        uint32
	retryable bool
}

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
	idChan     chan task
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
		idChan:     make(chan task),
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
		case t := <-i.idChan:
			// retryable tasks sleep between attempts, so they run off the worker pool.
			// some indexers run with a single worker and would otherwise stall completely.
			if t.retryable {
				go i.handle(ctx, t)
				continue
			}

			i.handle(ctx, t)
		case <-ctx.Done():
			return
		}
	}
}

func (i *Indexer[T]) handle(ctx context.Context, t task) {
	res, err := i.call(ctx, t)
	if err != nil {
		log.Debug().Err(err).Str("indexer", i.name).Uint32("twinId", t.id).Msg("failed to call")
		return
	}

	for _, item := range res {
		log.Debug().Str("indexer", i.name).Uint32("twinId", t.id).Msgf("response: %+v", item)
		i.resultChan <- item
	}
}

// call queries the node, retrying with exponential backoff for up to retryTimeout. Only new
// nodes are retried: they are commonly not answerable over rmb right after registration, and
// would otherwise wait for the next full sweep, which is a whole day for some indexers.
func (i *Indexer[T]) call(ctx context.Context, t task) ([]T, error) {
	res, err := i.work.Get(ctx, i.rmbClient, t.id)
	if err == nil || !t.retryable {
		return res, err
	}

	deadline := time.Now().Add(retryTimeout)
	backoff := retryInitialBackoff
	for time.Now().Add(backoff).Before(deadline) {
		log.Debug().Err(err).Str("indexer", i.name).Uint32("twinId", t.id).Dur("backoff", backoff).Msg("retrying new node")

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		res, err = i.work.Get(ctx, i.rmbClient, t.id)
		if err == nil {
			return res, nil
		}

		if backoff *= 2; backoff > retryMaxBackoff {
			backoff = retryMaxBackoff
		}
	}

	log.Warn().Err(err).Str("indexer", i.name).Uint32("twinId", t.id).Msg("giving up on new node")
	return nil, err
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
