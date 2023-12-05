package internal

import (
	"math"
	"time"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// Config is the inputs for configuration for farmerbot
type Config struct {
	FarmID        uint32   `json:"farm_id" yaml:"farm_id" toml:"farm_id"`
	IncludedNodes []uint32 `json:"included_nodes" yaml:"included_nodes" toml:"included_nodes"`
	ExcludedNodes []uint32 `json:"excluded_nodes" yaml:"excluded_nodes" toml:"excluded_nodes"`
	Power         Power    `json:"power" yaml:"power" toml:"power"`
}

// Farm of the farmer
type farm struct {
	ID          uint32 `json:"id" yaml:"id" toml:"id"`
	Description string `json:"description,omitempty" yaml:"description,omitempty" toml:"description,omitempty"`
	PublicIPs   uint64 `json:"public_ips,omitempty" yaml:"public_ips,omitempty" toml:"public_ips,omitempty"`
}

// Node represents a node type
type node struct {
	substrate.Node
	Dedicated bool `json:"dedicated,omitempty" yaml:"dedicated,omitempty" toml:"dedicated,omitempty"`

	// data manager computes
	Resources             ConsumableResources `json:"resources" yaml:"resources" toml:"resources"`
	PublicIPsUsed         uint64              `json:"public_ips_used,omitempty" yaml:"public_ips_used,omitempty" toml:"public_ips_used,omitempty"`
	Pools                 []pkg.PoolMetrics   `json:"pools,omitempty" yaml:"pools,omitempty" toml:"pools,omitempty"`
	GPUs                  []gpu               `json:"gpus,omitempty" yaml:"gpus,omitempty" toml:"gpus,omitempty"`
	HasActiveRentContract bool                `json:"has_active_rent_contract,omitempty" yaml:"has_active_rent_contract,omitempty" toml:"has_active_rent_contract,omitempty"`

	// TODO:
	NeverShutDown bool `json:"never_shutdown,omitempty" yaml:"never_shutdown,omitempty" toml:"never_shutdown,omitempty"`

	PowerState                PowerState `json:"power_state,omitempty" yaml:"power_state,omitempty" toml:"power_state,omitempty"`
	TimeoutClaimedResources   time.Time  `json:"timeout_claimed_resources,omitempty" yaml:"timeout_claimed_resources,omitempty" toml:"timeout_claimed_resources,omitempty"`
	LastTimePowerStateChanged time.Time  `json:"last_time_power_state_changed,omitempty" yaml:"last_time_power_state_changed,omitempty" toml:"last_time_power_state_changed,omitempty"`
	LastTimeAwake             time.Time  `json:"last_time_awake,omitempty" yaml:"last_time_awake,omitempty" toml:"last_time_awake,omitempty"`
	TimesRandomWakeUps        int        `json:"times_random_wake_ups,omitempty" yaml:"times_random_wake_ups,omitempty" toml:"times_random_wake_ups,omitempty"`
}

// NodeOptions represents the options to find a node
type nodeOptions struct {
	NodeExclude  []uint32 `json:"node_exclude,omitempty" yaml:"node_exclude,omitempty" toml:"node_exclude,omitempty"`
	HasGPUs      uint8    `json:"has_gpus,omitempty" yaml:"has_gpus,omitempty" toml:"has_gpus,omitempty"`
	GPUVendors   []string `json:"gpu_vendors,omitempty" yaml:"gpu_vendors,omitempty" toml:"gpu_vendors,omitempty"`
	GPUDevices   []string `json:"gpu_devices,omitempty" yaml:"gpu_devices,omitempty" toml:"gpu_devices,omitempty"`
	Certified    bool     `json:"certified,omitempty" yaml:"certified,omitempty" toml:"certified,omitempty"`
	Dedicated    bool     `json:"dedicated,omitempty" yaml:"dedicated,omitempty" toml:"dedicated,omitempty"`
	PublicConfig bool     `json:"public_config,omitempty" yaml:"public_config,omitempty" toml:"public_config,omitempty"`
	PublicIPs    uint64   `json:"public_ips,omitempty" yaml:"public_ips,omitempty" toml:"public_ips,omitempty"`
	Capacity     capacity `json:"capacity,omitempty" yaml:"capacity,omitempty" toml:"capacity,omitempty"`
}

// GPU information
type gpu struct {
	ID       string `json:"id" yaml:"id" toml:"id"`
	Vendor   string `json:"vendor" yaml:"vendor" toml:"vendor"`
	Device   string `json:"device" yaml:"device" toml:"device"`
	Contract uint64 `json:"contract" yaml:"contract" toml:"contract"`
}

type zosResourcesStatistics struct {
	// Total system capacity
	Total gridtypes.Capacity `json:"total"`
	// Used capacity this include user + system resources
	Used gridtypes.Capacity `json:"used"`
	// System resource reserved by zos
	System gridtypes.Capacity `json:"system"`
}

// UpdateResources updates the node resources from zos resources stats
func (n *node) updateResources(stats zosResourcesStatistics) {
	n.Resources.Total.update(stats.Total)
	n.Resources.Used.update(stats.Used)
	n.Resources.System.update(stats.System)
	n.PublicIPsUsed = stats.Used.IPV4U
}

// IsUnused node is an empty node
func (n *node) isUnused() bool {
	resources := n.Resources.Used.subtract(n.Resources.System)
	return resources.isEmpty() && !n.HasActiveRentContract
}

// CanClaimResources checks if a node can claim some resources
func (n *node) canClaimResources(cap capacity) bool {
	free := n.freeCapacity()
	return n.Resources.Total.CRU >= cap.CRU && free.CRU >= cap.CRU && free.MRU >= cap.MRU && free.HRU >= cap.HRU && free.SRU >= cap.SRU
}

// ClaimResources claims the resources from a node
func (n *node) claimResources(c capacity) {
	n.Resources.Used.add(c)
}

// FreeCapacity calculates the free capacity of a node
func (n *node) freeCapacity() capacity {
	total := n.Resources.Total
	total.CRU = uint64(math.Ceil(float64(total.CRU) * float64(n.Resources.OverProvisionCPU)))
	return total.subtract(n.Resources.Used)
}

// ConsumableResources for node resources
type ConsumableResources struct {
	OverProvisionCPU float32  `json:"overprovision_cpu,omitempty" yaml:"overprovision_cpu,omitempty" toml:"overprovision_cpu,omitempty"` // how much we allow over provisioning the CPU range: [1;3]
	Total            capacity `json:"total" yaml:"total" toml:"total"`
	Used             capacity `json:"used,omitempty" yaml:"used,omitempty" toml:"used,omitempty"`
	System           capacity `json:"system,omitempty" yaml:"system,omitempty" toml:"system,omitempty"`
}

// Capacity is node resource capacity
type capacity struct {
	HRU uint64 `json:"HRU"`
	SRU uint64 `json:"SRU"`
	CRU uint64 `json:"CRU"`
	MRU uint64 `json:"MRU"`
}

// IsEmpty checks empty capacity
func (cap *capacity) isEmpty() bool {
	return cap.CRU == 0 && cap.MRU == 0 && cap.SRU == 0 && cap.HRU == 0
}

func (cap *capacity) update(c gridtypes.Capacity) {
	cap.CRU = c.CRU
	cap.MRU = uint64(c.MRU)
	cap.SRU = uint64(c.SRU)
	cap.HRU = uint64(c.HRU)
}

// Add adds a new for capacity
func (cap *capacity) add(c capacity) {
	cap.CRU += c.CRU
	cap.MRU += c.MRU
	cap.SRU += c.SRU
	cap.HRU += c.HRU
}

// Subtract subtracts a new capacity
func (cap *capacity) subtract(c capacity) (result capacity) {
	result.CRU = subtract(cap.CRU, c.CRU)
	result.MRU = subtract(cap.MRU, c.MRU)
	result.SRU = subtract(cap.SRU, c.SRU)
	result.HRU = subtract(cap.HRU, c.HRU)
	return result
}

func subtract(a uint64, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return 0
}
