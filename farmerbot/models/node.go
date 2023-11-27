package models

import (
	"math"
	"time"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// Node represents a node type
type Node struct {
	// TODO: substrate.Node
	ID                        uint32              `json:"id" yaml:"id" toml:"id"`
	TwinID                    uint32              `json:"twin_id" yaml:"twin_id" toml:"twin_id"`
	FarmID                    uint32              `json:"farm_id,omitempty" yaml:"farm_id,omitempty" toml:"farm_id,omitempty"`
	Description               string              `json:"description,omitempty" yaml:"description,omitempty" toml:"description,omitempty"`
	Certified                 bool                `json:"certified,omitempty" yaml:"certified,omitempty" toml:"certified,omitempty"`
	Dedicated                 bool                `json:"dedicated,omitempty" yaml:"dedicated,omitempty" toml:"dedicated,omitempty"`
	PublicConfig              bool                `json:"public_config,omitempty" yaml:"public_config,omitempty" toml:"public_config,omitempty"`
	PublicIPsUsed             uint64              `json:"public_ips_used,omitempty" yaml:"public_ips_used,omitempty" toml:"public_ips_used,omitempty"`
	Resources                 ConsumableResources `json:"resources" yaml:"resources" toml:"resources"`
	Pools                     []pkg.PoolMetrics   `json:"pools,omitempty" yaml:"pools,omitempty" toml:"pools,omitempty"`
	GPUs                      []GPU               `json:"gpus,omitempty" yaml:"gpus,omitempty" toml:"gpus,omitempty"`
	HasActiveRentContract     bool                `json:"has_active_rent_contract,omitempty" yaml:"has_active_rent_contract,omitempty" toml:"has_active_rent_contract,omitempty"`
	PowerState                PowerState          `json:"power_state,omitempty" yaml:"power_state,omitempty" toml:"power_state,omitempty"`
	TimeoutClaimedResources   time.Time           `json:"timeout_claimed_resources,omitempty" yaml:"timeout_claimed_resources,omitempty" toml:"timeout_claimed_resources,omitempty"`
	LastTimePowerStateChanged time.Time           `json:"last_time_power_state_changed,omitempty" yaml:"last_time_power_state_changed,omitempty" toml:"last_time_power_state_changed,omitempty"`
	LastTimeAwake             time.Time           `json:"last_time_awake,omitempty" yaml:"last_time_awake,omitempty" toml:"last_time_awake,omitempty"`
	NeverShutDown             bool                `json:"never_shutdown,omitempty" yaml:"never_shutdown,omitempty" toml:"never_shutdown,omitempty"`
	TimesRandomWakeUps        int                 `json:"times_random_wake_ups,omitempty" yaml:"times_random_wake_ups,omitempty" toml:"times_random_wake_ups,omitempty"`
}

// NodeOptions represents the options to find a node
type NodeOptions struct {
	NodeExclude  []uint32 `json:"node_exclude,omitempty" yaml:"node_exclude,omitempty" toml:"node_exclude,omitempty"`
	HasGPUs      uint8    `json:"has_gpus,omitempty" yaml:"has_gpus,omitempty" toml:"has_gpus,omitempty"`
	GPUVendors   []string `json:"gpu_vendors,omitempty" yaml:"gpu_vendors,omitempty" toml:"gpu_vendors,omitempty"`
	GPUDevices   []string `json:"gpu_devices,omitempty" yaml:"gpu_devices,omitempty" toml:"gpu_devices,omitempty"`
	Certified    bool     `json:"certified,omitempty" yaml:"certified,omitempty" toml:"certified,omitempty"`
	Dedicated    bool     `json:"dedicated,omitempty" yaml:"dedicated,omitempty" toml:"dedicated,omitempty"`
	PublicConfig bool     `json:"public_config,omitempty" yaml:"public_config,omitempty" toml:"public_config,omitempty"`
	PublicIPs    uint64   `json:"public_ips,omitempty" yaml:"public_ips,omitempty" toml:"public_ips,omitempty"`
	Capacity     Capacity `json:"capacity,omitempty" yaml:"capacity,omitempty" toml:"capacity,omitempty"`
}

// GPU information
type GPU struct {
	ID       string `json:"id" yaml:"id" toml:"id"`
	Vendor   string `json:"vendor" yaml:"vendor" toml:"vendor"`
	Device   string `json:"device" yaml:"device" toml:"device"`
	Contract uint64 `json:"contract" yaml:"contract" toml:"contract"`
}

type ZosResourcesStatistics struct {
	// Total system capacity
	Total gridtypes.Capacity `json:"total"`
	// Used capacity this include user + system resources
	Used gridtypes.Capacity `json:"used"`
	// System resource reserved by zos
	System gridtypes.Capacity `json:"system"`
}

// UpdateResources updates the node resources from zos resources stats
func (n *Node) UpdateResources(stats ZosResourcesStatistics) {
	n.Resources.Total.update(stats.Total)
	n.Resources.Used.update(stats.Used)
	n.Resources.System.update(stats.System)
	n.PublicIPsUsed = stats.Used.IPV4U
}

// IsUnused node is an empty node
func (n *Node) IsUnused() bool {
	resources := n.Resources.Used.subtract(n.Resources.System)
	return resources.isEmpty() && !n.HasActiveRentContract
}

// CanClaimResources checks if a node can claim some resources
func (n *Node) CanClaimResources(cap Capacity) bool {
	free := n.freeCapacity()
	return n.Resources.Total.CRU >= cap.CRU && free.CRU >= cap.CRU && free.MRU >= cap.MRU && free.HRU >= cap.HRU && free.SRU >= cap.SRU
}

// ClaimResources claims the resources from a node
func (n *Node) ClaimResources(c Capacity) {
	n.Resources.Used.Add(c)
}

// FreeCapacity calculates the free capacity of a node
func (n *Node) freeCapacity() Capacity {
	total := n.Resources.Total
	total.CRU = uint64(math.Ceil(float64(total.CRU) * float64(n.Resources.OverProvisionCPU)))
	return total.subtract(n.Resources.Used)
}
