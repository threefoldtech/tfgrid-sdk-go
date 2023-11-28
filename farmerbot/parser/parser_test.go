// Package parser for parsing cmd configs
package parser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFileReader(t *testing.T) {
	t.Run("test invalid file", func(t *testing.T) {
		_, format, err := ReadFile("json.json")
		assert.Empty(t, format)
		assert.Error(t, err)
	})

	t.Run("test valid file", func(t *testing.T) {
		_, format, err := ReadFile("parser.go")
		assert.Equal(t, format, "go")
		assert.NoError(t, err)
	})
}

func TestYAMLParsers(t *testing.T) {
	t.Run("test invalid yaml", func(t *testing.T) {
		content := `[]`

		_, err := ParseIntoInputConfig([]byte(content), "yaml")
		assert.Error(t, err)
	})

	t.Run("test valid yaml", func(t *testing.T) {
		content := `
farm_id: 1
included_nodes:
  - 1
  - 2
excluded_nodes:
  - 3
power:
  periodic_wake_up_start: 08:30AM
  wake_up_threshold: 30
  periodic_wake_up_limit: 2
  overprovision_cpu: 2`

		c, err := ParseIntoInputConfig([]byte(content), "yaml")
		assert.NoError(t, err)
		assert.Equal(t, c.FarmID, uint32(1))
		assert.Equal(t, len(c.IncludedNodes), 2)
		assert.Equal(t, len(c.ExcludedNodes), 1)
		assert.Equal(t, c.Power.WakeUpThreshold, uint8(30))

		now := time.Now()
		assert.Equal(t, c.Power.PeriodicWakeUpStart.PeriodicWakeUpTime(), time.Date(now.Year(), now.Month(), now.Day(), 8, 30, 0, 0, time.Local))
		assert.Equal(t, c.Power.PeriodicWakeUpLimit, uint8(2))
		assert.Equal(t, c.Power.OverProvisionCPU, float32(2))
	})
}

func TestTOMLParsers(t *testing.T) {
	t.Run("test invalid toml", func(t *testing.T) {
		content := `key:`

		_, err := ParseIntoInputConfig([]byte(content), "toml")
		assert.Error(t, err)
	})

	t.Run("test valid toml", func(t *testing.T) {
		content := `
farm_id = 1
included_nodes = [1, 2]
excluded_nodes = [3]

[power]
periodic_wake_up_start = "08:30AM"
wake_up_threshold = 30
periodic_wake_up_limit = 2
overprovision_cpu = 2`

		c, err := ParseIntoInputConfig([]byte(content), "toml")
		assert.NoError(t, err)
		assert.Equal(t, c.FarmID, uint32(1))
		assert.Equal(t, len(c.IncludedNodes), 2)
		assert.Equal(t, len(c.ExcludedNodes), 1)
		assert.Equal(t, c.Power.WakeUpThreshold, uint8(30))

		now := time.Now()
		assert.Equal(t, c.Power.PeriodicWakeUpStart.PeriodicWakeUpTime(), time.Date(now.Year(), now.Month(), now.Day(), 8, 30, 0, 0, time.Local))
		assert.Equal(t, c.Power.PeriodicWakeUpLimit, uint8(2))
		assert.Equal(t, c.Power.OverProvisionCPU, float32(2))
	})
}

func TestJsonParsers(t *testing.T) {
	t.Run("test invalid format", func(t *testing.T) {
		_, err := ParseIntoInputConfig([]byte(""), "go")
		assert.Error(t, err)
	})

	t.Run("test invalid json", func(t *testing.T) {
		_, err := ParseIntoInputConfig([]byte(`{"power": ,}`), "json")
		assert.Error(t, err)
	})

	t.Run("test valid json", func(t *testing.T) {
		content := `{
			"farm_id": 1, 
			"included_nodes": [ 1, 2 ],
			"excluded_nodes": [ 3 ],
			"power": { "periodic_wake_up_start": "08:30AM", "wake_up_threshold": 30, "periodic_wake_up_limit": 2, "overprovision_cpu": 2 }
		}`

		c, err := ParseIntoInputConfig([]byte(content), "json")
		assert.NoError(t, err)
		assert.Equal(t, c.FarmID, uint32(1))
		assert.Equal(t, len(c.IncludedNodes), 2)
		assert.Equal(t, len(c.ExcludedNodes), 1)
		assert.Equal(t, c.Power.WakeUpThreshold, uint8(30))

		now := time.Now()
		assert.Equal(t, c.Power.PeriodicWakeUpStart.PeriodicWakeUpTime(), time.Date(now.Year(), now.Month(), now.Day(), 8, 30, 0, 0, time.Local))
		assert.Equal(t, c.Power.PeriodicWakeUpLimit, uint8(2))
		assert.Equal(t, c.Power.OverProvisionCPU, float32(2))
	})

	t.Run("test invalid json", func(t *testing.T) {
		content := `{
			"farm_id": 1, 
			"included_nodes": [ 1, 2 ],
			"excluded_nodes": [ 2 ],
			"power": { "periodic_wake_up_start": "08:30AM", "wake_up_threshold": 30 }
		}`

		_, err := ParseIntoInputConfig([]byte(content), "json")
		assert.Error(t, err)
	})
}
