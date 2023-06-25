package mock

import (
	"sort"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// Twins returns twins with the given filters and pagination parameters
func (g *GridProxyMockClient) Twins(filter types.TwinFilter, limit types.Limit) ([]types.Twin, int, error) {
	res := []types.Twin{}

	for _, twin := range g.data.Twins {
		if twin.twinSatisfies(filter) {
			res = append(res, types.Twin{
				TwinID:    twin.TwinID,
				AccountID: twin.AccountID,
				Relay:     twin.Relay,
				PublicKey: twin.PublicKey,
			})
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].TwinID < res[j].TwinID
	})

	res, count := getPage(res, limit)

	return res, count, nil
}

func (t *DBTwin) twinSatisfies(f types.TwinFilter) bool {
	if f.TwinID != nil && t.TwinID != *f.TwinID {
		return false
	}

	if f.AccountID != nil && t.AccountID != *f.AccountID {
		return false
	}

	return true
}
