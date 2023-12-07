package internal

import (
	"fmt"
	"math"
	"strings"
	"time"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	zos "github.com/threefoldtech/zos/client"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// Config is the inputs for configuration for farmerbot
type Config struct {
	FarmID             uint32   `yaml:"farm_id"`
	IncludedNodes      []uint32 `yaml:"included_nodes"`
	ExcludedNodes      []uint32 `yaml:"excluded_nodes"`
	NeverShutDownNodes []uint32 `yaml:"never_shutdown_nodes"`
	Power              power    `yaml:"power"`
}

type powerState uint8

const (
	on = powerState(iota)
	wakingUP
	off
	shuttingDown
)

// Node represents a node type
type node struct {
	substrate.Node

	// data manager computes
	resources             consumableResources
	publicIPsUsed         uint64
	pools                 []pkg.PoolMetrics
	gpus                  []zos.GPU
	hasActiveRentContract bool

	// TODO: check if we can update
	dedicated                 bool
	neverShutDown             bool
	powerState                powerState
	timeoutClaimedResources   time.Time
	lastTimePowerStateChanged time.Time
	lastTimeAwake             time.Time
	timesRandomWakeUps        int
}

// NodeOptions represents the options to find a node
type NodeOptions struct {
	NodeExclude  []uint32 `json:"node_exclude,omitempty"`
	HasGPUs      uint8    `json:"has_gpus,omitempty"`
	GPUVendors   []string `json:"gpu_vendors,omitempty"`
	GPUDevices   []string `json:"gpu_devices,omitempty"`
	Certified    bool     `json:"certified,omitempty"`
	Dedicated    bool     `json:"dedicated,omitempty"`
	PublicConfig bool     `json:"public_config,omitempty"`
	PublicIPs    uint64   `json:"public_ips,omitempty"`
	HRU          uint64   `json:"hru,omitempty"`
	SRU          uint64   `json:"sru,omitempty"`
	CRU          uint64   `json:"cru,omitempty"`
	MRU          uint64   `json:"mru,omitempty"`
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
func (n *node) canClaimResources(cap capacity) bool {
	free := n.freeCapacity()
	return n.resources.total.cru >= cap.cru && free.cru >= cap.cru && free.mru >= cap.mru && free.hru >= cap.hru && free.sru >= cap.sru
}

// ClaimResources claims the resources from a node
func (n *node) claimResources(c capacity) {
	n.resources.used.add(c)
}

// FreeCapacity calculates the free capacity of a node
func (n *node) freeCapacity() capacity {
	total := n.resources.total
	total.cru = uint64(math.Ceil(float64(total.cru) * float64(n.resources.overProvisionCPU)))
	return total.subtract(n.resources.used)
}

// ConsumableResources for node resources
type consumableResources struct {
	overProvisionCPU float32
	total            capacity
	used             capacity
	system           capacity
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
	result.cru = subtract(cap.cru, c.cru)
	result.mru = subtract(cap.mru, c.mru)
	result.sru = subtract(cap.sru, c.sru)
	result.hru = subtract(cap.hru, c.hru)
	return result
}

func subtract(a uint64, b uint64) uint64 {
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
	OverProvisionCPU    float32    `yaml:"overprovision_cpu,omitempty"`
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

// MarshalText marshals the wake up TOML date
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
