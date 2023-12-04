package models

import (
	"errors"
	"fmt"
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
	Farm  substrate.Farm  `json:"farm" yaml:"farm" toml:"farm"`
	Nodes map[uint32]Node `json:"nodes" yaml:"nodes" toml:"nodes"`
	Power Power           `json:"power" yaml:"power" toml:"power"`
	m     sync.Mutex
}

// NewConfig creates new config from inputs
func NewConfig(sub Sub, inputs InputConfig) (*Config, error) {
	overProvisionCPU := inputs.Power.OverProvisionCPU
	// required from power for nodes
	if overProvisionCPU == 0 {
		overProvisionCPU = constants.DefaultCPUProvision
	}

	if overProvisionCPU < 1 || overProvisionCPU > 4 {
		return nil, fmt.Errorf("cpu over provision should be a value between 1 and 4 not %v", overProvisionCPU)
	}

	// set farm
	farm, err := sub.GetFarm(inputs.FarmID)
	if err != nil {
		return nil, err
	}

	// set nodes
	nodes, err := ConvertInputsToNodes(sub, inputs, farm.DedicatedFarm, overProvisionCPU)
	if err != nil {
		return nil, err
	}

	config := Config{
		Farm:  *farm,
		Nodes: nodes,
		Power: inputs.Power,
	}

	err = config.validate()
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// ConvertInputsToNodes converts the config nodes from configuration inputs
func ConvertInputsToNodes(sub Sub, i InputConfig, dedicatedFarm bool, overProvisionCPU float32) (map[uint32]Node, error) {
	nodes := make(map[uint32]Node)

	farmNodes, err := sub.GetNodes(i.FarmID)
	if err != nil {
		return nil, err
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
				return nil, err
			}

			configNode := Node{
				Node: *node,
			}

			price, err := sub.GetDedicatedNodePrice(nodeID)
			if err != nil {
				return nil, err
			}

			if price != 0 || dedicatedFarm {
				configNode.Dedicated = true
			}

			configNode.Resources.Total.CRU = uint64(node.Resources.CRU)
			configNode.Resources.Total.SRU = uint64(node.Resources.SRU)
			configNode.Resources.Total.MRU = uint64(node.Resources.MRU)
			configNode.Resources.Total.HRU = uint64(node.Resources.HRU)
			configNode.Resources.OverProvisionCPU = overProvisionCPU

			nodes[nodeID] = configNode
		}
	}

	return nodes, nil
}

// UpdateNode updates a node in the config
func (c *Config) UpdateNode(node Node) error {
	c.m.Lock()
	defer c.m.Unlock()

	_, ok := c.Nodes[uint32(node.ID)]
	if !ok {
		return fmt.Errorf("node %d is not found", uint32(node.ID))
	}

	c.Nodes[uint32(node.ID)] = node

	return nil
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
	for _, n := range c.Nodes {
		if n.ID == 0 {
			return fmt.Errorf("node id %d is required", n.ID)
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
