package mock

import (
	"context"
	"slices"
	"sort"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

func (g *GridProxyMockClient) PublicIps(ctx context.Context, filter types.PublicIpFilter, limit types.Limit) ([]types.PublicIP, uint, error) {
	res := []types.PublicIP{}
	if limit.Page == 0 {
		limit.Page = 1
	}
	if limit.Size == 0 {
		limit.Size = 50
	}

	for _, ip := range g.data.PublicIPs {
		if ip.satisfies(filter, &g.data) {
			res = append(res, types.PublicIP{
				IP:         ip.IP,
				ID:         ip.ID,
				Gateway:    ip.Gateway,
				ContractID: ip.ContractID,
				FarmID:     g.data.FarmIDMap[ip.FarmID],
			})
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].ID < res[j].ID
	})

	res, count := getPage(res, limit)

	return res, uint(count), nil
}

func (ip *PublicIp) satisfies(f types.PublicIpFilter, data *DBData) bool {
	if f.Free != nil &&
		*f.Free != (ip.ContractID == 0) {
		return false
	}

	if len(f.FarmIDs) != 0 &&
		!slices.Contains(f.FarmIDs, data.FarmIDMap[ip.FarmID]) {
		return false
	}

	if f.Ip != nil &&
		*f.Ip != ip.IP {
		return false
	}

	if f.Gateway != nil &&
		*f.Gateway != ip.Gateway {
		return false
	}

	return true
}
