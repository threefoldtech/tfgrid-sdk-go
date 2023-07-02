// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// Disk struct
type Disk struct {
	Name        string `json:"name"`
	SizeGB      int    `json:"size"`
	Description string `json:"description"`
}

// NewDiskFromWorkload generates a new disk from a workload
func NewDiskFromWorkload(wl *gridtypes.Workload) (Disk, error) {
	dataI, err := wl.WorkloadData()
	if err != nil {
		return Disk{}, errors.Wrap(err, "failed to get workload data")
	}

	data, ok := dataI.(*zos.ZMount)
	if !ok {
		return Disk{}, errors.Errorf("could not create disk workload from data %v", dataI)
	}

	return Disk{
		Name:        wl.Name.String(),
		Description: wl.Description,
		SizeGB:      int(data.Size / gridtypes.Gigabyte),
	}, nil
}

// ZosWorkload generates a workload from a disk
func (d *Disk) ZosWorkload() gridtypes.Workload {
	return gridtypes.Workload{
		Name:        gridtypes.Name(d.Name),
		Version:     0,
		Type:        zos.ZMountType,
		Description: d.Description,
		Data: gridtypes.MustMarshal(zos.ZMount{
			Size: gridtypes.Unit(d.SizeGB) * gridtypes.Gigabyte,
		}),
	}
}
