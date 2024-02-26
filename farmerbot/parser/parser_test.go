package parser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/internal"
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
  wake_up_threshold: 
    cru: 30 
    mru: 30 
    sru: 30 
    hru: 30
  periodic_wake_up_limit: 2
  overprovision_cpu: 2`

		c, err := ParseIntoConfig([]byte(content))
		assert.NoError(t, err)
		assert.Equal(t, c.FarmID, uint32(1))
		assert.Equal(t, c.IncludedNodes, []uint32{1, 2})
		assert.Equal(t, c.ExcludedNodes, []uint32{3})
		assert.Equal(t, c.NeverShutDownNodes, []uint32{2})
		assert.Equal(t, c.Power.WakeUpThresholdPercentages, internal.ThresholdPercentages{
			CRU: 30,
			SRU: 30,
			MRU: 30,
			HRU: 30,
		})

		now := time.Now()
		assert.Equal(t, c.Power.PeriodicWakeUpStart.PeriodicWakeUpTime(), time.Date(now.Year(), now.Month(), now.Day(), 8, 30, 0, 0, time.Local))
		assert.Equal(t, c.Power.PeriodicWakeUpLimit, uint8(2))
		assert.Equal(t, c.Power.OverProvisionCPU, int8(2))
	})

	t.Run("test invalid yaml: node is included and excluded", func(t *testing.T) {
		content := `
farm_id: 1
included_nodes:
  - 1
  - 2
excluded_nodes:
  - 2`

		_, err := ParseIntoConfig([]byte(content))
		assert.Error(t, err)
	})
}

func TestEnvParser(t *testing.T) {
	t.Run("test invalid env", func(t *testing.T) {
		content := `invalid`

		_, _, _, err := ParseEnv(content)
		assert.Error(t, err)
	})

	t.Run("test invalid env key", func(t *testing.T) {
		content := `invalid=invalid`

		_, _, _, err := ParseEnv(content)
		assert.Error(t, err)
	})

	t.Run("test valid env", func(t *testing.T) {
		content := `
NETWORK=dev
MNEMONIC_OR_SEED=0xe5be9a5092b81bca64be81d212e7f2f9eba183bb7a90954f7b76361f6edb5c0a
KEY_TYPE=ed25519`

		net, seed, keyType, err := ParseEnv(content)
		assert.NoError(t, err)
		assert.Equal(t, net, internal.DevNetwork)
		assert.Equal(t, seed, "0xe5be9a5092b81bca64be81d212e7f2f9eba183bb7a90954f7b76361f6edb5c0a")
		assert.Equal(t, keyType, "ed25519")
	})

	t.Run("test valid env: network is missing", func(t *testing.T) {
		content := `
NETWORK=
MNEMONIC_OR_SEED=0xe5be9a5092b81bca64be81d212e7f2f9eba183bb7a90954f7b76361f6edb5c0a`

		net, seed, keyType, err := ParseEnv(content)
		assert.NoError(t, err)
		assert.Equal(t, net, internal.MainNetwork)
		assert.Equal(t, seed, "0xe5be9a5092b81bca64be81d212e7f2f9eba183bb7a90954f7b76361f6edb5c0a")
		assert.Equal(t, keyType, "sr25519")
	})

	t.Run("test invalid env: network is invalid", func(t *testing.T) {
		content := `
NETWORK=qenet
MNEMONIC_OR_SEED=0xe5be9a5092b81bca64be81d212e7f2f9eba183bb7a90954f7b76361f6edb5c0a`

		_, _, _, err := ParseEnv(content)
		assert.Error(t, err)
	})

	t.Run("test invalid env: mnemonic is missing", func(t *testing.T) {
		content := `
NETWORK=
MNEMONIC_OR_SEED=`

		_, _, _, err := ParseEnv(content)
		assert.Error(t, err)
	})

	t.Run("test invalid env: mnemonic is invalid", func(t *testing.T) {
		content := `
NETWORK=
MNEMONIC_OR_SEED=//alice`

		_, _, _, err := ParseEnv(content)
		assert.Error(t, err)
	})
}
