package workloads

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
)

var volumeWorkload = Volume{
	Name:        "volumetest",
	SizeGB:      10,
	Description: "volume test description",
}

func TestVolumeWorkload(t *testing.T) {
	var volume zos.Workload

	t.Run("test_volume_from_map", func(t *testing.T) {
		volumeMap, err := ToMap(volumeWorkload)
		assert.NoError(t, err)

		volumeFromMap, err := NewWorkloadFromMap(volumeMap, &Volume{})
		assert.NoError(t, err)
		assert.Equal(t, volumeFromMap, &volumeWorkload)
	})

	t.Run("test_volume_from_workload", func(t *testing.T) {
		volume = volumeWorkload.ZosWorkload()

		volumeFromWorkload, err := NewVolumeFromWorkload(&volume)
		assert.NoError(t, err)

		assert.Equal(t, volumeFromWorkload, volumeWorkload)
	})
}

func TestVolumeWorkloadFailures(t *testing.T) {
	volume := volumeWorkload.ZosWorkload()

	volume.Data = nil
	_, err := NewVolumeFromWorkload(&volume)
	assert.Contains(t, err.Error(), "failed to get workload data")
}
