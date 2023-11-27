// Package parser for parsing cmd configs
package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/constants"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/models"
	"gopkg.in/yaml.v3"
)

// ReadFile reads a file and returns its contents
func ReadFile(path string) ([]byte, string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return []byte{}, "", err
	}

	return content, filepath.Ext(path)[1:], nil
}

// ParseIntoConfig parses the configuration
func ParseIntoConfig(content []byte, format string) (*models.Config, error) {
	c := models.Config{}

	var err error
	switch {
	case strings.ToLower(format) == "json":
		err = json.Unmarshal(content, &c)
	case strings.ToLower(format) == "yml" || strings.ToLower(format) == "yaml":
		err = yaml.Unmarshal(content, &c)
	case strings.ToLower(format) == "toml":
		err = toml.Unmarshal(content, &c)
	default:
		err = fmt.Errorf("invalid config file format '%s'", format)
	}

	if err != nil {
		return nil, err
	}

	err = validate(&c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func validate(c *models.Config) error {
	// required values for farm
	if c.Farm.ID == 0 {
		return errors.New("farm ID is required")
	}
	log.Debug().Msgf("[FARMERBOT] define farm %d", c.Farm.ID)

	if len(c.Nodes) < 2 {
		return fmt.Errorf("configuration should contain at least 2 nodes, found %d. if more were configured make sure to check the configuration for mistakes", len(c.Nodes))
	}

	// required values for node
	for i, n := range c.Nodes {
		if n.ID == 0 {
			return fmt.Errorf("node id with index %d is required", i)
		}
		if n.TwinID == 0 {
			return fmt.Errorf("node %d: twin_id is required", n.ID)
		}
		if n.Resources.Total.SRU == 0 {
			return fmt.Errorf("node %d: total SRU is required", n.ID)
		}
		if n.Resources.Total.CRU == 0 {
			return fmt.Errorf("node %d: total CRU is required", n.ID)
		}
		if n.Resources.Total.MRU == 0 {
			return fmt.Errorf("node %d: total MRU is required", n.ID)
		}
		if n.Resources.Total.HRU == 0 {
			return fmt.Errorf("node %d: total HRU is required", n.ID)
		}

		if n.Resources.OverProvisionCPU == 0 {
			c.Nodes[i].Resources.OverProvisionCPU = constants.DefaultCPUProvision
			n = c.Nodes[i]
		}

		if n.Resources.OverProvisionCPU < 1 || n.Resources.OverProvisionCPU > 4 {
			return fmt.Errorf("node id %d: cpu over provision should be a value between 1 and 4 not %v", n.ID, n.Resources.OverProvisionCPU)
		}

		log.Debug().Msgf("[FARMERBOT] define node %d", n.ID)
	}

	// required values for power
	if c.Power.WakeUpThreshold == 0 {
		c.Power.WakeUpThreshold = constants.DefaultWakeUpThreshold
	}

	if c.Power.WakeUpThreshold < constants.MinWakeUpThreshold {
		c.Power.WakeUpThreshold = constants.MinWakeUpThreshold
		log.Warn().Msgf("[FARMERBOT] The setting wake_up_threshold should be in the range [%d, %d]", constants.MinWakeUpThreshold, constants.MaxWakeUpThreshold)
	}

	if c.Power.WakeUpThreshold > constants.MaxWakeUpThreshold {
		c.Power.WakeUpThreshold = constants.MaxWakeUpThreshold
		log.Warn().Msgf("[FARMERBOT] The setting wake_up_threshold should be in the range [%d, %d]", constants.MinWakeUpThreshold, constants.MaxWakeUpThreshold)
	}

	if c.Power.PeriodicWakeUpStart.PeriodicWakeUpTime().Hour() == 0 && c.Power.PeriodicWakeUpStart.PeriodicWakeUpTime().Minute() == 0 {
		c.Power.PeriodicWakeUpStart = models.WakeUpDate(time.Now())
		log.Warn().Msgf("[FARMERBOT] The setting periodic_wake_up_start is zero. It is set with current time '%v'", c.Power.PeriodicWakeUpStart)
	}
	c.Power.PeriodicWakeUpStart = models.WakeUpDate(c.Power.PeriodicWakeUpStart.PeriodicWakeUpTime())

	if c.Power.PeriodicWakeUpLimit == 0 {
		c.Power.PeriodicWakeUpLimit = constants.DefaultPeriodicWakeUPLimit
		log.Warn().Msg("[FARMERBOT] The setting periodic_wake_up_limit should be greater then 0!")
	}
	log.Debug().Msg("[FARMERBOT] configure power")

	return nil
}
