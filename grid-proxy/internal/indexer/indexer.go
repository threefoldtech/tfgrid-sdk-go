package indexer

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

type Watcher interface {
	Start(ctx context.Context)
}

type Indexer struct {
	Watchers  map[string]Watcher
	Paused    bool
	Context   context.Context
	RmbClient *peer.RpcClient
}

func NewIndexer(
	ctx context.Context,
	paused bool,
	rmbClient *peer.RpcClient,
) *Indexer {
	return &Indexer{
		Watchers:  make(map[string]Watcher),
		Paused:    paused,
		Context:   ctx,
		RmbClient: rmbClient,
	}
}

func (i *Indexer) RegisterWatcher(name string, watcher Watcher) {
	i.Watchers[name] = watcher
}

func (i *Indexer) Start() {
	if i.Paused {
		log.Info().Msg("Indexer paused")
		return
	}

	log.Info().Msg("Starting indexer...")
	for name, watcher := range i.Watchers {
		watcher.Start(i.Context)
		log.Info().Msgf("%s watcher started", name)
	}
}
