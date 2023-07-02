// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// ZDB workload struct
type ZDB struct {
	Name        string   `json:"name"`
	Password    string   `json:"password"`
	Public      bool     `json:"public"`
	Size        int      `json:"size"`
	Description string   `json:"description"`
	Mode        string   `json:"mode"`
	IPs         []string `json:"ips"`
	Port        uint32   `json:"port"`
	Namespace   string   `json:"namespace"`
}

// NewZDBFromMap converts a map including zdb data to a zdb struct
func NewZDBFromMap(zdb map[string]interface{}) (ZDB, error) {
	bytes, err := json.Marshal(zdb)
	if err != nil {
		return ZDB{}, errors.Wrap(err, "failed to marshal zdb map")
	}

	res := ZDB{}
	err = json.Unmarshal(bytes, &res)
	if err != nil {
		return ZDB{}, errors.Wrap(err, "failed to unmarshal zdb data")
	}

	return res, nil
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
		Size:        int(data.Size / gridtypes.Gigabyte),
		Mode:        data.Mode.String(),
		IPs:         result.IPs,
		Port:        uint32(result.Port),
		Namespace:   result.Namespace,
	}, nil
}

// ToMap converts a zdb to a map(dict) object
func (z *ZDB) ToMap() (map[string]interface{}, error) {
	var zdbMap map[string]interface{}
	zdbBytes, err := json.Marshal(z)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal zdb data")
	}

	err = json.Unmarshal(zdbBytes, &zdbMap)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal zdb bytes to map")
	}

	return zdbMap, nil
}

// ZosWorkload generates a workload from a zdb
func (z *ZDB) ZosWorkload() gridtypes.Workload {
	return gridtypes.Workload{
		Name:        gridtypes.Name(z.Name),
		Type:        zos.ZDBType,
		Description: z.Description,
		Version:     0,
		Data: gridtypes.MustMarshal(zos.ZDB{
			Size:     gridtypes.Unit(z.Size) * gridtypes.Gigabyte,
			Mode:     zos.ZDBMode(z.Mode),
			Password: z.Password,
			Public:   z.Public,
		}),
	}
}
