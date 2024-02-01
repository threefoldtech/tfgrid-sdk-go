package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	zos "github.com/threefoldtech/zos/client"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

var cap = capacity{
	cru: 1,
	sru: 1,
	mru: 1,
	hru: 1,
}

func TestCapacityModel(t *testing.T) {
	assert.False(t, cap.isEmpty())

	resultSub := cap.subtract(cap)
	assert.True(t, resultSub.isEmpty())

	cap.add(cap)
	assert.Equal(t, cap.cru, uint64(2))
}

func TestNodeModel(t *testing.T) {
	node := node{
		Node: substrate.Node{ID: 1, TwinID: 1},
		resources: consumableResources{
			total: cap,
		},
		powerState: on,
	}

	t.Run("test update node resources", func(t *testing.T) {
		zosResources := zos.Counters{
			Total: gridtypes.Capacity{
				CRU:   cap.cru,
				SRU:   gridtypes.Unit(cap.sru),
				HRU:   gridtypes.Unit(cap.hru),
				MRU:   gridtypes.Unit(cap.mru),
				IPV4U: 1,
			},
			Used:   gridtypes.Capacity{},
			System: gridtypes.Capacity{},
		}

		node.updateResources(zosResources)
		assert.True(t, node.resources.used.isEmpty())
		assert.True(t, node.isUnused())
		assert.True(t, node.canClaimResources(node.resources.total, 1))

		node.claimResources(node.resources.total)
		assert.False(t, node.resources.used.isEmpty())
		assert.False(t, node.isUnused())
		assert.False(t, node.canClaimResources(node.resources.total, 1))

		node.resources.used = capacity{}
	})
}

func TestPowerModel(t *testing.T) {
	power := power{
		WakeUpThreshold:     80,
		PeriodicWakeUpStart: wakeUpDate(time.Now()),
	}
	oldPower := time.Time(power.PeriodicWakeUpStart)

	// invalid date
	err := power.PeriodicWakeUpStart.UnmarshalText([]byte("7:3"))
	assert.Error(t, err)

	// valid date
	wakeUpBytes, err := power.PeriodicWakeUpStart.MarshalText()
	assert.NoError(t, err)

	err = power.PeriodicWakeUpStart.UnmarshalText(wakeUpBytes)
	assert.NoError(t, err)

	assert.Equal(t, time.Time(power.PeriodicWakeUpStart).Hour(), oldPower.Hour())
	assert.Equal(t, time.Time(power.PeriodicWakeUpStart).Minute(), oldPower.Minute())
	assert.Equal(t, time.Time(power.PeriodicWakeUpStart).Day(), oldPower.Day())

	power.PeriodicWakeUpStart = wakeUpDate(power.PeriodicWakeUpStart.PeriodicWakeUpTime())
	assert.Equal(t, time.Time(power.PeriodicWakeUpStart).Day(), oldPower.Day())
}
