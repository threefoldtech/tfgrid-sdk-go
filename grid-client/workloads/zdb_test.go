// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
)

// ZDBWorkload for tests
var ZDBWorkload = ZDB{
	Name:        "test",
	Password:    "password",
	Public:      true,
	SizeGB:      100,
	Description: "test des",
	Mode:        "user",
	//IPs:         ips,
	Port:      0,
	Namespace: "",
}

func TestZDB(t *testing.T) {
	var zdbWorkload zos.Workload

	t.Run("test zdb to/from map", func(t *testing.T) {
		zdbMap, err := ToMap(ZDBWorkload)
		assert.NoError(t, err)

		zdbFromMap, err := NewWorkloadFromMap(zdbMap, &ZDB{})
		assert.NoError(t, err)

		assert.Equal(t, zdbFromMap, &ZDBWorkload)
	})

	t.Run("test_zdb_from_workload", func(t *testing.T) {
		zdbWorkload = ZDBWorkload.ZosWorkload()

		res, err := json.Marshal(zos.ZDBResult{})
		assert.NoError(t, err)
		zdbWorkload.Result.Data = res

		zdbFromWorkload, err := NewZDBFromWorkload(&zdbWorkload)
		assert.NoError(t, err)
		assert.Equal(t, ZDBWorkload, zdbFromWorkload)
	})
}
