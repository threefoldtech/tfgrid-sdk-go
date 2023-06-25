package test

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
)

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

func serializeFilter(f interface{}) string {
	res := "Used Filter:\n"
	v := reflect.ValueOf(f)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.IsNil() {
			fieldVal := reflect.Indirect(field)
			fieldName := v.Type().Field(i).Name
			res = fmt.Sprintf("%s\t%+v: %+v\n", res, fieldName, fieldVal)
		}

	}

	return res
}
