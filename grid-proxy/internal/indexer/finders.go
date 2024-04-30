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
		"new":     newNodesFinder,
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

func newNodesFinder(ctx context.Context, interval time.Duration, db db.Database, idsChan chan uint32) {
	ticker := time.NewTicker(interval)
	latestCheckedID, err := db.GetLastNodeTwinID(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get last node twin id")
	}

	for {
		select {
		case <-ticker.C:
			newIDs, err := db.GetNodeTwinIDsAfter(ctx, latestCheckedID)
			if err != nil {
				log.Error().Err(err).Msgf("failed to get node twin ids after %d", latestCheckedID)
				continue
			}
			if len(newIDs) == 0 {
				continue
			}

			latestCheckedID = newIDs[0]
			for _, id := range newIDs {
				idsChan <- id
			}
		case <-ctx.Done():
			return
		}
	}
}
