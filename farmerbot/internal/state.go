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
func newState(ctx context.Context, sub Substrate, rmbNodeClient RMB, cfg Config) (*state, error) {
	s := state{config: cfg}

	// required from power for nodes
	if s.config.Power.OverProvisionCPU == 0 {
		s.config.Power.OverProvisionCPU = defaultCPUProvision
	}

	if s.config.Power.OverProvisionCPU < 1 || s.config.Power.OverProvisionCPU > 4 {
		return nil, fmt.Errorf("cpu over provision should be a value between 1 and 4 not %v", s.config.Power.OverProvisionCPU)
	}

	farm, err := sub.GetFarm(cfg.FarmID)
	if err != nil {
		return nil, err
	}

	s.farm = *farm

	nodes, err := fetchNodes(ctx, sub, rmbNodeClient, cfg, farm.DedicatedFarm)
	if err != nil {
		return nil, err
	}

	s.nodes = nodes

	if err := s.validate(); err != nil {
		return nil, err
	}

	return &s, nil
}

func fetchNodes(ctx context.Context, sub Substrate, rmbNodeClient RMB, config Config, dedicatedFarm bool) (map[uint32]node, error) {
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

			log.Debug().Uint32("nodeID", nodeID).Msg("Add node")
			configNode, err := getNode(ctx, sub, rmbNodeClient, nodeID, neverShutDown, false, dedicatedFarm, on)
			if err != nil {
				return nil, fmt.Errorf("failed to add node with id %d with error: %w", nodeID, err)
			}
			nodes[nodeID] = configNode
		}
	}

	return nodes, nil
}

func getNode(
	ctx context.Context,
	sub Substrate,
	rmbNodeClient RMB,
	nodeID uint32,
	neverShutDown,
	hasClaimedResources,
	dedicatedFarm bool,
	oldPowerState powerState,
) (node, error) {

	nodeObj, err := sub.GetNode(nodeID)
	if err != nil {
		return node{}, fmt.Errorf("failed to get node %d from substrate with error: %w", nodeID, err)
	}

	configNode := node{
		Node:          *nodeObj,
		neverShutDown: neverShutDown,
	}

	price, err := sub.GetDedicatedNodePrice(nodeID)
	if err != nil {
		return node{}, fmt.Errorf("failed to get node %d dedicated price from substrate with error: %w", nodeID, err)
	}

	if price != 0 || dedicatedFarm {
		configNode.dedicated = true
	}

	rentContract, err := sub.GetNodeRentContract(nodeID)
	if errors.Is(err, substrate.ErrNotFound) {
		configNode.hasActiveRentContract = false
	} else if err != nil {
		return node{}, fmt.Errorf("failed to get node %d rent contract from substrate with error: %w", nodeID, err)
	}

	configNode.hasActiveRentContract = rentContract != 0

	powerTarget, err := sub.GetPowerTarget(nodeID)
	if err != nil {
		return node{}, fmt.Errorf("failed to get node %d power target from substrate with error: %w", nodeID, err)
	}

	configNode.powerState = oldPowerState
	if powerTarget.State.IsDown && powerTarget.Target.IsUp && configNode.powerState != wakingUp {
		log.Warn().Uint32("nodeID", uint32(nodeObj.ID)).Msg("Updating power, Power target is waking up")
		configNode.powerState = wakingUp
		configNode.lastTimePowerStateChanged = time.Now()
	}

	if powerTarget.State.IsUp && powerTarget.Target.IsUp && configNode.powerState != on {
		log.Warn().Uint32("nodeID", uint32(nodeObj.ID)).Msg("Updating power, Power target is on")
		configNode.powerState = on
		configNode.lastTimeAwake = time.Now()
		configNode.lastTimePowerStateChanged = time.Now()
	}

	if powerTarget.State.IsUp && powerTarget.Target.IsDown && configNode.powerState != shuttingDown {
		log.Warn().Uint32("nodeID", uint32(nodeObj.ID)).Msg("Updating power, Power target is shutting down")
		configNode.powerState = shuttingDown
		configNode.lastTimePowerStateChanged = time.Now()
	}

	if powerTarget.State.IsDown && powerTarget.Target.IsDown && configNode.powerState != off {
		log.Warn().Uint32("nodeID", uint32(nodeObj.ID)).Msg("Updating power, Power target is off")
		configNode.powerState = off
		configNode.lastTimePowerStateChanged = time.Now()
	}

	// update resources for nodes that have no claimed resources
	// we do not update the resources for the nodes that have claimed resources because those resources should not be overwritten until the timeout
	if !hasClaimedResources {
		stats, err := rmbNodeClient.Statistics(ctx, uint32(configNode.TwinID))
		if err != nil {
			return node{}, fmt.Errorf("failed to get node %d statistics from rmb with error: %w", nodeID, err)
		}
		configNode.updateResources(stats)
	}

	pools, err := rmbNodeClient.GetStoragePools(ctx, uint32(configNode.TwinID))
	if err != nil {
		return node{}, fmt.Errorf("failed to get node %d pools from rmb with error: %w", nodeID, err)
	}
	configNode.pools = pools

	gpus, err := rmbNodeClient.ListGPUs(ctx, uint32(configNode.TwinID))
	if err != nil {
		return node{}, fmt.Errorf("failed to get node %d gpus from rmb with error: %w", nodeID, err)
	}
	configNode.gpus = gpus

	return configNode, nil
}

// addNode adds a node in the config
func (s *state) addNode(node node) {
	s.m.Lock()
	defer s.m.Unlock()

	s.nodes[uint32(node.ID)] = node
}

func (s *state) deleteNode(nodeID uint32) {
	s.m.Lock()
	defer s.m.Unlock()

	delete(s.nodes, nodeID)
}

// UpdateNode updates a node in the config
func (s *state) updateNode(node node) error {
	s.m.Lock()
	defer s.m.Unlock()

	nodeID := uint32(node.ID)

	_, ok := s.nodes[nodeID]
	if !ok {
		return fmt.Errorf("node %d is not found", nodeID)
	}

	s.nodes[nodeID] = node

	return nil
}

// FilterNodesPower filters ON, waking up, shutting down, or OFF nodes
func (s *state) filterNodesPower(states []powerState) map[uint32]node {
	filtered := make(map[uint32]node)
	for nodeID, node := range s.nodes {
		if slices.Contains(states, node.powerState) {
			filtered[nodeID] = node
		}
	}
	return filtered
}

// FilterAllowedNodesToShutDown filters nodes that are allowed to shut down
func (s *state) filterAllowedNodesToShutDown() map[uint32]node {
	filtered := make(map[uint32]node)
	for nodeID, node := range s.nodes {
		if node.canShutDown() {
			filtered[nodeID] = node
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

		// visit: https://github.com/threefoldtech/tfgrid-sdk-go/issues/586
		// if n.resources.total.hru == 0 {
		// 	return fmt.Errorf("node %d: total HRU is required", n.ID)
		// }
	}

	// required values for power
	if s.config.Power.WakeUpThreshold == 0 {
		s.config.Power.WakeUpThreshold = defaultWakeUpThreshold
		log.Warn().Msgf("The setting wake_up_threshold has not been set. setting it to %d", defaultWakeUpThreshold)
	}

	if s.config.Power.WakeUpThreshold < 1 || s.config.Power.WakeUpThreshold > 100 {
		return fmt.Errorf("invalid wake-up threshold %d, should be between [1-100]", s.config.Power.WakeUpThreshold)
	}

	if s.config.Power.WakeUpThreshold < minWakeUpThreshold {
		s.config.Power.WakeUpThreshold = minWakeUpThreshold
		log.Warn().Msgf("The setting wake_up_threshold should be in the range [%d, %d]. Setting it to minimum value %d", minWakeUpThreshold, maxWakeUpThreshold, minWakeUpThreshold)
	}

	if s.config.Power.WakeUpThreshold > maxWakeUpThreshold {
		s.config.Power.WakeUpThreshold = maxWakeUpThreshold
		log.Warn().Msgf("The setting wake_up_threshold should be in the range [%d, %d]. Setting it to maximum value %d", minWakeUpThreshold, maxWakeUpThreshold, minWakeUpThreshold)
	}

	if s.config.Power.PeriodicWakeUpStart.PeriodicWakeUpTime().Hour() == 0 && s.config.Power.PeriodicWakeUpStart.PeriodicWakeUpTime().Minute() == 0 {
		s.config.Power.PeriodicWakeUpStart = wakeUpDate(time.Now())
		log.Warn().Time("periodic wakeup start", s.config.Power.PeriodicWakeUpStart.PeriodicWakeUpTime()).Msg("The setting periodic_wake_up_start has not been set. Setting it with current time")
	}
	s.config.Power.PeriodicWakeUpStart = wakeUpDate(s.config.Power.PeriodicWakeUpStart.PeriodicWakeUpTime())

	if s.config.Power.PeriodicWakeUpLimit == 0 {
		s.config.Power.PeriodicWakeUpLimit = defaultPeriodicWakeUPLimit
		log.Warn().Msgf("The setting periodic_wake_up_limit has not been set. setting it to %d", defaultPeriodicWakeUPLimit)
	}

	return nil
}
