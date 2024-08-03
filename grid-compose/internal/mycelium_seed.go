package internal

import (
	"crypto/rand"

	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func getRandomMyceliumIPSeed() ([]byte, error) {
	key := make([]byte, zos.MyceliumIPSeedLen)
	_, err := rand.Read(key)
	return key, err
}
