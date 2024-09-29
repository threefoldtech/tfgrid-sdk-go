package indexer

import (
	"context"
	"strings"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

const (
	featuresCallCmd = "zos.system.node_features_get"
)

type FeatureWork struct {
	findersInterval map[string]time.Duration
}

func NewFeatureWork(interval uint) *FeatureWork {
	return &FeatureWork{
		findersInterval: map[string]time.Duration{
			"up":  time.Duration(interval) * time.Minute,
			"new": newNodesCheckInterval,
		},
	}
}

func (w *FeatureWork) Finders() map[string]time.Duration {
	return w.findersInterval
}

func (w *FeatureWork) Get(ctx context.Context, rmb *peer.RpcClient, twinId uint32) ([]types.NodeFeatures, error) {
	var features []string
	err := callNode(ctx, rmb, featuresCallCmd, nil, twinId, &features)
	if err != nil {
		return []types.NodeFeatures{}, err
	}

	res := parseNodeFeatures(twinId, features)
	return []types.NodeFeatures{res}, nil

}

func (w *FeatureWork) Upsert(ctx context.Context, db db.Database, batch []types.NodeFeatures) error {
	return db.UpsertNodeFeatures(ctx, batch)
}

func parseNodeFeatures(twinId uint32, features []string) types.NodeFeatures {
	res := types.NodeFeatures{
		NodeTwinId: twinId,
		UpdatedAt:  time.Now().Unix(),
		Light:      false,
	}

	for _, feat := range features {
		if strings.Contains(feat, "light") {
			res.Light = true
			return res
		}

	}

	return res
}
