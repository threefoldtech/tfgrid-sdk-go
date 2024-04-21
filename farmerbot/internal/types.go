package internal

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	zos "github.com/threefoldtech/zos/client"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// Config is the inputs for configuration for farmerbot
type Config struct {
	FarmID                  uint32   `yaml:"farm_id"`
	IncludedNodes           []uint32 `yaml:"included_nodes"`
	ExcludedNodes           []uint32 `yaml:"excluded_nodes"`
	PriorityNodes           []uint32 `yaml:"priority_nodes"`
	NeverShutDownNodes      []uint32 `yaml:"never_shutdown_nodes"`
	Power                   power    `yaml:"power"`
	ContinueOnPoweringOnErr bool
}

type powerState uint8

const (
	on = powerState(iota)
	wakingUp
	off
	shuttingDown
)

// Node represents a node type
type node struct {
	substrate.Node
	resources             consumableResources
	publicIPsUsed         uint64
	pools                 []pkg.PoolMetrics
	gpus                  []zos.GPU
	hasActiveRentContract bool
	hasActiveContracts    bool
	dedicated             bool
	neverShutDown         bool

	powerState                powerState
	timeoutClaimedResources   time.Time
	lastTimePowerStateChanged time.Time
	lastTimeAwake             time.Time
	timesRandomWakeUps        int
	// set the time the node wakes up every day
	lastTimePeriodicWakeUp time.Time
}

// NodeFilterOption represents the options to find a node
type NodeFilterOption struct {
	NodesExcluded []uint32 `json:"nodes_excluded,omitempty"`
	NumGPU        uint8    `json:"num_gpu,omitempty"`
	GPUVendors    []string `json:"gpu_vendors,omitempty"`
	GPUDevices    []string `json:"gpu_devices,omitempty"`
	Certified     bool     `json:"certified,omitempty"`
	Dedicated     bool     `json:"dedicated,omitempty"`
	PublicConfig  bool     `json:"public_config,omitempty"`
	PublicIPs     uint64   `json:"public_ips,omitempty"`
	HRU           uint64   `json:"hru,omitempty"` // in GB
	SRU           uint64   `json:"sru,omitempty"` // in GB
	CRU           uint64   `json:"cru,omitempty"`
	MRU           uint64   `json:"mru,omitempty"` // in GB
}

// TODO: if one update failed maybe other would not fail
func (n *node) update(
	ctx context.Context,
	sub Substrate,
	rmbNodeClient RMB,
	neverShutDown,
	dedicatedFarm,
	continueOnPoweringOnErr bool,
) error {
	nodeID := uint32(n.ID)
	if nodeID == 0 {
		return fmt.Errorf("invalid node id %d", nodeID)
	}

	twinID := uint32(n.TwinID)
	if twinID == 0 {
		return fmt.Errorf("invalid twin id %d", nodeID)
	}

	nodeObj, err := sub.GetNode(nodeID)
	if err != nil {
		return fmt.Errorf("failed to get node %d from substrate with error: %w", nodeID, err)
	}

	n.Node = *nodeObj
	n.neverShutDown = neverShutDown

	price, err := sub.GetDedicatedNodePrice(nodeID)
	if err != nil {
		return fmt.Errorf("failed to get node %d dedicated price from substrate with error: %w", nodeID, err)
	}

	if price != 0 || dedicatedFarm {
		n.dedicated = true
	}

	rentContract, err := sub.GetNodeRentContract(nodeID)
	if errors.Is(err, substrate.ErrNotFound) {
		n.hasActiveRentContract = false
	} else if err != nil {
		return fmt.Errorf("failed to get node %d rent contract from substrate with error: %w", nodeID, err)
	}

	n.hasActiveRentContract = rentContract != 0

	activeContracts, err := sub.GetNodeContracts(nodeID)
	if err != nil {
		return fmt.Errorf("failed to get node %d active contracts from substrate with error: %w", nodeID, err)
	}

	n.hasActiveContracts = len(activeContracts) > 0

	powerTarget, err := sub.GetPowerTarget(nodeID)
	if err != nil {
		return fmt.Errorf("failed to get node %d power target from substrate with error: %w", nodeID, err)
	}

	if powerTarget.State.IsDown && powerTarget.Target.IsUp && n.powerState != wakingUp {
		log.Warn().Uint32("nodeID", uint32(nodeObj.ID)).Msg("Updating power, Power target is waking up")
		n.powerState = wakingUp
		n.lastTimePowerStateChanged = time.Now()
	}

	if powerTarget.State.IsUp && powerTarget.Target.IsUp && n.powerState != on {
		log.Warn().Uint32("nodeID", uint32(nodeObj.ID)).Msg("Updating power, Power target is on")
		n.powerState = on
		n.lastTimeAwake = time.Now()
		n.lastTimePowerStateChanged = time.Now()
	}

	if powerTarget.State.IsUp && powerTarget.Target.IsDown && n.powerState != shuttingDown {
		log.Warn().Uint32("nodeID", uint32(nodeObj.ID)).Msg("Updating power, Power target is shutting down")
		n.powerState = shuttingDown
		n.lastTimePowerStateChanged = time.Now()
	}

	if powerTarget.State.IsDown && powerTarget.Target.IsDown && n.powerState != off {
		log.Warn().Uint32("nodeID", uint32(nodeObj.ID)).Msg("Updating power, Power target is off")
		n.powerState = off
		n.lastTimePowerStateChanged = time.Now()
	}

	// don't call rmb over off nodes (state and target are off)
	if (n.powerState == off || n.powerState == wakingUp) &&
		continueOnPoweringOnErr {
		log.Warn().Uint32("nodeID", uint32(nodeObj.ID)).Msg("Node is off, will skip rmb calls")
		return nil
	}

	// update resources for nodes that have no claimed resources
	// we do not update the resources for the nodes that have claimed resources because those resources should not be overwritten until the timeout
	hasClaimedResources := n.timeoutClaimedResources.After(time.Now())
	if !hasClaimedResources {
		stats, err := rmbNodeClient.Statistics(ctx, twinID)
		if err != nil {
			return fmt.Errorf("failed to get node %d statistics from rmb with error: %w", nodeID, err)
		}
		n.updateResources(stats)
	}

	pools, err := rmbNodeClient.GetStoragePools(ctx, twinID)
	if err != nil {
		return fmt.Errorf("failed to get node %d pools from rmb with error: %w", nodeID, err)
	}
	n.pools = pools

	gpus, err := rmbNodeClient.ListGPUs(ctx, twinID)
	if err != nil {
		return fmt.Errorf("failed to get node %d gpus from rmb with error: %w", nodeID, err)
	}
	n.gpus = gpus

	return nil
}

// UpdateResources updates the node resources from zos resources stats
func (n *node) updateResources(stats zos.Counters) {
	n.resources.total.update(stats.Total)
	n.resources.used.update(stats.Used)
	n.resources.system.update(stats.System)
	n.publicIPsUsed = stats.Used.IPV4U
}

// IsUnused node is an empty node
func (n *node) isUnused() bool {
	resources := n.resources.used.subtract(n.resources.system)
	return resources.isEmpty() && !n.hasActiveRentContract
}

// CanClaimResources checks if a node can claim some resources
func (n *node) canClaimResources(cap capacity, overProvisionCPU int8) bool {
	free := n.freeCapacity(overProvisionCPU)
	return n.resources.total.cru >= cap.cru && free.cru >= cap.cru && free.mru >= cap.mru && free.hru >= cap.hru && free.sru >= cap.sru
}

// ClaimResources claims the resources from a node
func (n *node) claimResources(c capacity) {
	n.resources.used.add(c)
}

// FreeCapacity calculates the free capacity of a node
func (n *node) freeCapacity(overProvisionCPU int8) capacity {
	total := n.resources.total
	total.cru = uint64(math.Ceil(float64(total.cru) * float64(overProvisionCPU)))
	return total.subtract(n.resources.used)
}

// canShutDown return if the node can be shutdown
// nodes with public config can't be shutdown
// Do not shutdown a node that just came up (give it some time `periodicWakeUpDuration`)
func (n *node) canShutDown() bool {
	if n.powerState != on ||
		!n.isUnused() ||
		n.PublicConfig.HasValue ||
		n.neverShutDown ||
		n.hasActiveRentContract ||
		n.hasActiveContracts ||
		n.timeoutClaimedResources.After(time.Now()) ||
		time.Since(n.lastTimePowerStateChanged) < periodicWakeUpDuration {
		return false
	}
	return true
}

// ConsumableResources for node resources
type consumableResources struct {
	total  capacity
	used   capacity
	system capacity
}

// Capacity is node resource capacity
type capacity struct {
	hru uint64
	sru uint64
	cru uint64
	mru uint64
}

// IsEmpty checks empty capacity
func (cap *capacity) isEmpty() bool {
	return cap.cru == 0 && cap.mru == 0 && cap.sru == 0 && cap.hru == 0
}

func (cap *capacity) update(c gridtypes.Capacity) {
	cap.cru = c.CRU
	cap.mru = uint64(c.MRU)
	cap.sru = uint64(c.SRU)
	cap.hru = uint64(c.HRU)
}

// Add adds a new for capacity
func (cap *capacity) add(c capacity) {
	cap.cru += c.cru
	cap.mru += c.mru
	cap.sru += c.sru
	cap.hru += c.hru
}

// Subtract subtracts a new capacity
func (cap *capacity) subtract(c capacity) (result capacity) {
	result.cru = subtractOrZero(cap.cru, c.cru)
	result.mru = subtractOrZero(cap.mru, c.mru)
	result.sru = subtractOrZero(cap.sru, c.sru)
	result.hru = subtractOrZero(cap.hru, c.hru)
	return result
}

func subtractOrZero(a uint64, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return 0
}

// Power represents power configuration
type power struct {
	WakeUpThreshold     uint8      `yaml:"wake_up_threshold"`
	PeriodicWakeUpStart wakeUpDate `yaml:"periodic_wake_up_start"`
	PeriodicWakeUpLimit uint8      `yaml:"periodic_wake_up_limit"`
	OverProvisionCPU    int8       `yaml:"overprovision_cpu,omitempty"`
}

// wakeUpDate is the date to wake up all nodes
type wakeUpDate time.Time

// UnmarshalText unmarshal the given TOML string into wakeUp date
func (d *wakeUpDate) UnmarshalText(b []byte) error {
	s := strings.Trim(string(b), "\"")
	t, err := time.Parse("03:04PM", s)
	if err != nil {
		return err
	}
	*d = wakeUpDate(t)
	return nil
}

// MarshalText marshals the wake up yaml date
func (d wakeUpDate) MarshalText() ([]byte, error) {
	date := time.Time(d)

	dayTime := "AM"
	if date.Hour() >= 12 {
		dayTime = "PM"
		date = date.Add(time.Duration(-12) * time.Hour)
	}

	timeFormat := fmt.Sprintf("%02d:%02d%s", date.Hour(), date.Minute(), dayTime)
	return []byte(timeFormat), nil
}

// PeriodicWakeUpTime returns periodic wake up date
func (d wakeUpDate) PeriodicWakeUpTime() time.Time {
	date := time.Time(d)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return today.Local().Add(time.Hour*time.Duration(date.Hour()) +
		time.Minute*time.Duration(date.Minute()) +
		0)
}
