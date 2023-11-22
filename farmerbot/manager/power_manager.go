// Package manager provides how to manage nodes, farms and power
package manager

import (
	"math"
	"math/rand"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/constants"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/models"
)

// PowerManager manages the power of nodes
type PowerManager struct {
	db       models.RedisManager
	identity substrate.Identity
	subConn  Sub
}

// NewPowerManager creates a new Power Manager
func NewPowerManager(identity substrate.Identity, subConn Sub, db models.RedisManager) PowerManager {
	return PowerManager{db, identity, subConn}
}

// Configure configures the power configs in farmerbot
func (p *PowerManager) Configure(power models.Power) error {
	if power.WakeUpThreshold == 0 {
		power.WakeUpThreshold = constants.DefaultWakeUpThreshold
	}

	if power.WakeUpThreshold < constants.MinWakeUpThreshold {
		power.WakeUpThreshold = constants.MinWakeUpThreshold
		log.Warn().Msgf("[POWER MANAGER] The setting wake_up_threshold should be in the range [%d, %d]", constants.MinWakeUpThreshold, constants.MaxWakeUpThreshold)
	}

	if power.WakeUpThreshold > constants.MaxWakeUpThreshold {
		power.WakeUpThreshold = constants.MaxWakeUpThreshold
		log.Warn().Msgf("[POWER MANAGER] The setting wake_up_threshold should be in the range [%d, %d]", constants.MinWakeUpThreshold, constants.MaxWakeUpThreshold)
	}

	if power.PeriodicWakeUpStart.PeriodicWakeUpTime().IsZero() {
		power.PeriodicWakeUpStart = models.WakeUpDate(time.Now())
	}
	power.PeriodicWakeUpStart = models.WakeUpDate(power.PeriodicWakeUpStart.PeriodicWakeUpTime())

	if power.PeriodicWakeUpLimit == 0 {
		power.PeriodicWakeUpLimit = constants.DefaultPeriodicWakeUPLimit
		log.Warn().Msg("[POWER MANAGER] The setting periodic_wake_up_limit should be greater then 0!")
	}

	return p.db.SetPower(power)
}

// PowerOn sets the node power state ON
func (p *PowerManager) PowerOn(nodeID uint32) error {
	log.Info().Msgf("[POWER MANAGER] POWER ON: %d", nodeID)

	node, err := p.db.GetNode(nodeID)
	if err != nil {
		return err
	}

	if node.PowerState == models.ON || node.PowerState == models.WakingUP {
		return nil
	}

	_, err = p.subConn.SetNodePowerState(p.identity, true)
	if err != nil {
		return err
	}

	node.PowerState = models.WakingUP
	node.LastTimePowerStateChanged = time.Now()

	return p.db.UpdatesNode(node)
}

// PowerOff sets the node power state OFF
func (p *PowerManager) PowerOff(nodeID uint32) error {
	log.Info().Msgf("[POWER MANAGER] POWER OFF: %d", nodeID)

	node, err := p.db.GetNode(nodeID)
	if err != nil {
		return err
	}

	if node.PowerState == models.OFF || node.PowerState == models.ShuttingDown {
		return nil
	}

	_, err = p.subConn.SetNodePowerState(p.identity, false)
	if err != nil {
		return err
	}

	node.PowerState = models.ShuttingDown
	node.LastTimePowerStateChanged = time.Now()

	return p.db.UpdatesNode(node)
}

func (p *PowerManager) PowerOnAllNodes(nodeID uint32) error {
	offNodes, err := p.db.FilterNodesPower([]models.PowerState{models.OFF, models.ShuttingDown})
	if err != nil {
		return err
	}

	for _, node := range offNodes {
		err := p.PowerOn(node.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

// PeriodicWakeUp for waking up nodes daily
func (p *PowerManager) PeriodicWakeUp() error {
	now := time.Now()
	power, err := p.db.GetPower()
	if err != nil {
		return errors.Wrap(err, "failed to get power from db")
	}

	nodes, err := p.db.GetNodes()
	if err != nil {
		return errors.Wrap(err, "failed to get nodes from db")
	}

	offNodes, err := p.db.FilterNodesPower([]models.PowerState{models.OFF})
	if err != nil {
		return errors.Wrap(err, "failed to get nodes from db")
	}

	periodicWakeUpStart := power.PeriodicWakeUpStart.PeriodicWakeUpTime()

	var wakeUpCalls uint8
	nodesLen := len(nodes)

	for _, node := range offNodes {
		// TODO: why??
		if now.Day() == 1 && now.Hour() == 1 && now.Minute() >= 0 && now.Minute() < 5 {
			node.TimesRandomWakeUps = 0
		}

		if periodicWakeUpStart.Before(now) && node.LastTimeAwake.Before(periodicWakeUpStart) {
			// Fixed periodic wake up (once a day)
			// we wake up the node if the periodic wake up start time has started and only if the last time the node was awake
			// was before the periodic wake up start of that day
			log.Info().Msgf("[POWER MANAGER] Periodic wake up for node %d", node.ID)
			err := p.PowerOn(node.ID)
			if err != nil {
				log.Error().Err(err).Msgf("[POWER MANAGER] failed to wake up for node %d", node.ID)
				continue
			}

			wakeUpCalls += 1
			if wakeUpCalls >= power.PeriodicWakeUpLimit {
				// reboot X nodes at a time others will be rebooted 5 min later
				break
			}
		} else if node.TimesRandomWakeUps < constants.DefaultRandomWakeUpsAMonth &&
			int(rand.Int31())%((8460-(constants.DefaultRandomWakeUpsAMonth*6)-
				(constants.DefaultRandomWakeUpsAMonth*(nodesLen-1))/int(math.Min(float64(power.PeriodicWakeUpLimit), float64(nodesLen))))/
				constants.DefaultRandomWakeUpsAMonth) == 0 {
			// Random periodic wake up (10 times a month on average if the node is almost always down)
			// we execute this code every 5 minutes => 288 times a day => 8640 times a month on average (30 days)
			// but we have 30 minutes of periodic wake up every day (6 times we do not go through this code) => so 282 times a day => 8460 times a month on average (30 days)
			// as we do a random wake up 10 times a month we know the node will be on for 30 minutes 10 times a month so we can subtract 6 times the amount of random wake ups a month
			// we also do not go through the code if we have woken up too many nodes at once => subtract (10 * (n-1))/min(periodic_wake up_limit, amount_of_nodes) from 8460
			// now we can divide that by 10 and randomly generate a number in that range, if it's 0 we do the random wake up
			log.Info().Msgf("[POWER MANAGER] Random wake up for node %d", node.ID)
			err := p.PowerOn(node.ID)
			if err != nil {
				log.Error().Err(err).Msgf("[POWER MANAGER] failed to wake up for node %d", node.ID)
				continue
			}

			wakeUpCalls += 1
			if wakeUpCalls >= power.PeriodicWakeUpLimit {
				// reboot X nodes at a time others will be rebooted 5 min later
				break
			}
		}
	}

	return nil
}

// PowerManagement for power management nodes
func (p *PowerManager) PowerManagement() error {
	usedResources, totalResources, err := p.calculateResourceUsage()
	if err != nil {
		return err
	}

	if totalResources == 0 {
		return nil
	}

	power, err := p.db.GetPower()
	if err != nil {
		return errors.Wrap(err, "failed to get power from db")
	}

	resourceUsage := uint8(100 * usedResources / totalResources)
	if resourceUsage >= power.WakeUpThreshold {
		log.Info().Msgf("[POWER MANAGER] Too much resource usage: %d", resourceUsage)
		return p.resourceUsageTooHigh()
	}

	log.Info().Msgf("[POWER MANAGER] Too low resource usage: %d", resourceUsage)
	return p.resourceUsageTooLow(power, usedResources, totalResources)
}

func (p *PowerManager) calculateResourceUsage() (uint64, uint64, error) {
	usedResources := models.Capacity{}
	totalResources := models.Capacity{}

	nodes, err := p.db.FilterNodesPower([]models.PowerState{models.ON, models.WakingUP})
	if err != nil {
		return 0, 0, err
	}

	for _, node := range nodes {
		if node.HasActiveRentContract {
			usedResources.Add(node.Resources.Total)
		} else {
			usedResources.Add(node.Resources.Used)
		}
		totalResources.Add(node.Resources.Total)
	}

	used := usedResources.CRU + usedResources.HRU + usedResources.MRU + usedResources.SRU
	total := totalResources.CRU + totalResources.HRU + totalResources.MRU + totalResources.SRU

	return used, total, nil
}

func (p *PowerManager) resourceUsageTooHigh() error {
	offNodes, err := p.db.FilterNodesPower([]models.PowerState{models.OFF})
	if err != nil {
		return err
	}

	if len(offNodes) > 0 {
		node := offNodes[0]
		log.Info().Msgf("[POWER MANAGER] Too much resource usage. Turning on node %d", node.ID)
		return p.PowerOn(node.ID)
	}

	return nil
}

func (p *PowerManager) resourceUsageTooLow(power models.Power, usedResources, totalResources uint64) error {
	onNodes, err := p.db.FilterNodesPower([]models.PowerState{models.ON})
	if err != nil {
		return err
	}

	// nodes with public config can't be shutdown
	// Do not shutdown a node that just came up (give it some time)
	nodesAllowedToShutdown, err := p.db.FilterAllowedNodesToShutDown()
	if err != nil {
		return err
	}

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
			newUsedResources -= node.Resources.Used.HRU + node.Resources.Used.SRU +
				node.Resources.Used.MRU + node.Resources.Used.CRU
			newTotalResources -= node.Resources.Total.HRU + node.Resources.Total.SRU +
				node.Resources.Total.MRU + node.Resources.Total.CRU

			if newTotalResources == 0 {
				break
			}

			newResourceUsage := uint8(100 * newUsedResources / newTotalResources)
			if newResourceUsage < power.WakeUpThreshold {
				// we need to keep the resource percentage lower then the threshold
				log.Info().Msgf("[POWER MANAGER] Resource usage too low: %d. Turning off unused node %d", newResourceUsage, node.ID)
				err := p.PowerOff(node.ID)
				if err != nil {
					log.Error().Err(err).Msgf("[POWER MANAGER] Job to power off node %d failed", node.ID)
					nodesLeftOnline += 1
					newUsedResources += node.Resources.Used.HRU + node.Resources.Used.SRU +
						node.Resources.Used.MRU + node.Resources.Used.CRU
					newTotalResources += node.Resources.Total.HRU + node.Resources.Total.SRU +
						node.Resources.Total.MRU + node.Resources.Total.CRU
				}
			}
		}
	} else {
		log.Debug().Msg("[POWER MANAGER] Nothing to shutdown.")
	}

	return nil
}
