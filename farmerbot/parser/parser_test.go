package parser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFileReader(t *testing.T) {
	t.Run("test invalid file", func(t *testing.T) {
		_, err := ReadFile("test.yml")
		assert.Error(t, err)
	})

	t.Run("test valid file", func(t *testing.T) {
		_, err := ReadFile("parser.go")
		assert.NoError(t, err)
	})
}

func TestYAMLParsers(t *testing.T) {
	t.Run("test invalid yaml", func(t *testing.T) {
		content := `[]`

		_, err := ParseIntoConfig([]byte(content))
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
never_shutdown_nodes:
  - 2
power:
  periodic_wake_up_start: 08:30AM
  wake_up_threshold: 30
  periodic_wake_up_limit: 2
  overprovision_cpu: 2`

		c, err := ParseIntoConfig([]byte(content))
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
