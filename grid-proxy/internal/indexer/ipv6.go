package indexer

import (
	"context"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

const cmd = "zos.network.has_ipv6"

var _ Work[types.HasIpv6] = (*Ipv6Work)(nil)

type Ipv6Work struct {
	finders map[string]time.Duration
}

func NewIpv6Work(interval uint) *Ipv6Work {
	return &Ipv6Work{
		finders: map[string]time.Duration{
			"up": time.Duration(interval) * time.Minute,
		},
	}
}

func (w *Ipv6Work) Finders() map[string]time.Duration {
	return w.finders
}

func (w *Ipv6Work) Get(ctx context.Context, rmb *peer.RpcClient, id uint32) ([]types.HasIpv6, error) {
	var has_ipv6 bool
	if err := callNode(ctx, rmb, cmd, nil, id, &has_ipv6); err != nil {
		return []types.HasIpv6{}, nil
	}

	return []types.HasIpv6{
		{
			NodeTwinId: id,
			HasIpv6:    has_ipv6,
			UpdatedAt:  time.Now().Unix(),
		},
	}, nil
}

func (w *Ipv6Work) Upsert(ctx context.Context, db db.Database, batch []types.HasIpv6) error {
	return db.UpsertNodeIpv6Report(ctx, batch)
}
