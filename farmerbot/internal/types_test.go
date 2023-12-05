package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

var cap = capacity{
	CRU: 1,
	SRU: 1,
	MRU: 1,
	HRU: 1,
}

func TestCapacityModel(t *testing.T) {
	assert.False(t, cap.isEmpty())

	resultSub := cap.subtract(cap)
	assert.True(t, resultSub.isEmpty())

	cap.add(cap)
	assert.Equal(t, cap.CRU, uint64(2))
}

func TestNodeModel(t *testing.T) {
	node := node{
		Node: substrate.Node{ID: 1, TwinID: 1},
		Resources: ConsumableResources{
			OverProvisionCPU: 1,
			Total:            cap,
		},
		PowerState: ON,
	}

	t.Run("test update node resources", func(t *testing.T) {
		zosResources := zosResourcesStatistics{
			Total: gridtypes.Capacity{
				CRU:   cap.CRU,
				SRU:   gridtypes.Unit(cap.SRU),
				HRU:   gridtypes.Unit(cap.HRU),
				MRU:   gridtypes.Unit(cap.MRU),
				IPV4U: 1,
			},
			Used:   gridtypes.Capacity{},
			System: gridtypes.Capacity{},
		}

		node.updateResources(zosResources)
		assert.True(t, node.Resources.Used.isEmpty())
		assert.True(t, node.isUnused())
		assert.Equal(t, node.Resources.OverProvisionCPU, float32(1))
		assert.True(t, node.canClaimResources(node.Resources.Total))

		node.claimResources(node.Resources.Total)
		assert.False(t, node.Resources.Used.isEmpty())
		assert.False(t, node.isUnused())
		assert.False(t, node.canClaimResources(node.Resources.Total))

		node.Resources.Used = capacity{}
	})
}

func TestPowerModel(t *testing.T) {
	power := Power{
		WakeUpThreshold:     80,
		PeriodicWakeUpStart: WakeUpDate(time.Now()),
	}
	oldPower := time.Time(power.PeriodicWakeUpStart)

	// invalid json
	err := power.PeriodicWakeUpStart.UnmarshalJSON([]byte("7:3"))
	assert.Error(t, err)

	// invalid yaml
	err = power.PeriodicWakeUpStart.UnmarshalYAML("7:3")
	assert.Error(t, err)

	// invalid toml
	err = power.PeriodicWakeUpStart.UnmarshalText([]byte("7:3"))
	assert.Error(t, err)

	// valid json
	wakeUpBytesJson, err := power.PeriodicWakeUpStart.MarshalJSON()
	assert.NoError(t, err)

	err = power.PeriodicWakeUpStart.UnmarshalJSON(wakeUpBytesJson)
	assert.NoError(t, err)

	// valid yaml
	wakeUpBytesYaml, err := power.PeriodicWakeUpStart.MarshalYAML()
	assert.NoError(t, err)

	err = power.PeriodicWakeUpStart.UnmarshalYAML(string(wakeUpBytesYaml))
	assert.NoError(t, err)

	// valid toml
	wakeUpBytesToml, err := power.PeriodicWakeUpStart.MarshalText()
	assert.NoError(t, err)

	err = power.PeriodicWakeUpStart.UnmarshalText(wakeUpBytesToml)
	assert.NoError(t, err)

	assert.Equal(t, wakeUpBytesToml, wakeUpBytesJson)
	assert.Equal(t, wakeUpBytesYaml, wakeUpBytesJson)

	assert.Equal(t, time.Time(power.PeriodicWakeUpStart).Hour(), oldPower.Hour())
	assert.Equal(t, time.Time(power.PeriodicWakeUpStart).Minute(), oldPower.Minute())
	assert.NotEqual(t, time.Time(power.PeriodicWakeUpStart).Day(), oldPower.Day())

	power.PeriodicWakeUpStart = WakeUpDate(power.PeriodicWakeUpStart.PeriodicWakeUpTime())
	assert.Equal(t, time.Time(power.PeriodicWakeUpStart).Day(), oldPower.Day())
}
