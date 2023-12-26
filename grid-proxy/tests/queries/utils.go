package test

import (
	"fmt"
	"math/rand"
	"reflect"
	"unicode"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	mock "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/tests/queries/mock_client"
)

type Filter interface {
	types.ContractFilter | types.NodeFilter | types.FarmFilter | types.TwinFilter | types.StatsFilter
}

func calcFreeResources(total mock.NodeResourcesTotal, used mock.NodeResourcesTotal) mock.NodeResourcesTotal {
	mru := total.MRU - used.MRU
	if mru < 0 {
		mru = 0
	}

	hru := total.HRU - used.HRU
	if hru < 0 {
		hru = 0
	}

	sru := total.SRU - used.SRU
	if sru < 0 {
		sru = 0
	}

	return mock.NodeResourcesTotal{
		HRU: hru,
		SRU: sru,
		MRU: mru,
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
	if len(s) == 0 {
		return s
	}

	runesList := []rune(s)
	idx := rand.Intn(len(runesList))
	runesList[idx] = unicode.ToUpper(runesList[idx])
	return string(runesList)
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
