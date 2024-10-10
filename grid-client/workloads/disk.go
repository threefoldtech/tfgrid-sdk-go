// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"github.com/pkg/errors"
	zosTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// Disk struct
type Disk struct {
	Name        string `json:"name"`
	SizeGB      uint64 `json:"size"`
	Description string `json:"description"`
}

// NewDiskFromWorkload generates a new disk from a workload
func NewDiskFromWorkload(wl *zosTypes.Workload) (Disk, error) {
	var dataI interface{}

	dataI, err := wl.Workload3().WorkloadData()
	if err != nil {
		dataI, err = wl.Workload4().WorkloadData()
		if err != nil {
			return Disk{}, errors.Wrap(err, "failed to get workload data")
		}
	}

	data, ok := dataI.(*zos.ZMount)
	if !ok {
		return Disk{}, errors.Errorf("could not create disk workload from data %v", dataI)
	}

	return Disk{
		Name:        wl.Name,
		Description: wl.Description,
		SizeGB:      uint64(data.Size) / zosTypes.Gigabyte,
	}, nil
}

// ZosWorkload generates a workload from a disk
func (d *Disk) ZosWorkload() zosTypes.Workload {
	return zosTypes.Workload{
		Name:        d.Name,
		Version:     0,
		Type:        zosTypes.ZMountType,
		Description: d.Description,
		Data: zosTypes.MustMarshal(zosTypes.ZMount{
			Size: d.SizeGB * zosTypes.Gigabyte,
		}),
	}
}

func (d *Disk) Validate() error {
	if err := validateName(d.Name); err != nil {
		return errors.Wrap(err, "disk name is invalid")
	}

	if d.SizeGB == 0 {
		return errors.New("disk size should be a positive integer not zero")
	}

	return nil
}
