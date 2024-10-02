package mock

import (
	"slices"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

type Result interface {
	types.Contract | types.Farm | types.Node | types.Twin | types.PublicIP
}

func CalcFreeResources(total NodeResourcesTotal, used NodeResourcesTotal) NodeResourcesTotal {
	return NodeResourcesTotal{
		HRU: total.HRU - used.HRU,
		SRU: total.SRU - used.SRU,
		MRU: total.MRU - used.MRU,
	}
}

func stringMatch(str string, sub_str string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(sub_str))
}

func getPage[R Result](res []R, limit types.Limit) ([]R, int) {
	if len(res) == 0 {
		return []R{}, 0
	}

	if limit.Page == 0 {
		limit.Page = 1
	}

	if limit.Size == 0 {
		limit.Size = 50
	}

	start, end := (limit.Page-1)*limit.Size, limit.Page*limit.Size

	if start >= uint64(len(res)) {
		start = uint64(len(res) - 1)
	}

	if end > uint64(len(res)) {
		end = uint64(len(res))
	}

	totalCount := 0
	if limit.RetCount {
		totalCount = len(res)
	}

	res = res[start:end]

	return res, totalCount
}

func sliceContains(set []string, subset []string) bool {
	for _, item := range subset {
		if !slices.Contains(set, item) {
			return false
		}
	}

	return true
}
