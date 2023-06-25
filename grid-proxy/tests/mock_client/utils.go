package mock

import (
	"fmt"
	"strings"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

func isUp(timestamp int64) bool {
	return timestamp > time.Now().Unix()-nodeStateFactor*int64(reportInterval.Seconds())
}

func CalculateFreeResources(total DBNodeResourcesTotal, used DBNodeResourcesTotal) (DBNodeResourcesTotal, error) {
	if total.MRU < used.MRU {
		return DBNodeResourcesTotal{}, fmt.Errorf("total mru %d is less than used mru %d", total.MRU, used.MRU)
	}

	if total.HRU < used.HRU {
		return DBNodeResourcesTotal{}, fmt.Errorf("total hru %d is less than used hru %d", total.HRU, used.HRU)
	}

	if total.SRU < used.SRU {
		return DBNodeResourcesTotal{}, fmt.Errorf("total sru %d is less than used sru %d", total.SRU, used.SRU)
	}

	return DBNodeResourcesTotal{
		HRU: total.HRU - used.HRU,
		SRU: total.SRU - used.SRU,
		MRU: total.MRU - used.MRU,
	}, nil
}

func stringMatch(str string, sub_str string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(sub_str))
}

type Result interface {
	types.Contract | types.Farm | types.Node | types.Twin
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

	totalCount := len(res)
	res = res[start:end]

	return res, totalCount
}
