package internal

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/exp/maps"
)

var thresholdPercentages = ThresholdPercentages{cru: 200, mru: 80, sru: 80, hru: 80}

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

	demand := calculateDemandBasedOnThresholds(totalResources, usedResources, thresholdPercentages)
	if demand.cru == 0 || demand.mru == 0 || demand.sru == 0 || demand.hru == 0 {

		log.Info().Any("resources usage", usedResources).Msg("Too high resource usage")
		return f.resourceUsageTooHigh(sub)
	}

	log.Info().Any("resources usage", usedResources).Msg("Too low resource usage")
	return f.resourceUsageTooLow(sub, usedResources, totalResources)
}

func calculateDemandBasedOnThresholds(total, used capacity, thresholdPercentages ThresholdPercentages) capacity {
	var demand capacity

	if float64(used.cru)/float64(total.cru)*100 > thresholdPercentages.cru {
		demand.cru = uint64((float64(used.cru)/float64(total.cru)*100 - thresholdPercentages.cru) / 100 * float64(total.cru))
	}
	if float64(used.mru)/float64(total.mru)*100 > thresholdPercentages.mru {
		demand.mru = uint64((float64(used.mru)/float64(total.mru)*100 - thresholdPercentages.mru) / 100 * float64(total.mru))
	}
	if float64(used.sru)/float64(total.sru)*100 > thresholdPercentages.sru {
		demand.sru = uint64((float64(used.sru)/float64(total.sru)*100 - thresholdPercentages.sru) / 100 * float64(total.sru))
	}
	if float64(used.hru)/float64(total.hru)*100 > thresholdPercentages.hru {
		demand.hru = uint64((float64(used.hru)/float64(total.hru)*100 - thresholdPercentages.hru) / 100 * float64(total.hru))
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
		return nil, fmt.Errorf("unable to meet resource demand with available nodes")
	}

	return selectedNodes, nil
}

func (f *FarmerBot) resourceUsageTooHigh(sub Substrate) error {
	log.Info().Msg("Too high resource usage. Powering on some nodes")
	used, total := calculateResourceUsage(f.nodes)
	nodes, err := f.selectNodesToPowerOn(calculateDemandBasedOnThresholds(total, used, thresholdPercentages))
	if err != nil {
		return err
	}

	for nodeID, node := range nodes {
		if node.powerState == off {
			log.Info().Uint32("nodeID", uint32(nodeID)).Msg("Too much resource usage. Turning on node")
			return f.powerOn(sub, uint32(nodeID))
		}
	}

	return fmt.Errorf("no available node to wake up, resources usage is high")
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
	var underutilized []node = findUnderutilizedNodes(nodesAllowedToShutdown) // why didn't := work here?

	for _, node := range underutilized {
		if nodesLeftOnline == 1 {
			break
		}
		nodesLeftOnline -= 1

		cpNewUsedResources := newUsedResources
		cpNewTotalResources := newTotalResources

		newUsedResources.cru -= node.resources.used.cru
		newUsedResources.mru -= node.resources.used.sru
		newUsedResources.sru -= node.resources.used.mru
		newUsedResources.hru -= node.resources.used.hru

		newTotalResources.cru -= node.resources.total.cru
		newTotalResources.mru -= node.resources.total.sru
		newTotalResources.sru -= node.resources.total.mru
		newTotalResources.hru -= node.resources.total.hru

		currentDemand := calculateDemandBasedOnThresholds(newTotalResources, newUsedResources, thresholdPercentages)

		if checkResourcesMeetDemand(newTotalResources, newUsedResources, currentDemand) {

			log.Info().Uint32("nodeID", uint32(node.ID)).Any("resources usage", newUsedResources).Msg("Resource usage too low. Turning off unused node")
			err := f.powerOff(sub, uint32(node.ID))
			if err != nil {
				log.Error().Err(err).Uint32("nodeID", uint32(node.ID)).Msg("Failed to power off node")
				// restore the newUsedResources and newTotalResources
				newUsedResources = cpNewUsedResources
				newTotalResources = cpNewTotalResources

				if node.powerState == shuttingDown {
					continue
				}
			}
		}
	}
	return nil
}

func findUnderutilizedNodes(nodes map[uint32]node) []node {
	var underutilizedNodes []node
	for _, n := range nodes {
		if n.powerState == on && isNodeUnderutilized(n) {
			underutilizedNodes = append(underutilizedNodes, n)
		}
	}
	return underutilizedNodes
}

func isNodeUnderutilized(n node) bool {
	return n.resources.used.cru == 0 && n.resources.used.sru == 0 &&
		n.resources.used.mru == 0 && n.resources.used.hru == 0
}

func checkResourcesMeetDemand(total, used, demand capacity) bool {
	remaining := capacity{
		cru: total.cru - used.cru,
		sru: total.sru - used.sru,
		mru: total.mru - used.mru,
		hru: total.hru - used.hru,
	}

	// Check if remaining resources meet or exceed demand for each resource type
	meetsCRUDemand := remaining.cru >= uint64(demand.cru)
	meetsSRUDemand := remaining.sru >= demand.sru
	meetsMRUDemand := remaining.mru >= demand.mru
	meetsHRUDemand := remaining.hru >= demand.hru

	return meetsCRUDemand && meetsSRUDemand && meetsMRUDemand && meetsHRUDemand
}
