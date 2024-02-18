package indexer

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	indexerCallTimeout     = 30 * time.Second // rmb calls timeout
	flushingBufferInterval = 60 * time.Second // upsert buffer in db if it didn't reach the batch size
	newNodesCheckInterval  = 5 * time.Minute
)

type Indexer interface {
	Start(ctx context.Context)
	StartNodeFinder(ctx context.Context)
	StartNodeCaller(ctx context.Context)
	StartResultBatcher(ctx context.Context)
	StartBatchUpserter(ctx context.Context)
}

type Manager struct {
	Indexers map[string]Indexer
	Context  context.Context
}

func NewManager(
	ctx context.Context,
) *Manager {
	return &Manager{
		Indexers: make(map[string]Indexer),
		Context:  ctx,
	}
}

func (m *Manager) Register(name string, indexer Indexer) {
	m.Indexers[name] = indexer
}

func (m *Manager) Start() {
	log.Info().Msg("Starting indexers manager...")
	for name, watcher := range m.Indexers {
		watcher.Start(m.Context)
		log.Info().Msgf("%s indexer started", name)
	}
}
