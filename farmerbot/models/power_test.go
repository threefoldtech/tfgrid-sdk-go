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
