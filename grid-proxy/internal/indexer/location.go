package indexer

import (
	"context"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
	"github.com/threefoldtech/zosbase/pkg/geoip"
)

const locationCmd = "zos.location.get"

var _ Work[types.NodeLocation] = (*LocationWork)(nil)

type LocationWork struct {
	finders map[string]time.Duration
}

func NewLocationWork(interval uint) *LocationWork {
	return &LocationWork{
		finders: map[string]time.Duration{
			"up": time.Duration(interval) * time.Minute,
		},
	}
}

func (w *LocationWork) Finders() map[string]time.Duration {
	return w.finders
}

func (w *LocationWork) Get(ctx context.Context, rmb *peer.RpcClient, id uint32) ([]types.NodeLocation, error) {
	var loc geoip.Location
	if err := callNode(ctx, rmb, locationCmd, nil, id, &loc); err != nil {
		return []types.NodeLocation{}, nil
	}

	return []types.NodeLocation{
		{
			Country:   loc.Country,
			Continent: loc.Continent,
			UpdatedAt: time.Now().Unix(),
		},
	}, nil
}

func (w *LocationWork) Upsert(ctx context.Context, db db.Database, batch []types.NodeLocation) error {
	unique := removeDuplicates(batch, func(n types.NodeLocation) string {
		return n.Country
	})

	return db.UpsertNodeLocation(ctx, unique)
}
