// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
)

// VMLightWorkload for tests
var VMLightWorkload = VMLight{
	Name:          "test",
	NodeID:        1,
	Flist:         "https://hub.grid.tf/tf-official-apps/base:latest.flist",
	FlistChecksum: "f94b5407f2e8635bd1b6b3dac7fef2d9",
	CPU:           2,
	MemoryMB:      1024,
	RootfsSizeMB:  20 * 1024,
	Entrypoint:    "/sbin/zinit init",
	EnvVars: map[string]string{
		"SSH_KEY": "",
	},
	IP:          "10.20.2.5",
	NetworkName: "testingNetwork",
}

func TestVMLightWorkload(t *testing.T) {
	var workloadsFromVM []zos.Workload
	var vmWorkload zos.Workload

	VMLightWorkload.Zlogs = []Zlog{ZlogWorkload}
	deployment := NewGridDeployment(1, 0, []zos.Workload{vmWorkload})

	t.Run("test vm from/to map", func(t *testing.T) {
		vmMap, err := ToMap(VMLightWorkload)
		assert.NoError(t, err)

		vmFromMap, err := NewWorkloadFromMap(vmMap, &VMLight{})
		assert.NoError(t, err)
		assert.Equal(t, vmFromMap, &VMLightWorkload)
	})

	t.Run("test_vm_from_workload", func(t *testing.T) {
		workloadsFromVM = VMLightWorkload.ZosWorkload()
		vmWorkload = workloadsFromVM[1]

		res, err := json.Marshal(zos.ZMachineLightResult{})
		assert.NoError(t, err)
		vmWorkload.Result.Data = res

		vmFromWorkload, err := NewVMLightFromWorkload(&vmWorkload, &deployment, 1)
		assert.NoError(t, err)

		// no result yet so they are set manually
		vmFromWorkload.Zlogs = []Zlog{ZlogWorkload}

		assert.Equal(t, vmFromWorkload, VMLightWorkload)
	})

	t.Run("test_mounts", func(t *testing.T) {
		zosZmachine, err := vmWorkload.ZMachineLightWorkload()
		assert.NoError(t, err)

		var dataMounts []zos.MachineMount
		for _, m := range zosZmachine.Mounts {
			dataMounts = append(dataMounts, zos.MachineMount{
				Name:       m.Name.String(),
				Mountpoint: m.Mountpoint,
			})
		}

		mountsOfVMWorkload := mounts(dataMounts)
		assert.Equal(t, mountsOfVMWorkload, VMLightWorkload.Mounts)
	})

	t.Run("test_vm_validate", func(t *testing.T) {
		assert.NoError(t, VMLightWorkload.Validate())
	})

	t.Run("test_vm_failed_validate", func(t *testing.T) {
		VMLightWorkload.CPU = 0
		assert.Error(t, VMLightWorkload.Validate())
	})
}
