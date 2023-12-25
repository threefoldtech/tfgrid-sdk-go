package internal

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
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

	return f.updateNode(node)
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
		return fmt.Errorf("failed to set node '%d' power target to down with error: %w", nodeID, err)
	}

	node.powerState = shuttingDown
	node.lastTimePowerStateChanged = time.Now()

	return f.updateNode(node)
}

// manageNodesPower for power management nodes
func (f *FarmerBot) manageNodesPower(sub Substrate) error {
	nodes := f.filterNodesPower([]powerState{on, wakingUp})

	usedResources, totalResources := calculateResourceUsage(nodes)
	if totalResources == 0 {
		return nil
	}

	resourceUsage := uint8(100 * usedResources / totalResources)
	if resourceUsage >= f.config.Power.WakeUpThreshold {
		log.Info().Uint8("resources usage", resourceUsage).Msg("Too high resource usage")
		return f.resourceUsageTooHigh(sub)
	}

	log.Info().Uint8("resources usage", resourceUsage).Msg("Too low resource usage")
	return f.resourceUsageTooLow(sub, usedResources, totalResources)
}

func calculateResourceUsage(nodes map[uint32]node) (uint64, uint64) {
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

	used := usedResources.cru + usedResources.hru + usedResources.mru + usedResources.sru
	total := totalResources.cru + totalResources.hru + totalResources.mru + totalResources.sru

	return used, total
}

func (f *FarmerBot) resourceUsageTooHigh(sub Substrate) error {
	offNodes := f.filterNodesPower([]powerState{off})

	if len(offNodes) > 0 {
		node := offNodes[0]
		log.Info().Uint32("nodeID", uint32(node.ID)).Msg("Too much resource usage. Turning on node")
		return f.powerOn(sub, uint32(node.ID))
	}

	return fmt.Errorf("no available node to wake up, resources usage is high")
}

func (f *FarmerBot) resourceUsageTooLow(sub Substrate, usedResources, totalResources uint64) error {
	onNodes := f.filterNodesPower([]powerState{on})

	// nodes with public config can't be shutdown
	// Do not shutdown a node that just came up (give it some time `periodicWakeUpDuration`)
	nodesAllowedToShutdown := f.filterAllowedNodesToShutDown()

	if len(onNodes) <= 1 {
		log.Debug().Msg("Nothing to shutdown.")
		return nil
	}

	newUsedResources := usedResources
	newTotalResources := totalResources
	nodesLeftOnline := len(onNodes)

	// shutdown a node if there is more then 1 unused node (aka keep at least one node online)
	for _, node := range nodesAllowedToShutdown {
		if nodesLeftOnline == 1 {
			break
		}
		nodesLeftOnline -= 1
		newUsedResources -= node.resources.used.hru + node.resources.used.sru +
			node.resources.used.mru + node.resources.used.cru
		newTotalResources -= node.resources.total.hru + node.resources.total.sru +
			node.resources.total.mru + node.resources.total.cru

		if newTotalResources == 0 {
			break
		}

		if totalResources == 0 {
			log.Error().Uint32("nodeID", uint32(node.ID)).Msg("cannot calculate resources usage after removing node resources, nodes total resources is 0")
			nodesLeftOnline += 1
			newUsedResources += node.resources.used.hru + node.resources.used.sru +
				node.resources.used.mru + node.resources.used.cru
			newTotalResources += node.resources.total.hru + node.resources.total.sru +
				node.resources.total.mru + node.resources.total.cru
			continue
		}

		newResourceUsage := uint8(100 * newUsedResources / newTotalResources)
		if newResourceUsage < f.config.Power.WakeUpThreshold {
			// we need to keep the resource percentage lower then the threshold
			log.Info().Uint32("nodeID", uint32(node.ID)).Uint8("resources usage", newResourceUsage).Msg("Resource usage too low. Turning off unused node")
			err := f.powerOff(sub, uint32(node.ID))
			if err != nil {
				log.Error().Err(err).Uint32("nodeID", uint32(node.ID)).Msg("Power off node")
				nodesLeftOnline += 1
				newUsedResources += node.resources.used.hru + node.resources.used.sru +
					node.resources.used.mru + node.resources.used.cru
				newTotalResources += node.resources.total.hru + node.resources.total.sru +
					node.resources.total.mru + node.resources.total.cru
			}
		}
	}

	return nil
}
