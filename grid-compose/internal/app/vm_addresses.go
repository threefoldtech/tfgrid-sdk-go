package app

import (
	"fmt"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

func getVmAddresses(vm workloads.VM) string {
	var addresses strings.Builder

	if vm.IP != "" {
		addresses.WriteString(fmt.Sprintf("wireguard: %v, ", vm.IP))
	}
	if vm.Planetary {
		addresses.WriteString(fmt.Sprintf("yggdrasil: %v, ", vm.PlanetaryIP))
	}
	if vm.PublicIP {
		addresses.WriteString(fmt.Sprintf("publicIp4: %v, ", vm.ComputedIP))
	}
	if vm.PublicIP6 {
		addresses.WriteString(fmt.Sprintf("publicIp6: %v, ", vm.ComputedIP6))
	}
	if len(vm.MyceliumIPSeed) != 0 {
		addresses.WriteString(fmt.Sprintf("mycelium: %v, ", vm.MyceliumIP))
	}

	return strings.TrimSuffix(addresses.String(), ", ")
}
