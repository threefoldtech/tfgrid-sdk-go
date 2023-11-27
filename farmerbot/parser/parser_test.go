// Package parser for parsing cmd configs
package parser

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/constants"
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
		content := `key:`

		_, err := ParseIntoConfig([]byte(content), "yaml")
		assert.Error(t, err)
	})

	t.Run("test valid yaml", func(t *testing.T) {
		content := `
farm:
  id: 1
nodes:
  - id: 1
    twin_id: 1
    resources:
      total:
        sru: 1
        cru: 1
        hru: 1
        mru: 1
  - id: 2
    twin_id: 2
    resources:
      total:
        sru: 1
        cru: 1
        hru: 1
        mru: 1
power:
  periodic_wake_up_start: 08:30AM
  wake_up_threshold: 90`

		c, err := ParseIntoConfig([]byte(content), "yaml")
		assert.NoError(t, err)
		assert.Equal(t, c.Farm.ID, uint32(1))
		assert.Equal(t, c.Nodes[0].Resources.OverProvisionCPU, constants.DefaultCPUProvision)
		assert.Equal(t, c.Nodes[0].ID, uint32(1))
		assert.Equal(t, c.Power.WakeUpThreshold, constants.MaxWakeUpThreshold)

		now := time.Now()
		assert.Equal(t, c.Power.PeriodicWakeUpStart.PeriodicWakeUpTime(), time.Date(now.Year(), now.Month(), now.Day(), 8, 30, 0, 0, time.Local))
	})
}

func TestTOMLParsers(t *testing.T) {
	t.Run("test invalid toml", func(t *testing.T) {
		content := `key:`

		_, err := ParseIntoConfig([]byte(content), "toml")
		assert.Error(t, err)
	})

	t.Run("test valid toml", func(t *testing.T) {
		content := `
[farm]
id = 1

[[nodes]]
id = 1
twin_id = 1
[nodes.resources]
total.sru = 1
total.cru = 1
total.hru = 1
total.mru = 1

[[nodes]]
id = 2
twin_id = 2
[nodes.resources]
total.sru = 1
total.cru = 1
total.hru = 1
total.mru = 1

[power]
periodic_wake_up_start = "08:30AM"
wake_up_threshold = 0`

		c, err := ParseIntoConfig([]byte(content), "toml")
		assert.NoError(t, err)
		assert.Equal(t, c.Farm.ID, uint32(1))
		assert.Equal(t, c.Nodes[0].Resources.OverProvisionCPU, constants.DefaultCPUProvision)
		assert.Equal(t, c.Nodes[0].ID, uint32(1))
		assert.Equal(t, c.Power.WakeUpThreshold, constants.DefaultWakeUpThreshold)

		now := time.Now()
		assert.Equal(t, c.Power.PeriodicWakeUpStart.PeriodicWakeUpTime(), time.Date(now.Year(), now.Month(), now.Day(), 8, 30, 0, 0, time.Local))
	})
}

func TestJsonParsers(t *testing.T) {
	farmContent := `{ "ID": 1 }`

	t.Run("test invalid format", func(t *testing.T) {
		_, err := ParseIntoConfig([]byte(""), "go")
		assert.Error(t, err)
	})

	t.Run("test invalid json", func(t *testing.T) {
		_, err := ParseIntoConfig([]byte(`{"power": ,}`), "json")
		assert.Error(t, err)
	})

	t.Run("test valid json", func(t *testing.T) {
		nodeContent := `{ "ID": 1, "twin_id" : 1, "resources": { "total": { "SRU": 1, "CRU": 1, "HRU": 1, "MRU": 1 } } }`
		nodeContent2 := `{ "ID": 2, "twin_id" : 2, "resources": { "total": { "SRU": 1, "CRU": 1, "HRU": 1, "MRU": 1 } } }`
		powerContent := `{ "periodic_wake_up_start": "08:30AM", "wake_up_threshold": 30 }`
		content := fmt.Sprintf(`
		{ 
			"nodes": [ %v,%v ],
			"farm": %v, 
			"power": %v
		}
		`, nodeContent, nodeContent2, farmContent, powerContent)

		c, err := ParseIntoConfig([]byte(content), "json")
		assert.NoError(t, err)
		assert.Equal(t, c.Farm.ID, uint32(1))
		assert.Equal(t, c.Nodes[0].Resources.OverProvisionCPU, constants.DefaultCPUProvision)
		assert.Equal(t, c.Nodes[0].ID, uint32(1))
		assert.Equal(t, c.Power.WakeUpThreshold, constants.MinWakeUpThreshold)

		now := time.Now()
		assert.Equal(t, c.Power.PeriodicWakeUpStart.PeriodicWakeUpTime(), time.Date(now.Year(), now.Month(), now.Day(), 8, 30, 0, 0, time.Local))
		assert.Equal(t, c.Power.PeriodicWakeUpLimit, constants.DefaultPeriodicWakeUPLimit)
	})

	t.Run("test valid json: no periodic wake up start", func(t *testing.T) {
		nodeContent := `{ "ID": 1, "twin_id" : 1, "resources": { "total": { "SRU": 1, "CRU": 1, "HRU": 1, "MRU": 1 } } }`
		nodeContent2 := `{ "ID": 2, "twin_id" : 2, "resources": { "total": { "SRU": 1, "CRU": 1, "HRU": 1, "MRU": 1 } } }`
		powerContent := `{ "wake_up_threshold": 30 }`
		content := fmt.Sprintf(`
		{ 
			"nodes": [ %v,%v ],
			"farm": %v, 
			"power": %v
		}
		`, nodeContent, nodeContent2, farmContent, powerContent)

		c, err := ParseIntoConfig([]byte(content), "json")
		assert.NoError(t, err)
		assert.Equal(t, c.Farm.ID, uint32(1))
		assert.Equal(t, c.Nodes[0].Resources.OverProvisionCPU, constants.DefaultCPUProvision)
		assert.Equal(t, c.Nodes[0].ID, uint32(1))
		assert.Equal(t, c.Power.WakeUpThreshold, constants.MinWakeUpThreshold)

		now := time.Now()
		assert.Equal(t, c.Power.PeriodicWakeUpStart.PeriodicWakeUpTime().Hour(), now.Hour())
		assert.Equal(t, c.Power.PeriodicWakeUpStart.PeriodicWakeUpTime().Minute(), now.Minute())
		assert.Equal(t, c.Power.PeriodicWakeUpLimit, constants.DefaultPeriodicWakeUPLimit)
	})

	t.Run("test invalid json < 2 nodes are provided", func(t *testing.T) {
		content := fmt.Sprintf(`
		{ 
			"nodes": [ {} ],
			"farm": %v, 
			"power": {}
		}
		`, farmContent)

		_, err := ParseIntoConfig([]byte(content), "json")
		assert.Error(t, err)
	})

	t.Run("test invalid json no node ID", func(t *testing.T) {
		content := fmt.Sprintf(`
		{ 
			"nodes": [ {}, {} ],
			"farm": %v, 
			"power": {}
		}
		`, farmContent)

		_, err := ParseIntoConfig([]byte(content), "json")
		assert.Error(t, err)
	})

	t.Run("test invalid json no node twin ID", func(t *testing.T) {
		nodeContent := `{ "ID": 1 }`
		content := fmt.Sprintf(`
		{ 
			"nodes": [ %v, {} ],
			"farm": %v, 
			"power": {}
		}
		`, nodeContent, farmContent)

		_, err := ParseIntoConfig([]byte(content), "json")
		assert.Error(t, err)
	})

	t.Run("test invalid json no node sru", func(t *testing.T) {
		nodeContent := `{ "ID": 1, "twin_id" : 1 }`
		content := fmt.Sprintf(`
		{ 
			"nodes": [ %v, {} ],
			"farm": %v, 
			"power": {}
		}
		`, nodeContent, farmContent)

		_, err := ParseIntoConfig([]byte(content), "json")
		assert.Error(t, err)
	})

	t.Run("test invalid json no cru", func(t *testing.T) {
		nodeContent := `{ "ID": 1, "twin_id" : 1, "resources": { "total": { "SRU": 1 } } }`
		content := fmt.Sprintf(`
		{ 
			"nodes": [ %v, {} ],
			"farm": %v, 
			"power": {}
		}
		`, nodeContent, farmContent)

		_, err := ParseIntoConfig([]byte(content), "json")
		assert.Error(t, err)
	})

	t.Run("test invalid json no mru", func(t *testing.T) {
		nodeContent := `{ "ID": 1, "twin_id" : 1, "resources": { "total": { "SRU": 1, "CRU": 1 } } }`
		content := fmt.Sprintf(`
		{ 
			"nodes": [ %v, {} ],
			"farm": %v, 
			"power": {}
		}
		`, nodeContent, farmContent)

		_, err := ParseIntoConfig([]byte(content), "json")
		assert.Error(t, err)
	})

	t.Run("test invalid json no hru", func(t *testing.T) {
		nodeContent := `{ "ID": 1, "twin_id" : 1, "resources": { "total": { "SRU": 1, "CRU": 1, "MRU": 1 } } }`
		content := fmt.Sprintf(`
		{ 
			"nodes": [ %v, {} ],
			"farm": %v, 
			"power": {}
		}
		`, nodeContent, farmContent)

		_, err := ParseIntoConfig([]byte(content), "json")
		assert.Error(t, err)
	})

	t.Run("test invalid json node over provision CPU", func(t *testing.T) {
		nodeContent := `{ "ID": 1, "twin_id" : 1, "resources": { "overprovision_cpu": 5, "total": { "SRU": 1, "CRU": 1, "HRU": 1, "MRU": 1 } } }`
		content := fmt.Sprintf(`
		{ 
			"nodes": [ %v, {} ],
			"farm": %v, 
			"power": {}
		}
		`, nodeContent, farmContent)

		_, err := ParseIntoConfig([]byte(content), "json")
		assert.Error(t, err)
	})

	t.Run("test invalid json no farm ID", func(t *testing.T) {
		content := `
		{ 
			"nodes": [ ],
			"farm": {}, 
			"power": {}
		}`

		_, err := ParseIntoConfig([]byte(content), "json")
		assert.Error(t, err)
	})
}
