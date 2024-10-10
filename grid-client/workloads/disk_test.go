// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
)

// DiskWorkload to be used for tests
var DiskWorkload = Disk{
	Name:        "test",
	SizeGB:      10,
	Description: "disk test description",
}

func TestDiskWorkload(t *testing.T) {
	var disk zos.Workload

	t.Run("test_disk_from_map", func(t *testing.T) {
		diskMap, err := ToMap(DiskWorkload)
		assert.NoError(t, err)

		diskFromMap, err := NewWorkloadFromMap(diskMap, &Disk{})
		assert.NoError(t, err)
		assert.Equal(t, diskFromMap, &DiskWorkload)
	})

	t.Run("test_disk_from_workload", func(t *testing.T) {
		disk = DiskWorkload.ZosWorkload()

		diskFromWorkload, err := NewDiskFromWorkload(&disk)
		assert.NoError(t, err)

		assert.Equal(t, diskFromWorkload, DiskWorkload)
	})
}

func TestDiskWorkloadFailures(t *testing.T) {
	disk := DiskWorkload.ZosWorkload()

	disk.Data = nil
	_, err := NewDiskFromWorkload(&disk)
	assert.Contains(t, err.Error(), "failed to get workload data")
}
