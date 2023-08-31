package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"strings"
)

func rnd(min, max uint64) (uint64, error) {
	if max-min+1 <= 0 {
		return 0, errors.New("min cannot be greater than max")
	}
	return uint64(rand.Intn(int(max)-int(min)+1) + int(min)), nil
}

func flip(success float32) bool {
	return rand.Float32() < success
}

func randomIPv4() net.IP {
	ip := make([]byte, 4)
	rand.Read(ip)
	return net.IP(ip)
}

// IPv4Subnet gets the ipv4 subnet given the ip
func IPv4Subnet(ip net.IP) *net.IPNet {
	return &net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(24, 32),
	}
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

func objectToTupleString(v interface{}) string {
	vals := "("
	val := reflect.ValueOf(v)
	for i := 0; i < val.NumField(); i++ {
		if i == 0 {
			v := fmt.Sprint(val.Field(i))
			if v == "" {
				v = "NULL"
			}
			if v != "NULL" && val.Field(i).Type().Name() == "string" {
				v = fmt.Sprintf(`'%s'`, v)
			}
			vals = fmt.Sprintf("%s%s", vals, v)
		} else {
			v := fmt.Sprint(val.Field(i))
			if v == "" {
				v = "NULL"
			}
			if v != "NULL" && val.Field(i).Type().Name() == "string" {
				v = fmt.Sprintf(`'%s'`, v)
			}
			if v != "NULL" && val.Field(i).Type().Name() == "nodePower" {
				// Construct the nodePower object
				val2 := val.Field(i)
				power := make(map[string]string)
				for j := 0; j < val2.NumField(); j++ {
					fieldName := strings.ToLower(val2.Type().Field(j).Name)
					fieldValue := val2.Field(j).String()
					power[fieldName] = fieldValue
				}

				// Marshal the power map to JSON and wrap it in quotes
				powerJSON, err := json.Marshal(power)
				if err != nil {
					panic(err)
				}
				v = fmt.Sprintf("'%s'", string(powerJSON))
			}
			vals = fmt.Sprintf("%s, %s", vals, v)
		}
	}
	return fmt.Sprintf("%s)", vals)
}
func popRandom(l []uint64) ([]uint64, uint64, error) {
	idx, err := rnd(0, uint64(len(l)-1))
	if err != nil {
		return nil, 0, err
	}
	e := l[idx]
	l[idx], l[len(l)-1] = l[len(l)-1], l[idx]
	return l[:len(l)-1], e, nil
}
