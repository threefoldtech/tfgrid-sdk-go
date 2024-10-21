package workloads

import (
	"fmt"

	"github.com/pkg/errors"
	zosTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// Volume struct
type Volume struct {
	Name        string `json:"name"`
	SizeGB      uint64 `json:"size"`
	Description string `json:"description"`
}

// NewVolumeFromWorkload generates a new volume from a workload
func NewVolumeFromWorkload(wl *zosTypes.Workload) (Volume, error) {
	var dataI interface{}

	dataI, err := wl.Workload3().WorkloadData()
	if err != nil {
		dataI, err = wl.Workload4().WorkloadData()
		if err != nil {
			return Volume{}, errors.Wrap(err, "failed to get workload data")
		}
	}

	data, ok := dataI.(*zos.Volume)
	if !ok {
		return Volume{}, fmt.Errorf("could not create volume workload from data %v", dataI)
	}

	return Volume{
		Name:        wl.Name,
		Description: wl.Description,
		SizeGB:      uint64(data.Size) / zosTypes.Gigabyte,
	}, nil
}

// ZosWorkload generates a workload from a volume
func (v *Volume) ZosWorkload() zosTypes.Workload {
	return zosTypes.Workload{
		Name:        v.Name,
		Version:     0,
		Type:        zosTypes.VolumeType,
		Description: v.Description,
		Data: zosTypes.MustMarshal(zosTypes.ZMount{
			Size: v.SizeGB * zosTypes.Gigabyte,
		}),
	}
}

func (v *Volume) Validate() error {
	if err := validateName(v.Name); err != nil {
		return errors.Wrap(err, "volume name is invalid")
	}

	if v.SizeGB == 0 {
		return errors.New("volume size should be a positive integer not zero")
	}

	return nil
}
