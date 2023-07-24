package test

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	mock "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/tests/queries/mock_client"
)

type Filter interface {
	types.ContractFilter | types.NodeFilter | types.FarmFilter | types.TwinFilter | types.StatsFilter
}

func calcFreeResources(total mock.NodeResourcesTotal, used mock.NodeResourcesTotal) mock.NodeResourcesTotal {
	if total.MRU < used.MRU {
		panic("total mru is less than mru")
	}
	if total.HRU < used.HRU {
		panic("total hru is less than hru")
	}
	if total.SRU < used.SRU {
		panic("total sru is less than sru")
	}
	return mock.NodeResourcesTotal{
		HRU: total.HRU - used.HRU,
		SRU: total.SRU - used.SRU,
		MRU: total.MRU - used.MRU,
	}
}

func flip(success float32) bool {
	return rand.Float32() < success
}

func rndref(min, max uint64) *uint64 {
	v := rand.Uint64()%(max-min+1) + min
	return &v
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func changeCase(s string) string {
	idx := rand.Intn(len(s))
	return strings.Replace(s, string(s[idx]), strings.ToUpper(string(s[idx])), 1)
}

func stringMatch(str string, sub_str string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(sub_str))
}

func SerializeFilter[F Filter](f F) string {
	res := ""
	v := reflect.ValueOf(f)
	for i := 0; i < v.NumField(); i++ {
		if !v.Field(i).IsNil() {
			res = fmt.Sprintf("%s%s : %+v\n", res, v.Type().Field(i).Name, reflect.Indirect(v.Field(i)))
		}

	}

	return res
}
