package indexer

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
)

var (
	finders = map[string]Finder{
		"up":      upNodesFinder,
		"healthy": healthyNodesFinder,
	}
)

type Finder func(context.Context, time.Duration, db.Database, chan uint32)

func upNodesFinder(ctx context.Context, interval time.Duration, db db.Database, idsChan chan uint32) {
	ticker := time.NewTicker(interval)

	queryUpNodes(ctx, db, idsChan)
	for {
		select {
		case <-ticker.C:
			queryUpNodes(ctx, db, idsChan)
		case <-ctx.Done():
			return
		}
	}
}

func healthyNodesFinder(ctx context.Context, interval time.Duration, db db.Database, idsChan chan uint32) {
	ticker := time.NewTicker(interval)

	queryHealthyNodes(ctx, db, idsChan)
	for {
		select {
		case <-ticker.C:
			queryHealthyNodes(ctx, db, idsChan)
		case <-ctx.Done():
			return
		}
	}
}

// newUnindexedNodesFinder finds nodes that exist in node table but not in the indexed table
func newUnindexedNodesFinder(ctx context.Context, interval time.Duration, db db.Database, idsChan chan uint32, indexedTable string) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	queryUnindexedNodes := func() {
		unindexedIDs, err := db.GetUnindexedNodeTwinIDs(ctx, indexedTable)
		if err != nil {
			log.Error().Err(err).Str("table", indexedTable).Msg("failed to get unindexed nodes")
			return
		}

		if len(unindexedIDs) == 0 {
			return
		}

		log.Info().Int("count", len(unindexedIDs)).Str("table", indexedTable).Msg("found unindexed nodes")

		for _, id := range unindexedIDs {
			idsChan <- id
		}
	}

	queryUnindexedNodes()

	for {
		select {
		case <-ticker.C:
			queryUnindexedNodes()
		case <-ctx.Done():
			return
		}
	}
}
