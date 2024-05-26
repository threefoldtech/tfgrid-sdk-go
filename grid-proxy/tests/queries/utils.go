package test

import (
	"fmt"
	"math/rand"
	"reflect"
	"unicode"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

type Filter interface {
	types.ContractFilter | types.NodeFilter | types.FarmFilter | types.TwinFilter | types.StatsFilter
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

func getRandomSliceFrom[T any](original []T, length int) []T {
	copied := append([]T(nil), original...)
	rand.Shuffle(len(copied), func(i, j int) {
		copied[i], copied[j] = copied[j], copied[i]
	})
	randomSlice := copied[:length]
	return randomSlice
}
