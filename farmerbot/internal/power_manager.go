package internal

import (
	"fmt"
	"math"
	"math/rand"
	"slices"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
)

// PowerManager manages the power of nodes
type PowerManager struct {
	state    *state
	identity substrate.Identity
}

// NewPowerManager creates a new Power Manager
func NewPowerManager(identity substrate.Identity, state *state) PowerManager {
	return PowerManager{
		state:    state,
		identity: identity,
	}
}

// PowerOn sets the node power state ON
func (p *PowerManager) PowerOn(sub Sub, nodeID uint32) error {
	log.Info().Uint32("node ID", nodeID).Str("manager", "power").Msg("POWER ON")
	p.state.m.Lock()
	defer p.state.m.Unlock()

	node, ok := p.state.nodes[nodeID]
	if !ok {
		return fmt.Errorf("node %d is not found", nodeID)
	}

	if node.powerState == on || node.powerState == wakingUP {
		return nil
	}

	_, err := sub.SetNodePowerTarget(p.identity, nodeID, true)
	if err != nil {
		return errors.Wrapf(err, "failed to set node %d power target to up", nodeID)
	}

	node.powerState = wakingUP
	node.lastTimePowerStateChanged = time.Now()

	return p.state.updateNode(node)
}

// PowerOff sets the node power state OFF
func (p *PowerManager) PowerOff(sub Sub, nodeID uint32) error {
	log.Info().Uint32("node ID", nodeID).Str("manager", "power").Msg("POWER OFF")
	p.state.m.Lock()
	defer p.state.m.Unlock()

	node, ok := p.state.nodes[nodeID]
	if !ok {
		return fmt.Errorf("node %d is not found", nodeID)
	}

	if node.powerState == off || node.powerState == shuttingDown {
		return nil
	}

	if node.neverShutDown {
		return errors.Errorf("cannot power off node, node is configured to never be shutdown")
	}

	if node.PublicConfig.HasValue {
		return errors.Errorf("cannot power off node, node has public config")
	}

	if node.timeoutClaimedResources.After(time.Now()) {
		return errors.Errorf("cannot power off node, node has claimed resources")
	}

	onNodes := p.state.filterNodesPower([]powerState{on})

	if len(onNodes) < 2 {
		return errors.Errorf("cannot power off node, at least one node should be on in the farm")
	}

	_, err := sub.SetNodePowerTarget(p.identity, nodeID, false)
	if err != nil {
		return errors.Wrapf(err, "failed to set node %d power target to down", nodeID)
	}

	node.powerState = shuttingDown
	node.lastTimePowerStateChanged = time.Now()

	return p.state.updateNode(node)
}

// FindNode finds an available node in the farm
func (p *PowerManager) FindNode(sub Sub, nodeOptions NodeOptions) (uint32, error) {
	log.Info().Str("manager", "node").Msg("Finding a node")

	nodeOptionsCapacity := capacity{
		hru: nodeOptions.HRU,
		sru: nodeOptions.SRU,
		cru: nodeOptions.CRU,
		mru: nodeOptions.MRU,
	}

	if (len(nodeOptions.GPUVendors) > 0 || len(nodeOptions.GPUDevices) > 0) && nodeOptions.HasGPUs == 0 {
		// at least one gpu in case the user didn't provide the amount
		nodeOptions.HasGPUs = 1
	}

	log.Debug().Interface("required filter options", nodeOptions)

	if nodeOptions.PublicIPs > 0 {
		var publicIpsUsedByNodes uint64
		for _, node := range p.state.nodes {
			publicIpsUsedByNodes += node.publicIPsUsed
		}

		if publicIpsUsedByNodes+nodeOptions.PublicIPs > uint64(len(p.state.farm.PublicIPs)) {
			return 0, fmt.Errorf("not enough public ips available for farm %d", p.state.farm.ID)
		}
	}

	var possibleNodes []node
	for _, node := range p.state.nodes {
		gpus := node.gpus
		if nodeOptions.HasGPUs > 0 {
			if len(nodeOptions.GPUVendors) > 0 {
				gpus = filterGPUs(gpus, nodeOptions.GPUVendors, false)
			}

			if len(nodeOptions.GPUDevices) > 0 {
				gpus = filterGPUs(gpus, nodeOptions.GPUDevices, true)
			}

			if len(gpus) < int(nodeOptions.HasGPUs) {
				continue
			}
		}

		if nodeOptions.Certified && !node.Certification.IsCertified {
			continue
		}

		if nodeOptions.PublicConfig && !node.PublicConfig.HasValue {
			continue
		}

		if node.hasActiveRentContract {
			continue
		}

		if nodeOptions.Dedicated {
			if !node.dedicated || !node.isUnused() {
				continue
			}
		} else {
			if node.dedicated && nodeOptionsCapacity != node.resources.total {
				continue
			}
		}

		if slices.Contains(nodeOptions.NodeExclude, uint32(node.ID)) {
			continue
		}

		if !node.canClaimResources(nodeOptionsCapacity) {
			continue
		}

		possibleNodes = append(possibleNodes, node)
	}

	if len(possibleNodes) == 0 {
		return 0, fmt.Errorf("could not find a suitable node with the given options: %+v", possibleNodes)
	}

	// Sort the nodes on power state (the ones that are ON first then waking up, off, shutting down)
	sort.Slice(possibleNodes, func(i, j int) bool {
		return possibleNodes[i].powerState < possibleNodes[j].powerState
	})

	nodeFound := possibleNodes[0]
	log.Debug().Str("manager", "node").Uint32("node ID", uint32(nodeFound.ID)).Msg("Found a node")

	// claim the resources until next update of the data
	// add a timeout (after 30 minutes we update the resources)
	nodeFound.timeoutClaimedResources = time.Now().Add(timeoutPowerStateChange)

	if nodeOptions.Dedicated || nodeOptions.HasGPUs > 0 {
		// claim all capacity
		nodeFound.claimResources(nodeFound.resources.total)
	} else {
		nodeFound.claimResources(nodeOptionsCapacity)
	}

	// claim public ips until next update of the data
	if nodeOptions.PublicIPs > 0 {
		nodeFound.publicIPsUsed += nodeOptions.PublicIPs
	}

	// power on the node if it is down or if it is shutting down
	if nodeFound.powerState == off || nodeFound.powerState == shuttingDown {
		if err := p.PowerOn(sub, uint32(nodeFound.ID)); err != nil {
			return 0, fmt.Errorf("failed to power on found node %d", nodeFound.ID)
		}
	}

	// update claimed resources
	err := p.state.updateNode(nodeFound)
	if err != nil {
		return 0, fmt.Errorf("failed to power on found node %d", nodeFound.ID)
	}

	return uint32(nodeFound.ID), nil
}

// PeriodicWakeUp for waking up nodes daily
func (p *PowerManager) PeriodicWakeUp(sub Sub) error {
	now := time.Now()

	offNodes := p.state.filterNodesPower([]powerState{off})

	periodicWakeUpStart := p.state.config.Power.PeriodicWakeUpStart.PeriodicWakeUpTime()

	var wakeUpCalls uint8
	nodesLen := len(p.state.nodes)

	for _, node := range offNodes {
		// TODO: why??
		if now.Day() == 1 && now.Hour() == 1 && now.Minute() >= 0 && now.Minute() < 5 {
			node.timesRandomWakeUps = 0
		}

		if periodicWakeUpStart.Before(now) && node.lastTimeAwake.Before(periodicWakeUpStart) {
			// Fixed periodic wake up (once a day)
			// we wake up the node if the periodic wake up start time has started and only if the last time the node was awake
			// was before the periodic wake up start of that day
			log.Info().Uint32("node ID", uint32(node.ID)).Str("manager", "power").Msg("Periodic wake up")
			err := p.PowerOn(sub, uint32(node.ID))
			if err != nil {
				log.Error().Err(err).Uint32("node ID", uint32(node.ID)).Str("manager", "power").Msg("failed to wake up")
				continue
			}

			wakeUpCalls += 1
			if wakeUpCalls >= p.state.config.Power.PeriodicWakeUpLimit {
				// reboot X nodes at a time others will be rebooted 5 min later
				break
			}
		} else if node.timesRandomWakeUps < defaultRandomWakeUpsAMonth &&
			int(rand.Int31())%((8460-(defaultRandomWakeUpsAMonth*6)-
				(defaultRandomWakeUpsAMonth*(nodesLen-1))/int(math.Min(float64(p.state.config.Power.PeriodicWakeUpLimit), float64(nodesLen))))/
				defaultRandomWakeUpsAMonth) == 0 {
			// Random periodic wake up (10 times a month on average if the node is almost always down)
			// we execute this code every 5 minutes => 288 times a day => 8640 times a month on average (30 days)
			// but we have 30 minutes of periodic wake up every day (6 times we do not go through this code) => so 282 times a day => 8460 times a month on average (30 days)
			// as we do a random wake up 10 times a month we know the node will be on for 30 minutes 10 times a month so we can subtract 6 times the amount of random wake ups a month
			// we also do not go through the code if we have woken up too many nodes at once => subtract (10 * (n-1))/min(periodic_wake up_limit, amount_of_nodes) from 8460
			// now we can divide that by 10 and randomly generate a number in that range, if it's 0 we do the random wake up
			log.Info().Uint32("node ID", uint32(node.ID)).Str("manager", "power").Msg("Random wake up")
			err := p.PowerOn(sub, uint32(node.ID))
			if err != nil {
				log.Error().Err(err).Uint32("node ID", uint32(node.ID)).Str("manager", "power").Msg("failed to wake up")
				continue
			}

			wakeUpCalls += 1
			if wakeUpCalls >= p.state.config.Power.PeriodicWakeUpLimit {
				// reboot X nodes at a time others will be rebooted 5 min later
				break
			}
		}
	}

	return nil
}

// PowerManagement for power management nodes
func (p *PowerManager) PowerManagement(sub Sub) error {
	usedResources, totalResources := p.calculateResourceUsage()

	if totalResources == 0 {
		return nil
	}

	resourceUsage := uint8(100 * usedResources / totalResources)
	if resourceUsage >= p.state.config.Power.WakeUpThreshold {
		log.Info().Uint8("resources usage", resourceUsage).Str("manager", "power").Msg("Too high resource usage")
		return p.resourceUsageTooHigh(sub)
	}

	log.Info().Uint8("resources usage", resourceUsage).Str("manager", "power").Msg("Too low resource usage")
	return p.resourceUsageTooLow(sub, usedResources, totalResources)
}

func (p *PowerManager) calculateResourceUsage() (uint64, uint64) {
	usedResources := capacity{}
	totalResources := capacity{}

	nodes := p.state.filterNodesPower([]powerState{on, wakingUP})

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

func (p *PowerManager) resourceUsageTooHigh(sub Sub) error {
	offNodes := p.state.filterNodesPower([]powerState{off})

	if len(offNodes) > 0 {
		node := offNodes[0]
		log.Info().Uint32("node ID", uint32(node.ID)).Str("manager", "power").Msg("Too much resource usage. Turning on node")
		return p.PowerOn(sub, uint32(node.ID))
	}

	return fmt.Errorf("no available node to wake up, resources usage is high")
}

func (p *PowerManager) resourceUsageTooLow(sub Sub, usedResources, totalResources uint64) error {
	onNodes := p.state.filterNodesPower([]powerState{on})

	// nodes with public config can't be shutdown
	// Do not shutdown a node that just came up (give it some time)
	nodesAllowedToShutdown := p.state.filterAllowedNodesToShutDown()

	if len(onNodes) > 1 {
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

			newResourceUsage := uint8(100 * newUsedResources / newTotalResources)
			if newResourceUsage < p.state.config.Power.WakeUpThreshold {
				// we need to keep the resource percentage lower then the threshold
				log.Info().Uint32("node ID", uint32(node.ID)).Uint8("resources usage", newResourceUsage).Str("manager", "power").Msg("Resource usage too low. Turning off unused node")
				err := p.PowerOff(sub, uint32(node.ID))
				if err != nil {
					log.Error().Err(err).Uint32("node ID", uint32(node.ID)).Str("manager", "power").Msg("Power off node")
					nodesLeftOnline += 1
					newUsedResources += node.resources.used.hru + node.resources.used.sru +
						node.resources.used.mru + node.resources.used.cru
					newTotalResources += node.resources.total.hru + node.resources.total.sru +
						node.resources.total.mru + node.resources.total.cru
				}
			}
		}
	} else {
		log.Debug().Str("manager", "power").Msg("Nothing to shutdown.")
	}

	return nil
}

// FilterIncludesSubStr filters a string slice according to if elements include a sub string
func filterGPUs(gpus []gpu, vendorsOrDevices []string, device bool) (filtered []gpu) {
	for _, gpu := range gpus {
		for _, filter := range vendorsOrDevices {
			if gpu.device == filter && device {
				filtered = append(filtered, gpu)
			}

			if gpu.vendor == filter && !device {
				filtered = append(filtered, gpu)
			}
		}
	}
	return
}
