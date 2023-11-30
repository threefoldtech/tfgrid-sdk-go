package models

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/constants"
)

// InputConfig is the inputs for configuration for farmerbot
type InputConfig struct {
	FarmID        uint32   `json:"farm_id" yaml:"farm_id" toml:"farm_id"`
	IncludedNodes []uint32 `json:"included_nodes" yaml:"included_nodes" toml:"included_nodes"`
	ExcludedNodes []uint32 `json:"excluded_nodes" yaml:"excluded_nodes" toml:"excluded_nodes"`
	Power         Power    `json:"power" yaml:"power" toml:"power"`
}

// Config is the configuration data for farmerbot
type Config struct {
	Farm  substrate.Farm `json:"farm" yaml:"farm" toml:"farm"`
	Nodes []Node         `json:"nodes" yaml:"nodes" toml:"nodes"`
	Power Power          `json:"power" yaml:"power" toml:"power"`
	m     sync.Mutex
}

// Set sets the config data from configuration inputs
func (c *Config) Set(sub Sub, inputs InputConfig) error {
	log.Debug().Msg("Set power")

	if !reflect.DeepEqual(c.Power, inputs.Power) {
		c.Power = inputs.Power
	} else {
		log.Debug().Msg("Configuration power didn't change")
	}

	// required from power for nodes
	if c.Power.OverProvisionCPU == 0 {
		c.Power.OverProvisionCPU = constants.DefaultCPUProvision
	}

	if c.Power.OverProvisionCPU < 1 || c.Power.OverProvisionCPU > 4 {
		return fmt.Errorf("cpu over provision should be a value between 1 and 4 not %v", c.Power.OverProvisionCPU)
	}

	// set farm
	log.Debug().Uint32("ID", inputs.FarmID).Msg("Set farm")
	if inputs.FarmID != uint32(c.Farm.ID) {
		farm, err := sub.GetFarm(inputs.FarmID)
		if err != nil {
			return err
		}

		c.Farm = *farm
	} else {
		log.Debug().Msg("Configuration farm didn't change")
	}

	// set nodes
	err := c.SetConfigNodes(sub, inputs)
	if err != nil {
		return err
	}

	return c.validate()
}

// SetConfigNodes sets the config nodes from configuration inputs
func (c *Config) SetConfigNodes(sub Sub, i InputConfig) error {
	farmNodes, err := sub.GetNodes(i.FarmID)
	if err != nil {
		return err
	}

	for _, nodeID := range farmNodes {
		if slices.Contains(i.ExcludedNodes, nodeID) {
			continue
		}

		// if the user specified included nodes or
		// no nodes are specified so all nodes will be added (except excluded)
		if slices.Contains(i.IncludedNodes, nodeID) || len(i.IncludedNodes) == 0 {
			log.Debug().Uint32("ID", nodeID).Msg("Set node")
			node, err := sub.GetNode(nodeID)
			if err != nil {
				return err
			}

			configNode := Node{
				Node: *node,
			}

			price, err := sub.GetDedicatedNodePrice(nodeID)
			if err != nil {
				return err
			}

			if price != 0 || c.Farm.DedicatedFarm {
				configNode.Dedicated = true
			}

			configNode.Resources.Total.CRU = uint64(node.Resources.CRU)
			configNode.Resources.Total.SRU = uint64(node.Resources.SRU)
			configNode.Resources.Total.MRU = uint64(node.Resources.MRU)
			configNode.Resources.Total.HRU = uint64(node.Resources.HRU)
			configNode.Resources.OverProvisionCPU = c.Power.OverProvisionCPU

			c.Nodes = append(c.Nodes, configNode)
		}
	}

	return nil
}

// GetNodeByNodeID gets a node by id
func (c *Config) GetNodeByNodeID(nodeID uint32) (Node, error) {
	for _, n := range c.Nodes {
		if uint32(n.ID) == nodeID {
			return n, nil
		}
	}

	return Node{}, fmt.Errorf("node %d not found", nodeID)
}

// UpdateNode updates a node in the config
func (c *Config) UpdateNode(node Node) error {
	c.m.Lock()
	defer c.m.Unlock()

	for i, n := range c.Nodes {
		if n.ID == node.ID {
			c.Nodes[i] = node
			return nil
		}
	}

	return fmt.Errorf("node %d not found", node.ID)
}

// FilterNodesPower filters ON or OFF nodes
func (c *Config) FilterNodesPower(states []PowerState) []Node {
	filtered := make([]Node, 0)
	for _, node := range c.Nodes {
		if slices.Contains(states, node.PowerState) {
			filtered = append(filtered, node)
		}
	}
	return filtered
}

// FilterAllowedNodesToShutDown filters nodes that are allowed to shut down
//
// nodes with public config can't be shutdown
// Do not shutdown a node that just came up (give it some time)
func (c *Config) FilterAllowedNodesToShutDown() []Node {
	filtered := make([]Node, 0)
	for _, node := range c.Nodes {
		if node.IsUnused() && !node.PublicConfig.HasValue && !node.NeverShutDown &&
			time.Since(node.LastTimePowerStateChanged) >= constants.PeriodicWakeUpDuration {
			filtered = append(filtered, node)
		}
	}
	return filtered
}

func (c *Config) validate() error {
	// required values for farm
	if c.Farm.ID == 0 {
		return errors.New("farm ID is required")
	}

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
	}

	// required values for power
	if c.Power.WakeUpThreshold == 0 {
		c.Power.WakeUpThreshold = constants.DefaultWakeUpThreshold
	}

	if c.Power.WakeUpThreshold < constants.MinWakeUpThreshold {
		c.Power.WakeUpThreshold = constants.MinWakeUpThreshold
		log.Warn().Msgf("The setting wake_up_threshold should be in the range [%d, %d], setting it to minimum value %d", constants.MinWakeUpThreshold, constants.MaxWakeUpThreshold, constants.MinWakeUpThreshold)
	}

	if c.Power.WakeUpThreshold > constants.MaxWakeUpThreshold {
		c.Power.WakeUpThreshold = constants.MaxWakeUpThreshold
		log.Warn().Msgf("The setting wake_up_threshold should be in the range [%d, %d], setting it to maximum value %d", constants.MinWakeUpThreshold, constants.MaxWakeUpThreshold, constants.MinWakeUpThreshold)
	}

	if c.Power.PeriodicWakeUpStart.PeriodicWakeUpTime().Hour() == 0 && c.Power.PeriodicWakeUpStart.PeriodicWakeUpTime().Minute() == 0 {
		c.Power.PeriodicWakeUpStart = WakeUpDate(time.Now())
		log.Warn().Time("periodic wakeup start", c.Power.PeriodicWakeUpStart.PeriodicWakeUpTime()).Msg("The setting periodic_wake_up_start is zero. setting it with current time")
	}
	c.Power.PeriodicWakeUpStart = WakeUpDate(c.Power.PeriodicWakeUpStart.PeriodicWakeUpTime())

	if c.Power.PeriodicWakeUpLimit == 0 {
		c.Power.PeriodicWakeUpLimit = constants.DefaultPeriodicWakeUPLimit
		log.Warn().Msgf("The setting periodic_wake_up_limit should be greater then 0! setting it to %d", constants.DefaultPeriodicWakeUPLimit)
	}

	return nil
}
