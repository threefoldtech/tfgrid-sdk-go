// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"crypto/md5"
	"encoding/hex"

	"github.com/pkg/errors"
	zosTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// Zlog logger struct
type Zlog struct {
	Zmachine string `json:"zmachine"`
	Output   string `json:"output"`
}

// ZosWorkload generates a zlog workload
func (zlog *Zlog) ZosWorkload() zosTypes.Workload {
	url := []byte(zlog.Output)
	urlHash := md5.Sum(url)

	return zosTypes.Workload{
		Version: 0,
		Name:    hex.EncodeToString(urlHash[:]),
		Type:    zosTypes.ZLogsType,
		Data: zosTypes.MustMarshal(zosTypes.ZLogs{
			ZMachine: zlog.Zmachine,
			Output:   zlog.Output,
		}),
	}
}

func zlogs(dl *zosTypes.Deployment, name string) []Zlog {
	var res []Zlog
	for _, wl := range dl.ByType(zosTypes.ZLogsType) {
		if !wl.Result.State.IsOkay() {
			continue
		}

		var dataI interface{}

		dataI, err := wl.Workload3().WorkloadData()
		if err != nil {
			dataI, err = wl.Workload4().WorkloadData()
			if err != nil {
				continue
			}
		}

		data, ok := dataI.(*zos.ZLogs)
		if !ok {
			continue
		}

		if data.ZMachine.String() != name {
			continue
		}

		res = append(res, Zlog{
			Output:   data.Output,
			Zmachine: name,
		})
	}
	return res
}

func (z *Zlog) Validate() error {
	if err := validateName(z.Zmachine); err != nil {
		return errors.Wrap(err, "zmachine name is invalid")
	}

	return nil
}
