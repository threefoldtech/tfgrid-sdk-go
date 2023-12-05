package internal

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/constants"
)

// state is the state data for farmerbot
type state struct {
	farm  substrate.Farm
	nodes map[uint32]node
	power Power
	m     sync.Mutex
}

// NewState creates new state from configs
func newState(sub Sub, inputs Config) (*state, error) {
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
	nodes, err := convertInputsToNodes(sub, inputs, farm.DedicatedFarm, overProvisionCPU)
	if err != nil {
		return nil, err
	}

	config := state{
		farm:  *farm,
		nodes: nodes,
		power: inputs.Power,
	}

	err = config.validate()
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// ConvertInputsToNodes converts the config nodes from configuration inputs
func convertInputsToNodes(sub Sub, i Config, dedicatedFarm bool, overProvisionCPU float32) (map[uint32]node, error) {
	nodes := make(map[uint32]node)

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
			nodeObj, err := sub.GetNode(nodeID)
			if err != nil {
				return nil, err
			}

			configNode := node{
				Node: *nodeObj,
			}

			price, err := sub.GetDedicatedNodePrice(nodeID)
			if err != nil {
				return nil, err
			}

			if price != 0 || dedicatedFarm {
				configNode.Dedicated = true
			}

			configNode.Resources.Total.CRU = uint64(nodeObj.Resources.CRU)
			configNode.Resources.Total.SRU = uint64(nodeObj.Resources.SRU)
			configNode.Resources.Total.MRU = uint64(nodeObj.Resources.MRU)
			configNode.Resources.Total.HRU = uint64(nodeObj.Resources.HRU)
			configNode.Resources.OverProvisionCPU = overProvisionCPU

			nodes[nodeID] = configNode
		}
	}

	return nodes, nil
}

// UpdateNode updates a node in the config
func (s *state) updateNode(node node) error {
	s.m.Lock()
	defer s.m.Unlock()

	_, ok := s.nodes[uint32(node.ID)]
	if !ok {
		return fmt.Errorf("node %d is not found", uint32(node.ID))
	}

	s.nodes[uint32(node.ID)] = node

	return nil
}

// FilterNodesPower filters ON or OFF nodes
func (s *state) filterNodesPower(states []PowerState) []node {
	filtered := make([]node, 0)
	for _, node := range s.nodes {
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
func (s *state) filterAllowedNodesToShutDown() []node {
	filtered := make([]node, 0)
	for _, node := range s.nodes {
		if node.isUnused() && !node.PublicConfig.HasValue && !node.NeverShutDown &&
			time.Since(node.LastTimePowerStateChanged) >= constants.PeriodicWakeUpDuration {
			filtered = append(filtered, node)
		}
	}
	return filtered
}

func (s *state) validate() error {
	// required values for farm
	if s.farm.ID == 0 {
		return errors.New("farm ID is required")
	}

	if len(s.nodes) < 2 {
		return fmt.Errorf("configuration should contain at least 2 nodes, found %d. if more were configured make sure to check the configuration for mistakes", len(s.nodes))
	}

	// required values for node
	for _, n := range s.nodes {
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
	if s.power.WakeUpThreshold == 0 {
		s.power.WakeUpThreshold = constants.DefaultWakeUpThreshold
	}

	if s.power.WakeUpThreshold < constants.MinWakeUpThreshold {
		s.power.WakeUpThreshold = constants.MinWakeUpThreshold
		log.Warn().Msgf("The setting wake_up_threshold should be in the range [%d, %d], setting it to minimum value %d", constants.MinWakeUpThreshold, constants.MaxWakeUpThreshold, constants.MinWakeUpThreshold)
	}

	if s.power.WakeUpThreshold > constants.MaxWakeUpThreshold {
		s.power.WakeUpThreshold = constants.MaxWakeUpThreshold
		log.Warn().Msgf("The setting wake_up_threshold should be in the range [%d, %d], setting it to maximum value %d", constants.MinWakeUpThreshold, constants.MaxWakeUpThreshold, constants.MinWakeUpThreshold)
	}

	if s.power.PeriodicWakeUpStart.PeriodicWakeUpTime().Hour() == 0 && s.power.PeriodicWakeUpStart.PeriodicWakeUpTime().Minute() == 0 {
		s.power.PeriodicWakeUpStart = WakeUpDate(time.Now())
		log.Warn().Time("periodic wakeup start", s.power.PeriodicWakeUpStart.PeriodicWakeUpTime()).Msg("The setting periodic_wake_up_start is zero. setting it with current time")
	}
	s.power.PeriodicWakeUpStart = WakeUpDate(s.power.PeriodicWakeUpStart.PeriodicWakeUpTime())

	if s.power.PeriodicWakeUpLimit == 0 {
		s.power.PeriodicWakeUpLimit = constants.DefaultPeriodicWakeUPLimit
		log.Warn().Msgf("The setting periodic_wake_up_limit should be greater then 0! setting it to %d", constants.DefaultPeriodicWakeUPLimit)
	}

	return nil
}
