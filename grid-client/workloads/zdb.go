// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

const (
	ZDBModeUser = "user"
	ZDBModeSeq  = "seq"
)

// ZDB workload struct
type ZDB struct {
	Name        string `json:"name"`
	Password    string `json:"password"`
	Public      bool   `json:"public"`
	SizeGB      uint64 `json:"size"`
	Description string `json:"description"`
	Mode        string `json:"mode"`

	// OUTPUT
	IPs       []string `json:"ips"`
	Port      uint32   `json:"port"`
	Namespace string   `json:"namespace"`
}

// NewZDBFromWorkload generates a new zdb from a workload
func NewZDBFromWorkload(wl *gridtypes.Workload) (ZDB, error) {
	dataI, err := wl.WorkloadData()
	if err != nil {
		return ZDB{}, errors.Wrap(err, "failed to get workload data")
	}

	data, ok := dataI.(*zos.ZDB)
	if !ok {
		return ZDB{}, errors.Errorf("could not create zdb workload from data %v", dataI)
	}

	var result zos.ZDBResult

	if err := json.Unmarshal(wl.Result.Data, &result); err != nil {
		return ZDB{}, errors.Wrap(err, "failed to get zdb result")
	}

	return ZDB{
		Name:        wl.Name.String(),
		Description: wl.Description,
		Password:    data.Password,
		Public:      data.Public,
		SizeGB:      uint64(data.Size / gridtypes.Gigabyte),
		Mode:        data.Mode.String(),
		IPs:         result.IPs,
		Port:        uint32(result.Port),
		Namespace:   result.Namespace,
	}, nil
}

// ZosWorkload generates a workload from a zdb
func (z *ZDB) ZosWorkload() gridtypes.Workload {
	return gridtypes.Workload{
		Name:        gridtypes.Name(z.Name),
		Type:        zos.ZDBType,
		Description: z.Description,
		Version:     0,
		Data: gridtypes.MustMarshal(zos.ZDB{
			Size:     gridtypes.Unit(z.SizeGB) * gridtypes.Gigabyte,
			Mode:     zos.ZDBMode(z.Mode),
			Password: z.Password,
			Public:   z.Public,
		}),
	}
}

func (z *ZDB) Validate() error {
	if err := validateName(z.Name); err != nil {
		return errors.Wrap(err, "zdb name is invalid")
	}

	if z.SizeGB == 0 {
		return errors.New("zdb size should be a positive integer not zero")
	}

	if z.Mode != ZDBModeUser && z.Mode != ZDBModeSeq {
		return fmt.Errorf("invalid zdb mode '%s'", z.Mode)
	}

	return nil
}
