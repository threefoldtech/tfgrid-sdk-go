package models

import "github.com/threefoldtech/zos/pkg/gridtypes"

// ConsumableResources for node resources
type ConsumableResources struct {
	OverProvisionCPU float32  `json:"overprovision_cpu,omitempty" yaml:"overprovision_cpu,omitempty" toml:"overprovision_cpu,omitempty"` // how much we allow over provisioning the CPU range: [1;3]
	Total            Capacity `json:"total" yaml:"total" toml:"total"`
	Used             Capacity `json:"used,omitempty" yaml:"used,omitempty" toml:"used,omitempty"`
	System           Capacity `json:"system,omitempty" yaml:"system,omitempty" toml:"system,omitempty"`
}

// Capacity is node resource capacity
type Capacity struct {
	HRU uint64 `json:"HRU"`
	SRU uint64 `json:"SRU"`
	CRU uint64 `json:"CRU"`
	MRU uint64 `json:"MRU"`
}

// IsEmpty checks empty capacity
func (cap *Capacity) isEmpty() bool {
	return cap.CRU == 0 && cap.MRU == 0 && cap.SRU == 0 && cap.HRU == 0
}

func (cap *Capacity) update(c gridtypes.Capacity) {
	cap.CRU = c.CRU
	cap.MRU = uint64(c.MRU)
	cap.SRU = uint64(c.SRU)
	cap.HRU = uint64(c.HRU)
}

// Add adds a new for capacity
func (cap *Capacity) Add(c Capacity) {
	cap.CRU += c.CRU
	cap.MRU += c.MRU
	cap.SRU += c.SRU
	cap.HRU += c.HRU
}

// Subtract subtracts a new capacity
func (cap *Capacity) subtract(c Capacity) (result Capacity) {
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
