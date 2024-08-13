package utils

import (
	"crypto/rand"

	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func GetRandomMyceliumIPSeed() ([]byte, error) {
	key := make([]byte, zos.MyceliumIPSeedLen)
	_, err := rand.Read(key)
	return key, err
}
