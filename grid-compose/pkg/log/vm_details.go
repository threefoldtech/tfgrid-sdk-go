package log

import (
	"fmt"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// WriteVmDetails writes the details of a VM to the output string builder
func WriteVmDetails(output *strings.Builder, vm workloads.VM, wl gridtypes.Workload, deploymentName string, nodeID uint32, dlAdded bool, vmAddresses string) {
	if !dlAdded {
		if len(vm.Mounts) < 1 {
			output.WriteString(fmt.Sprintf("%-15s | %-15d | %-15s | %-15s | %-15s | %-10s | %s\n",
				deploymentName, nodeID, vm.NetworkName, vm.Name, "None", wl.Result.State, vmAddresses))
			return
		}

		output.WriteString(fmt.Sprintf("%-15s | %-15d | %-15s | %-15s | %-15s | %-10s | %s\n",
			deploymentName, nodeID, vm.NetworkName, vm.Name, vm.Mounts[0].DiskName, wl.Result.State, vmAddresses))

		for _, mount := range vm.Mounts[1:] {
			output.WriteString(fmt.Sprintf("%-15s | %-15s | %-15s | %-15s | %-15s | %-10s | %s\n",
				strings.Repeat("-", 15), strings.Repeat("-", 15), strings.Repeat("-", 15), strings.Repeat("-", 15), mount.DiskName, wl.Result.State, strings.Repeat("-", 47)))
		}
	} else {
		if len(vm.Mounts) < 1 {
			output.WriteString(fmt.Sprintf("%-15s | %-15s | %-15s | %-15s | %-15s | %-10s | %s\n",
				strings.Repeat("-", 15), strings.Repeat("-", 15), strings.Repeat("-", 15), vm.Name, "None", wl.Result.State, vmAddresses))
			return
		}

		output.WriteString(fmt.Sprintf("%-15s | %-15s | %-15s | %-15s | %-15s | %-10s | %s\n",
			strings.Repeat("-", 15), strings.Repeat("-", 15), strings.Repeat("-", 15), vm.Name, vm.Mounts[0].DiskName, wl.Result.State, vmAddresses))

		for _, mount := range vm.Mounts[1:] {
			output.WriteString(fmt.Sprintf("%-15s | %-15s | %-15s | %-15s | %-15s | %-10s | %s\n",
				strings.Repeat("-", 15), strings.Repeat("-", 15), strings.Repeat("-", 15), strings.Repeat("-", 15), mount.DiskName, wl.Result.State, strings.Repeat("-", 15)))
		}
	}
}
