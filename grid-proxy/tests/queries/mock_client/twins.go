package mock

import (
	"fmt"
	"reflect"
	"sort"

	proxytypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

var twinFilterFieldValidator = map[string]func(twin Twin, f proxytypes.TwinFilter) bool{
	"TwinID": func(twin Twin, f proxytypes.TwinFilter) bool {
		return f.TwinID == nil || twin.TwinID == *f.TwinID
	},
	"AccountID": func(twin Twin, f proxytypes.TwinFilter) bool {
		return f.AccountID == nil || twin.AccountID == *f.AccountID
	},
	"Relay": func(twin Twin, f proxytypes.TwinFilter) bool {
		return f.Relay == nil || *f.Relay == twin.Relay
	},
	"PublicKey": func(twin Twin, f proxytypes.TwinFilter) bool {
		return f.PublicKey == nil || *f.PublicKey == twin.PublicKey
	},
}

// Twins returns twins with the given filters and pagination parameters
func (g *GridProxyMockClient) Twins(filter proxytypes.TwinFilter, limit proxytypes.Limit) (res []proxytypes.Twin, totalCount int, err error) {
	res = []proxytypes.Twin{}

	if limit.Page == 0 {
		limit.Page = 1
	}
	if limit.Size == 0 {
		limit.Size = 50
	}
	for _, twin := range g.data.Twins {
		satisfies, err := twinSatisfies(twin, filter)
		if err != nil {
			return res, totalCount, err
		}

		if satisfies {
			res = append(res, proxytypes.Twin{
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

func twinSatisfies(twin Twin, f proxytypes.TwinFilter) (bool, error) {
	v := reflect.ValueOf(f)
	for i := 0; i < v.NumField(); i++ {
		valid, ok := twinFilterFieldValidator[v.Type().Field(i).Name]
		if !ok {
			return false, fmt.Errorf("Field %s has no validator", v.Type().Field(i).Name)
		}

		if !valid(twin, f) {
			return false, nil
		}
	}

	return true, nil
}
