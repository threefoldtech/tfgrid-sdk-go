package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPowerModel(t *testing.T) {
	power := Power{
		WakeUpThreshold:     80,
		PeriodicWakeUpStart: WakeUpDate(time.Now()),
	}
	oldPower := time.Time(power.PeriodicWakeUpStart)

	// invalid
	err := power.PeriodicWakeUpStart.UnmarshalJSON([]byte("7:3"))
	assert.Error(t, err)

	// valid
	wakeUpBytes, err := power.PeriodicWakeUpStart.MarshalJSON()
	assert.NoError(t, err)

	err = power.PeriodicWakeUpStart.UnmarshalJSON(wakeUpBytes)
	assert.NoError(t, err)

	assert.Equal(t, time.Time(power.PeriodicWakeUpStart).Hour(), oldPower.Hour())
	assert.Equal(t, time.Time(power.PeriodicWakeUpStart).Minute(), oldPower.Minute())
	assert.NotEqual(t, time.Time(power.PeriodicWakeUpStart).Day(), oldPower.Day())

	power.PeriodicWakeUpStart = WakeUpDate(power.PeriodicWakeUpStart.PeriodicWakeUpTime())
	assert.Equal(t, time.Time(power.PeriodicWakeUpStart).Day(), oldPower.Day())
}
