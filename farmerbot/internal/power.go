package internal

import (
	"fmt"
	"math"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/exp/maps"
)

// powerOn sets the node power state ON
func (f *FarmerBot) powerOn(sub Substrate, nodeID uint32) error {
	log.Info().Uint32("nodeID", nodeID).Msg("POWER ON")
	f.m.Lock()
	defer f.m.Unlock()

	node, ok := f.nodes[nodeID]
	if !ok {
		return fmt.Errorf("node %d is not found", nodeID)
	}

	if node.powerState == on || node.powerState == wakingUp {
		return nil
	}

	_, err := sub.SetNodePowerTarget(f.identity, nodeID, true)
	if err != nil {
		return fmt.Errorf("failed to set node %d power target to up with error: %w", nodeID, err)
	}

	node.powerState = wakingUp
	node.lastTimeAwake = time.Now()
	node.lastTimePowerStateChanged = time.Now()

	f.nodes[nodeID] = node
	return nil
}

// powerOff sets the node power state OFF
func (f *FarmerBot) powerOff(sub Substrate, nodeID uint32) error {
	log.Info().Uint32("nodeID", nodeID).Msg("POWER OFF")
	f.m.Lock()
	defer f.m.Unlock()

	node, ok := f.nodes[nodeID]
	if !ok {
		return fmt.Errorf("node '%d' is not found", nodeID)
	}

	if node.powerState == off || node.powerState == shuttingDown {
		return nil
	}

	if node.neverShutDown {
		return fmt.Errorf("cannot power off node '%d', node is configured to never be shutdown", nodeID)
	}

	if node.PublicConfig.HasValue {
		return fmt.Errorf("cannot power off node '%d', node has public config", nodeID)
	}

	if node.timeoutClaimedResources.After(time.Now()) {
		return fmt.Errorf("cannot power off node '%d', node has claimed resources", nodeID)
	}

	if node.hasActiveRentContract {
		return fmt.Errorf("cannot power off node '%d', node has a rent contract", nodeID)
	}

	if !node.isUnused() {
		return fmt.Errorf("cannot power off node '%d', node is used", nodeID)
	}

	if time.Since(node.lastTimePowerStateChanged) < periodicWakeUpDuration {
		return fmt.Errorf("cannot power off node '%d', node is still in its wakeup duration", nodeID)
	}

	onNodes := f.filterNodesPower([]powerState{on})

	if len(onNodes) < 2 {
		return fmt.Errorf("cannot power off node '%d', at least one node should be on in the farm", nodeID)
	}

	_, err := sub.SetNodePowerTarget(f.identity, nodeID, false)
	if err != nil {
		powerTarget, getErr := sub.GetPowerTarget(nodeID)
		if getErr != nil {
			return fmt.Errorf("failed to get node '%d' power target with error: %w", nodeID, getErr)
		}

		if powerTarget.State.IsDown || powerTarget.Target.IsDown {
			log.Warn().Uint32("nodeID", nodeID).Msg("Node is shutting down although it failed to set power target in tfchain")
			node.powerState = shuttingDown
			node.lastTimePowerStateChanged = time.Now()
			f.nodes[nodeID] = node
		}

		return fmt.Errorf("failed to set node '%d' power target to down with error: %w", nodeID, err)
	}

	node.powerState = shuttingDown
	node.lastTimePowerStateChanged = time.Now()

	f.nodes[nodeID] = node
	return nil
}

// manageNodesPower for power management nodes
func (f *FarmerBot) manageNodesPower(sub Substrate) error {
	nodes := f.filterNodesPower([]powerState{on, wakingUp})

	usedResources, totalResources := calculateResourceUsage(nodes)

	demand := calculateDemandBasedOnThresholds(totalResources, usedResources, f.config.Power.WakeUpThresholdPercentages)
	if demand.cru == 0 || demand.mru == 0 || demand.sru == 0 || demand.hru == 0 {
		log.Info().Any("resources usage", usedResources).Msg("Too low resource usage")
		return f.resourceUsageTooLow(sub, usedResources, totalResources)
	}

	log.Info().Any("resources usage", usedResources).Msg("Too high resource usage")
	return f.resourceUsageTooHigh(sub, demand)
}

func calculateDemandBasedOnThresholds(total, used capacity, thresholdPercentages ThresholdPercentages) capacity {
	var demand capacity

	if float64(used.cru)/float64(total.cru)*100 > thresholdPercentages.CRU {
		demand.cru = uint64(math.Ceil((float64(used.cru)/float64(total.cru)*100 - thresholdPercentages.CRU) / 100 * float64(total.cru)))
	}
	if float64(used.mru)/float64(total.mru)*100 > thresholdPercentages.MRU {
		demand.mru = uint64(math.Ceil((float64(used.mru)/float64(total.mru)*100 - thresholdPercentages.MRU) / 100 * float64(total.mru)))
	}
	if float64(used.sru)/float64(total.sru)*100 > thresholdPercentages.SRU {
		demand.sru = uint64(math.Ceil((float64(used.sru)/float64(total.sru)*100 - thresholdPercentages.SRU) / 100 * float64(total.sru)))
	}
	if total.hru > 0 && float64(used.hru)/float64(total.hru)*100 > thresholdPercentages.HRU {
		demand.hru = uint64(math.Ceil((float64(used.hru)/float64(total.hru)*100 - thresholdPercentages.HRU) / 100 * float64(total.hru)))
	}

	return demand
}

func calculateResourceUsage(nodes map[uint32]node) (capacity, capacity) {
	usedResources := capacity{}
	totalResources := capacity{}

	for _, node := range nodes {
		if node.hasActiveRentContract {
			usedResources.add(node.resources.total)
		} else {
			usedResources.add(node.resources.used)
		}
		totalResources.add(node.resources.total)
	}

	return usedResources, totalResources
}

func (f *FarmerBot) selectNodesToPowerOn(demand capacity) ([]node, error) {
	var selectedNodes []node
	remainingDemand := demand

	for _, node := range f.nodes {
		if node.powerState != off {
			continue // Skip nodes that are already on or waking up
		}

		// Check if this node can contribute to the remaining demand
		contribute := false
		if remainingDemand.cru > 0 && uint64(node.Resources.CRU) >= remainingDemand.cru {
			contribute = true
			remainingDemand.cru -= uint64(node.Resources.CRU)
		}
		if remainingDemand.sru > 0 && uint64(node.Resources.SRU) >= remainingDemand.sru {
			contribute = true
			remainingDemand.sru -= uint64(node.Resources.SRU)
		}
		if remainingDemand.mru > 0 && uint64(node.Resources.MRU) >= remainingDemand.mru {
			contribute = true
			remainingDemand.mru -= uint64(node.Resources.MRU)
		}
		if remainingDemand.hru > 0 && uint64(node.Resources.HRU) >= remainingDemand.hru {
			contribute = true
			remainingDemand.hru -= uint64(node.Resources.HRU)
		}

		if contribute {
			selectedNodes = append(selectedNodes, node)
		}

		// Check if all demands have been met
		if remainingDemand.cru <= 0 && remainingDemand.sru <= 0 && remainingDemand.mru <= 0 && remainingDemand.hru <= 0 {
			break // All demands have been met, no need to check more nodes
		}
	}

	if remainingDemand.cru > 0 || remainingDemand.sru > 0 || remainingDemand.mru > 0 || remainingDemand.hru > 0 {
		return nil, fmt.Errorf("unable to meet resources demand with available nodes")
	}

	return selectedNodes, nil
}

func (f *FarmerBot) resourceUsageTooHigh(sub Substrate, demand capacity) error {
	log.Info().Msg("Too high resource usage. Powering on some nodes")
	nodes, err := f.selectNodesToPowerOn(demand)
	if err != nil {
		return err
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no available nodes to wake up, resources usage is high")
	}

	for _, node := range nodes {
		if node.powerState == off {
			log.Info().Uint32("nodeID", uint32(node.ID)).Msg("Too much resource usage. Turning on node")
			if err := f.powerOn(sub, uint32(node.ID)); err != nil {
				return fmt.Errorf("couldn't power on node %v with error: %w", node.ID, err)
			}
		}
	}

	return nil
}

func (f *FarmerBot) resourceUsageTooLow(sub Substrate, usedResources, totalResources capacity) error {
	onNodes := f.filterNodesPower([]powerState{on})

	// nodes with public config can't be shutdown
	// Do not shutdown a node that just came up (give it some time `periodicWakeUpDuration`)
	nodesAllowedToShutdown := f.filterAllowedNodesToShutDown()

	if len(onNodes) <= 1 {
		log.Debug().Msg("Nothing to shutdown")
		return nil
	}

	if len(nodesAllowedToShutdown) == 0 {
		log.Debug().Msg("No nodes are allowed to shutdown")
		return nil
	}

	log.Debug().Uints32("nodes IDs", maps.Keys(nodesAllowedToShutdown)).Msg("Nodes allowed to shutdown")

	newUsedResources := usedResources
	newTotalResources := totalResources
	nodesLeftOnline := len(onNodes)

	// shutdown a node if there is more than an unused node (aka keep at least one node online)
	for _, node := range nodesAllowedToShutdown {
		if nodesLeftOnline == 1 {
			break
		}
		nodesLeftOnline -= 1

		cpNewUsedResources := newUsedResources
		cpNewTotalResources := newTotalResources

		newUsedResources.cru -= node.resources.used.cru
		newUsedResources.sru -= node.resources.used.sru
		newUsedResources.mru -= node.resources.used.mru
		newUsedResources.hru -= node.resources.used.hru

		newTotalResources.cru -= node.resources.total.cru
		newTotalResources.sru -= node.resources.total.sru
		newTotalResources.mru -= node.resources.total.mru
		newTotalResources.hru -= node.resources.total.hru

		if newTotalResources.isEmpty() {
			break
		}

		currentDemand := calculateDemandBasedOnThresholds(newTotalResources, newUsedResources, f.config.Power.WakeUpThresholdPercentages)

		if checkResourcesMeetDemand(newTotalResources, newUsedResources, currentDemand) {
			log.Info().Uint32("nodeID", uint32(node.ID)).Any("resources usage", newUsedResources).Msg("Resource usage too low. Turning off unused node")
			err := f.powerOff(sub, uint32(node.ID))
			if err != nil {
				log.Error().Err(err).Uint32("nodeID", uint32(node.ID)).Msg("Failed to power off node")
				if node.powerState == shuttingDown {
					continue
				}
				// restore the newUsedResources and newTotalResources
				newUsedResources = cpNewUsedResources
				newTotalResources = cpNewTotalResources
				nodesLeftOnline += 1
			}
		}
	}
	return nil
}

func checkResourcesMeetDemand(total, used, demand capacity) bool {
	remaining := capacity{
		cru: total.cru - used.cru,
		sru: total.sru - used.sru,
		mru: total.mru - used.mru,
		hru: total.hru - used.hru,
	}

	// Check if remaining resources meet or exceed demand for each resource type
	meetsCRUDemand := remaining.cru >= demand.cru
	meetsSRUDemand := remaining.sru >= demand.sru
	meetsMRUDemand := remaining.mru >= demand.mru
	meetsHRUDemand := remaining.hru >= demand.hru

	return meetsCRUDemand && meetsSRUDemand && meetsMRUDemand && meetsHRUDemand
}
