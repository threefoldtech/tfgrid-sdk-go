// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
)

// ConstructPublicIPWorkload constructs a public IP workload
func ConstructPublicIPWorkload(workloadName string, ipv4 bool, ipv6 bool) zos.Workload {
	return zos.Workload{
		Version: 0,
		Name:    workloadName,
		Type:    zos.PublicIPType,
		Data: zos.MustMarshal(zos.PublicIP{
			V4: ipv4,
			V6: ipv6,
		}),
	}
}
