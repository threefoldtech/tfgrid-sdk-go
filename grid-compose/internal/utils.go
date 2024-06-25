package internal

import (
	"crypto/rand"
	"fmt"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func GetProjectName(key string, twinId uint32) string {
	return fmt.Sprintf("compose/%v/%v", twinId, key)
}

func GetVmAddresses(vm workloads.VM) string {
	var res string

	if vm.IP != "" {
		res += fmt.Sprintf("\twireguard: %v\n", vm.IP)
	}
	if vm.Planetary {
		res += fmt.Sprintf("\tyggdrasil: %v\n", vm.PlanetaryIP)
	}
	if vm.PublicIP {
		res += fmt.Sprintf("\tpublicIp4: %v\n", vm.ComputedIP)
	}
	if vm.PublicIP6 {
		res += fmt.Sprintf("\tpublicIp6: %v\n", vm.ComputedIP6)
	}
	if len(vm.MyceliumIPSeed) != 0 {
		res += fmt.Sprintf("\tmycelium: %v\n", vm.MyceliumIP)
	}

	return res
}

func GetRandomMyceliumIPSeed() ([]byte, error) {
	key := make([]byte, zos.MyceliumIPSeedLen)
	_, err := rand.Read(key)
	return key, err
}
