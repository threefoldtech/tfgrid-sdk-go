package mock

import (
	"context"
	"sort"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// Twins returns twins with the given filters and pagination parameters
func (g *GridProxyMockClient) Twins(ctx context.Context, filter types.TwinFilter, limit types.Limit) (res []types.Twin, totalCount int, err error) {
	res = []types.Twin{}

	if limit.Page == 0 {
		limit.Page = 1
	}
	if limit.Size == 0 {
		limit.Size = 50
	}
	for _, twin := range g.data.Twins {
		if twin.satisfies(filter) {
			res = append(res, types.Twin{
				TwinID:    uint(twin.TwinID),
				AccountID: twin.AccountID,
				Relay:     twin.Relay,
				PublicKey: twin.PublicKey,
			})
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].TwinID < res[j].TwinID
	})

	res, totalCount = getPage(res, limit)

	return
}

func (twin *Twin) satisfies(f types.TwinFilter) bool {
	if f.TwinID != nil && *f.TwinID != twin.TwinID {
		return false
	}

	if f.AccountID != nil && *f.AccountID != twin.AccountID {
		return false
	}

	if f.Relay != nil && *f.Relay != twin.Relay {
		return false
	}

	if f.PublicKey != nil && *f.PublicKey != twin.PublicKey {
		return false
	}

	return true
}
