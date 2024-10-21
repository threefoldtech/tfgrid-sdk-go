// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
)

// VMWorkload for tests
var VMWorkload = VM{
	Name:          "test",
	NodeID:        1,
	Flist:         "https://hub.grid.tf/tf-official-apps/base:latest.flist",
	FlistChecksum: "f94b5407f2e8635bd1b6b3dac7fef2d9",
	CPU:           2,
	PublicIP:      true,
	Planetary:     true,
	MemoryMB:      1024,
	RootfsSizeMB:  20 * 1024,
	Entrypoint:    "/sbin/zinit init",
	EnvVars: map[string]string{
		"SSH_KEY": "",
	},
	IP:          "10.20.2.5",
	NetworkName: "testingNetwork",
}

func TestVMWorkload(t *testing.T) {
	var workloadsFromVM []zos.Workload
	var vmWorkload zos.Workload

	VMWorkload.Zlogs = []Zlog{ZlogWorkload}

	pubIPWorkload := zos.Workload{
		Version: 0,
		Name:    "testip",
		Type:    zos.PublicIPType,
		Data: zos.MustMarshal(zos.PublicIP{
			V4: true,
			V6: false,
		}),
	}

	pubIPWorkload.Result.State = "ok"
	deployment := NewGridDeployment(1, 0, []zos.Workload{vmWorkload, pubIPWorkload})

	t.Run("test vm from/to map", func(t *testing.T) {
		vmMap, err := ToMap(VMWorkload)
		assert.NoError(t, err)

		vmFromMap, err := NewWorkloadFromMap(vmMap, &VM{})
		assert.NoError(t, err)
		assert.Equal(t, vmFromMap, &VMWorkload)
	})

	t.Run("test_vm_from_workload", func(t *testing.T) {
		workloadsFromVM = VMWorkload.ZosWorkload()
		vmWorkload = workloadsFromVM[2]

		res, err := json.Marshal(zos.ZMachineResult{})
		assert.NoError(t, err)
		vmWorkload.Result.Data = res

		vmFromWorkload, err := NewVMFromWorkload(&vmWorkload, &deployment, 1)
		assert.NoError(t, err)

		// no result yet so they are set manually
		vmFromWorkload.Planetary = true
		vmFromWorkload.PublicIP = true
		vmFromWorkload.Zlogs = []Zlog{ZlogWorkload}

		assert.Equal(t, vmFromWorkload, VMWorkload)
	})

	t.Run("test_pubIP_from_deployment", func(t *testing.T) {
		pubIP, err := pubIP(&deployment, "testip")
		assert.NoError(t, err)
		assert.Equal(t, pubIP.HasIPv6(), false)
	})

	t.Run("test_mounts", func(t *testing.T) {
		zosZmachine, err := vmWorkload.ZMachineWorkload()
		assert.NoError(t, err)

		var dataMounts []zos.MachineMount
		for _, m := range zosZmachine.Mounts {
			dataMounts = append(dataMounts, zos.MachineMount{
				Name:       m.Name.String(),
				Mountpoint: m.Mountpoint,
			})
		}

		mountsOfVMWorkload := mounts(dataMounts)
		assert.Equal(t, mountsOfVMWorkload, VMWorkload.Mounts)
	})

	t.Run("test_vm_validate", func(t *testing.T) {
		assert.NoError(t, VMWorkload.Validate())
	})

	t.Run("test_vm_failed_validate", func(t *testing.T) {
		VMWorkload.CPU = 0
		assert.Error(t, VMWorkload.Validate())
	})
}
