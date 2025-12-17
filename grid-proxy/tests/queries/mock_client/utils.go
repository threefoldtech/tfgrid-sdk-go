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

func satisfiesFreeCapacityFilter(mruVal, sruVal, hruVal uint64, sliceMru, sliceSru, sliceHru uint64, free NodeResourcesTotal) bool {
	if (mruVal > 0 && sliceMru == 0) || (sruVal > 0 && sliceSru == 0) || (hruVal > 0 && sliceHru == 0) {
		return false
	}

	var slicesNeededMRU, slicesNeededSRU, slicesNeededHRU uint64
	if sliceMru > 0 {
		slicesNeededMRU = (mruVal + sliceMru - 1) / sliceMru
	}
	if sliceSru > 0 {
		slicesNeededSRU = (sruVal + sliceSru - 1) / sliceSru
	}
	if sliceHru > 0 {
		slicesNeededHRU = (hruVal + sliceHru - 1) / sliceHru
	}

	slicesNeeded := slicesNeededMRU
	if slicesNeededSRU > slicesNeeded {
		slicesNeeded = slicesNeededSRU
	}
	if slicesNeededHRU > slicesNeeded {
		slicesNeeded = slicesNeededHRU
	}

	var slicesAvailableMRU, slicesAvailableSRU, slicesAvailableHRU uint64 = 1e18, 1e18, 1e18
	if sliceMru > 0 {
		slicesAvailableMRU = free.MRU / sliceMru
	}
	if sliceSru > 0 {
		slicesAvailableSRU = free.SRU / sliceSru
	}
	if sliceHru > 0 {
		slicesAvailableHRU = free.HRU / sliceHru
	}

	slicesAvailable := slicesAvailableMRU
	if slicesAvailableSRU < slicesAvailable {
		slicesAvailable = slicesAvailableSRU
	}
	if slicesAvailableHRU < slicesAvailable {
		slicesAvailable = slicesAvailableHRU
	}

	return slicesNeeded > 0 && slicesNeeded <= slicesAvailable
}
