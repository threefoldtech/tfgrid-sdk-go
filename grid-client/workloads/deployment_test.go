// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
)

func TestNewDeployment(t *testing.T) {
	var zosDeployment zos.Deployment
	deployment := NewDeployment(
		"test", 1, "", nil, n.Name,
		[]Disk{DiskWorkload},
		[]ZDB{ZDBWorkload},
		[]VM{VMWorkload},
		[]VMLight{},
		[]QSFS{QSFSWorkload},
		[]Volume{volumeWorkload},
	)

	t.Run("test deployment validate", func(t *testing.T) {
		assert.NoError(t, deployment.Validate())
	})

	t.Run("test zos deployment", func(t *testing.T) {
		var err error
		zosDeployment, err = deployment.ZosDeployment(1)
		assert.NoError(t, err)

		workloads := []zos.Workload{DiskWorkload.ZosWorkload(), ZDBWorkload.ZosWorkload()}
		workloads = append(workloads, VMWorkload.ZosWorkload()...)
		QSFS, err := QSFSWorkload.ZosWorkload()
		assert.NoError(t, err)
		workloads = append(workloads, QSFS, volumeWorkload.ZosWorkload())

		newZosDeployment := NewGridDeployment(1, 0, workloads)
		assert.Equal(t, newZosDeployment, zosDeployment)
	})

	t.Run("test deployment used ips", func(t *testing.T) {
		for i := range zosDeployment.Workloads {
			zosDeployment.Workloads[i].Result.State = "ok"
		}

		res, err := json.Marshal(zos.ZMachineResult{})
		assert.NoError(t, err)
		zosDeployment.Workloads[3].Result.Data = res

		usedIPs, err := GetUsedIPs(zosDeployment, 1)
		assert.NoError(t, err)
		assert.Equal(t, usedIPs, []byte{5})
	})

	t.Run("test deployment match", func(t *testing.T) {
		dlCp := deployment
		deployment.Match([]Disk{}, []QSFS{}, []ZDB{}, []VM{}, []VMLight{}, []Volume{})
		assert.Equal(t, deployment, dlCp)
	})

	t.Run("test deployment nullify", func(t *testing.T) {
		deployment.Nullify()
		assert.Equal(t, deployment.Vms, []VM(nil))
		assert.Equal(t, deployment.Disks, []Disk(nil))
		assert.Equal(t, deployment.QSFS, []QSFS(nil))
		assert.Equal(t, deployment.Zdbs, []ZDB(nil))
		assert.Equal(t, deployment.Volumes, []Volume(nil))
		assert.Equal(t, deployment.ContractID, uint64(0))
	})
}
