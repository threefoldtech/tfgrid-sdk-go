package internal

import (
	"fmt"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

func GetProjectName(key string, twinId uint32) string {
	return fmt.Sprintf("compose/%v/%v", twinId, key)
}

func GetVmAddresses(vm workloads.VM) string {
	var res string

	res += fmt.Sprintf("ygg: %v", vm.PlanetaryIP)

	return res
}
