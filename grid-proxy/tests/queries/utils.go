package test

import (
	"fmt"
	"math/rand"
	"reflect"
	"unicode"

	"github.com/threefoldtech/zos_sdk_go/grid-proxy/pkg/types"
)

type Filter interface {
	types.ContractFilter | types.NodeFilter | types.FarmFilter | types.TwinFilter | types.StatsFilter | types.PublicIpFilter
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

// getRandomItemSubstring
// - selects a random item from input slice
// - transform case the selected item
// - extracts a random substring from the selected item
func getRandomItemSubstring(items []string) string {
	if len(items) == 0 {
		return ""
	}

	item := items[rand.Intn(len(items))]
	item = changeCase(item)
	if len(item) == 0 {
		return ""
	}

	runesList := []rune(item)
	a, b := rand.Intn(len(runesList)), rand.Intn(len(runesList))
	if a > b {
		a, b = b, a
	}
	return string(runesList[a : b+1])
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
