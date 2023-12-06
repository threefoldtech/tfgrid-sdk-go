package internal

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
)

// state is the state data for farmerbot
type state struct {
	farm   substrate.Farm
	nodes  map[uint32]node
	config Config
	m      sync.Mutex
}

// NewState creates new state from configs
func newState(ctx context.Context, sub Sub, rmbNodeClient RMB, inputs Config) (*state, error) {
	overProvisionCPU := inputs.Power.OverProvisionCPU
	// required from power for nodes
	if overProvisionCPU == 0 {
		overProvisionCPU = defaultCPUProvision
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
	nodes, err := convertInputsToNodes(ctx, sub, rmbNodeClient, inputs, farm.DedicatedFarm)
	if err != nil {
		return nil, err
	}

	config := state{
		farm:   *farm,
		nodes:  nodes,
		config: inputs,
	}

	err = config.validate()
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// ConvertInputsToNodes converts the config nodes from configuration inputs
func convertInputsToNodes(ctx context.Context, sub Sub, rmbNodeClient RMB, config Config, dedicatedFarm bool) (map[uint32]node, error) {
	nodes := make(map[uint32]node)

	farmNodes, err := sub.GetNodes(config.FarmID)
	if err != nil {
		return nil, err
	}

	for _, nodeID := range farmNodes {
		if slices.Contains(config.ExcludedNodes, nodeID) {
			continue
		}

		// if the user specified included nodes or
		// no nodes are specified so all nodes will be added (except excluded)
		if slices.Contains(config.IncludedNodes, nodeID) || len(config.IncludedNodes) == 0 {
			neverShutDown := slices.Contains(config.NeverShutDownNodes, nodeID)

			configNode, err := getNodeWithLatestChanges(ctx, sub, rmbNodeClient, nodeID, neverShutDown, false, dedicatedFarm, config.Power.OverProvisionCPU)
			if err != nil {
				return nil, fmt.Errorf("failed to include node with id %d", nodeID)
			}
			nodes[nodeID] = configNode
		}
	}

	return nodes, nil
}

func getNodeWithLatestChanges(
	ctx context.Context,
	sub Sub,
	rmbNodeClient RMB,
	nodeID uint32,
	neverShutDown,
	hasClaimedResources,
	dedicatedFarm bool,
	overProvisionCPU float32,
) (node, error) {

	log.Debug().Uint32("ID", nodeID).Msg("Include node")
	nodeObj, err := sub.GetNode(nodeID)
	if err != nil {
		return node{}, err
	}

	configNode := node{
		Node: *nodeObj,
	}

	configNode.neverShutDown = neverShutDown
	configNode.resources.overProvisionCPU = overProvisionCPU

	price, err := sub.GetDedicatedNodePrice(nodeID)
	if err != nil {
		return node{}, err
	}

	if price != 0 || dedicatedFarm {
		configNode.dedicated = true
	}

	rentContract, err := sub.GetNodeRentContract(nodeID)
	if err != nil {
		return node{}, err
	}

	configNode.hasActiveRentContract = rentContract != 0

	powerTarget, err := sub.GetPowerTarget(nodeID)
	if err != nil {
		return node{}, err
	}

	if powerTarget.State.IsUp && powerTarget.Target.IsUp && configNode.powerState != on {
		configNode.powerState = on
		configNode.lastTimeAwake = time.Now()
		configNode.lastTimePowerStateChanged = time.Now()
	}

	if powerTarget.State.IsDown && powerTarget.Target.IsDown && configNode.powerState != off {
		configNode.powerState = off
		configNode.lastTimePowerStateChanged = time.Now()
	}

	// update resources for nodes that have no claimed resources
	// we do not update the resources for the nodes that have claimed resources because those resources should not be overwritten until the timeout
	if !hasClaimedResources {
		stats, err := rmbNodeClient.Statistics(ctx, uint32(configNode.TwinID))
		if err != nil {
			return node{}, err
		}
		configNode.updateResources(stats)
	}

	pools, err := rmbNodeClient.GetStoragePools(ctx, uint32(configNode.TwinID))
	if err != nil {
		return node{}, err
	}
	configNode.pools = pools

	gpus, err := rmbNodeClient.ListGPUs(ctx, uint32(configNode.TwinID))
	if err != nil {
		return node{}, err
	}
	configNode.gpus = gpus

	return configNode, nil
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
func (s *state) filterNodesPower(states []powerState) []node {
	filtered := make([]node, 0)
	for _, node := range s.nodes {
		if slices.Contains(states, node.powerState) {
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
		if node.isUnused() && !node.PublicConfig.HasValue && !node.neverShutDown &&
			time.Since(node.lastTimePowerStateChanged) >= periodicWakeUpDuration {
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
		if n.resources.total.sru == 0 {
			return fmt.Errorf("node %d: total SRU is required", n.ID)
		}
		if n.resources.total.cru == 0 {
			return fmt.Errorf("node %d: total CRU is required", n.ID)
		}
		if n.resources.total.mru == 0 {
			return fmt.Errorf("node %d: total MRU is required", n.ID)
		}
		if n.resources.total.hru == 0 {
			return fmt.Errorf("node %d: total HRU is required", n.ID)
		}
	}

	// required values for power
	if s.config.Power.WakeUpThreshold == 0 {
		s.config.Power.WakeUpThreshold = defaultWakeUpThreshold
	}

	if s.config.Power.WakeUpThreshold < minWakeUpThreshold {
		s.config.Power.WakeUpThreshold = minWakeUpThreshold
		log.Warn().Msgf("The setting wake_up_threshold should be in the range [%d, %d], setting it to minimum value %d", minWakeUpThreshold, MaxWakeUpThreshold, minWakeUpThreshold)
	}

	if s.config.Power.WakeUpThreshold > MaxWakeUpThreshold {
		s.config.Power.WakeUpThreshold = MaxWakeUpThreshold
		log.Warn().Msgf("The setting wake_up_threshold should be in the range [%d, %d], setting it to maximum value %d", minWakeUpThreshold, MaxWakeUpThreshold, minWakeUpThreshold)
	}

	if s.config.Power.PeriodicWakeUpStart.PeriodicWakeUpTime().Hour() == 0 && s.config.Power.PeriodicWakeUpStart.PeriodicWakeUpTime().Minute() == 0 {
		s.config.Power.PeriodicWakeUpStart = wakeUpDate(time.Now())
		log.Warn().Time("periodic wakeup start", s.config.Power.PeriodicWakeUpStart.PeriodicWakeUpTime()).Msg("The setting periodic_wake_up_start is zero. setting it with current time")
	}
	s.config.Power.PeriodicWakeUpStart = wakeUpDate(s.config.Power.PeriodicWakeUpStart.PeriodicWakeUpTime())

	if s.config.Power.PeriodicWakeUpLimit == 0 {
		s.config.Power.PeriodicWakeUpLimit = defaultPeriodicWakeUPLimit
		log.Warn().Msgf("The setting periodic_wake_up_limit should be greater then 0! setting it to %d", defaultPeriodicWakeUPLimit)
	}

	return nil
}
