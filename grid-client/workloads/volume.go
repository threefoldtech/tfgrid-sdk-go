package workloads

import (
	"fmt"

	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// Volume struct
type Volume struct {
	Name        string `json:"name"`
	SizeGB      int    `json:"size"`
	Description string `json:"description"`
}

// NewVolumeFromWorkload generates a new volume from a workload
func NewVolumeFromWorkload(wl *gridtypes.Workload) (Volume, error) {
	dataI, err := wl.WorkloadData()
	if err != nil {
		return Volume{}, fmt.Errorf("failed to get workload data: %w", err)
	}

	data, ok := dataI.(*zos.Volume)
	if !ok {
		return Volume{}, fmt.Errorf("could not create volume workload from data %v", dataI)
	}

	return Volume{
		Name:        wl.Name.String(),
		Description: wl.Description,
		SizeGB:      int(data.Size / gridtypes.Gigabyte),
	}, nil
}

// ZosWorkload generates a workload from a volume
func (v *Volume) ZosWorkload() gridtypes.Workload {
	return gridtypes.Workload{
		Name:        gridtypes.Name(v.Name),
		Version:     0,
		Type:        zos.VolumeType,
		Description: v.Description,
		Data: gridtypes.MustMarshal(zos.ZMount{
			Size: gridtypes.Unit(v.SizeGB) * gridtypes.Gigabyte,
		}),
	}
}
