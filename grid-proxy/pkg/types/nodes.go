package types

import "github.com/threefoldtech/zosbase/pkg/gridtypes"

// Location represent the geographic info about the node
type Location struct {
	Country   string   `json:"country"`
	City      string   `json:"city"`
	Region    string   `json:"region"`
	Longitude *float64 `json:"longitude"`
	Latitude  *float64 `json:"latitude"`
}

// NodePower struct is the farmerbot report for node status
type NodePower struct {
	State  string `json:"state"`
	Target string `json:"target"`
}

// Node is a struct holding the data for a Node for the nodes view
type Node struct {
	ID                string       `json:"id"`
	NodeID            int          `json:"nodeId" sort:"node_id"`
	FarmID            int          `json:"farmId" sort:"farm_id"`
	FarmName          string       `json:"farmName"`
	TwinID            int          `json:"twinId" sort:"twin_id"`
	Country           string       `json:"country" sort:"country"`
	GridVersion       int          `json:"gridVersion"`
	City              string       `json:"city" sort:"city"`
	Uptime            int64        `json:"uptime" sort:"uptime"`
	Created           int64        `json:"created" sort:"created"`
	FarmingPolicyID   int          `json:"farmingPolicyId"`
	UpdatedAt         int64        `json:"updatedAt" sort:"updated_at"`
	TotalResources    Capacity     `json:"total_resources" sort:"total_"`
	UsedResources     Capacity     `json:"used_resources" sort:"used_"`
	Location          Location     `json:"location"`
	PublicConfig      PublicConfig `json:"publicConfig"`
	Status            string       `json:"status" sort:"status"`
	CertificationType string       `json:"certificationType"`
	Dedicated         bool         `json:"dedicated"`
	InDedicatedFarm   bool         `json:"inDedicatedFarm" sort:"dedicated_farm"`
	RentContractID    uint         `json:"rentContractId" sort:"rent_contract_id"`
	Rented            bool         `json:"rented" sort:"rented"`
	Rentable          bool         `json:"rentable" sort:"rentable"`
	RentedByTwinID    uint         `json:"rentedByTwinId"`
	SerialNumber      string       `json:"serialNumber"`
	Power             NodePower    `json:"power"`
	NumGPU            int          `json:"num_gpu" sort:"num_gpu"`
	ExtraFee          uint64       `json:"extraFee" sort:"extra_fee"`
	Healthy           bool         `json:"healthy"`
	Dmi               Dmi          `json:"dmi"`
	Speed             Speed        `json:"speed"`
	CpuBenchmark      CpuBenchmark `json:"cpu_benchmark"`
	GPUs              []NodeGPU    `json:"gpus"`
	PriceUsd          float64      `json:"price_usd" sort:"price_usd"`
	FarmFreeIps       uint         `json:"farm_free_ips"`
	Features          []string     `json:"features"`
	_                 string       `sort:"free_cru"`
}

// CapacityResult is the NodeData capacity results to unmarshal json in it
type CapacityResult struct {
	Total Capacity `json:"total_resources"`
	Used  Capacity `json:"used_resources"`
}

// Node to be compatible with old view
type NodeWithNestedCapacity struct {
	ID                string         `json:"id"`
	NodeID            int            `json:"nodeId"`
	FarmID            int            `json:"farmId"`
	FarmName          string         `json:"farmName"`
	TwinID            int            `json:"twinId"`
	Country           string         `json:"country"`
	GridVersion       int            `json:"gridVersion"`
	City              string         `json:"city"`
	Uptime            int64          `json:"uptime"`
	Created           int64          `json:"created"`
	FarmingPolicyID   int            `json:"farmingPolicyId"`
	UpdatedAt         int64          `json:"updatedAt"`
	Capacity          CapacityResult `json:"capacity"`
	Location          Location       `json:"location"`
	PublicConfig      PublicConfig   `json:"publicConfig"`
	Status            string         `json:"status"` // added node status field for up or down
	CertificationType string         `json:"certificationType"`
	Dedicated         bool           `json:"dedicated"`
	InDedicatedFarm   bool           `json:"inDedicatedFarm"`
	RentContractID    uint           `json:"rentContractId"`
	RentedByTwinID    uint           `json:"rentedByTwinId"`
	Rented            bool           `json:"rented"`
	Rentable          bool           `json:"rentable"`
	SerialNumber      string         `json:"serialNumber"`
	Power             NodePower      `json:"power"`
	NumGPU            int            `json:"num_gpu"`
	ExtraFee          uint64         `json:"extraFee"`
	Healthy           bool           `json:"healthy"`
	Dmi               Dmi            `json:"dmi"`
	Speed             Speed          `json:"speed"`
	CpuBenchmark      CpuBenchmark   `json:"cpu_benchmark"`
	GPUs              []NodeGPU      `json:"gpus"`
	PriceUsd          float64        `json:"price_usd"`
	FarmFreeIps       uint           `json:"farm_free_ips"`
	Features          []string       `json:"features"`
}

// PublicConfig node public config
type PublicConfig struct {
	Domain string `json:"domain"`
	Gw4    string `json:"gw4"`
	Gw6    string `json:"gw6"`
	Ipv4   string `json:"ipv4"`
	Ipv6   string `json:"ipv6"`
}

// Capacity is the resources needed for workload(cpu, memory, SSD disk, HDD disks)
type Capacity struct {
	CRU uint64         `json:"cru"`
	SRU gridtypes.Unit `json:"sru"`
	HRU gridtypes.Unit `json:"hru"`
	MRU gridtypes.Unit `json:"mru"`
}

// NodeFilter node filters
type NodeFilter struct {
	Status             []string `schema:"status,omitempty"`
	FreeMRU            *uint64  `schema:"free_mru,omitempty"`
	FreeHRU            *uint64  `schema:"free_hru,omitempty"`
	FreeSRU            *uint64  `schema:"free_sru,omitempty"`
	TotalMRU           *uint64  `schema:"total_mru,omitempty"`
	TotalHRU           *uint64  `schema:"total_hru,omitempty"`
	TotalSRU           *uint64  `schema:"total_sru,omitempty"`
	TotalCRU           *uint64  `schema:"total_cru,omitempty"`
	Country            *string  `schema:"country,omitempty"`
	CountryContains    *string  `schema:"country_contains,omitempty"`
	City               *string  `schema:"city,omitempty"`
	CityContains       *string  `schema:"city_contains,omitempty"`
	Region             *string  `schema:"region,omitempty"`
	FarmName           *string  `schema:"farm_name,omitempty"`
	FarmNameContains   *string  `schema:"farm_name_contains,omitempty"`
	FarmIDs            []uint64 `schema:"farm_ids,omitempty"`
	FreeIPs            *uint64  `schema:"free_ips,omitempty"`
	IPv4               *bool    `schema:"ipv4,omitempty"`
	IPv6               *bool    `schema:"ipv6,omitempty"`
	Domain             *bool    `schema:"domain,omitempty"`
	Dedicated          *bool    `schema:"dedicated,omitempty"`
	InDedicatedFarm    *bool    `schema:"in_dedicated_farm,omitempty"`
	Rentable           *bool    `schema:"rentable,omitempty"`
	OwnedBy            *uint64  `schema:"owned_by,omitempty"`
	Rented             *bool    `schema:"rented,omitempty"`
	RentedBy           *uint64  `schema:"rented_by,omitempty"`
	RentableOrRentedBy *uint64  `schema:"rentable_or_rented_by,omitempty"` // rented by twin or rentable
	AvailableFor       *uint64  `schema:"available_for,omitempty"`         // rented by twin or free
	NodeID             *uint64  `schema:"node_id,omitempty"`
	NodeIDs            []uint64 `schema:"node_ids,omitempty"`
	TwinID             *uint64  `schema:"twin_id,omitempty"`
	CertificationType  *string  `schema:"certification_type,omitempty"`
	HasGPU             *bool    `schema:"has_gpu,omitempty"`
	NumGPU             *uint64  `schema:"num_gpu,omitempty"`
	GpuDeviceID        *string  `schema:"gpu_device_id,omitempty"`
	GpuDeviceName      *string  `schema:"gpu_device_name,omitempty"`
	GpuVendorID        *string  `schema:"gpu_vendor_id,omitempty"`
	GpuVendorName      *string  `schema:"gpu_vendor_name,omitempty"`
	GpuAvailable       *bool    `schema:"gpu_available,omitempty"`
	Healthy            *bool    `schema:"healthy,omitempty"`
	PriceMin           *float64 `schema:"price_min,omitempty"`
	PriceMax           *float64 `schema:"price_max,omitempty"`
	Excluded           []uint64 `schema:"excluded,omitempty"`
	HasIpv6            *bool    `schema:"has_ipv6,omitempty"`
	Features           []string `schema:"features,omitempty"`
}

func (f NodeFilter) Validate() error {
	return validateNodeFeatures(f.Features)
}

func (f NodeFilter) IsGpuFilterRequested() bool {
	return f.HasGPU != nil || f.GpuDeviceName != nil ||
		f.GpuVendorName != nil || f.GpuVendorID != nil ||
		f.GpuDeviceID != nil || f.GpuAvailable != nil
}

// NodeSelect represents fields that can be selected in nodes API response
type NodeSelect struct {
	ID                bool `schema:"id"`
	NodeID            bool `schema:"node_id"`
	FarmID            bool `schema:"farm_id"`
	FarmName          bool `schema:"farm_name"`
	TwinID            bool `schema:"twin_id"`
	Country           bool `schema:"country"`
	GridVersion       bool `schema:"grid_version"`
	City              bool `schema:"city"`
	Uptime            bool `schema:"uptime"`
	Created           bool `schema:"created"`
	FarmingPolicyID   bool `schema:"farming_policy_id"`
	UpdatedAt         bool `schema:"updated_at"`
	TotalResources    bool `schema:"total_resources"`
	UsedResources     bool `schema:"used_resources"`
	Location          bool `schema:"location"`
	PublicConfig      bool `schema:"public_config"`
	Status            bool `schema:"status"`
	CertificationType bool `schema:"certification_type"`
	Dedicated         bool `schema:"dedicated"`
	InDedicatedFarm   bool `schema:"in_dedicated_farm"`
	RentContractID    bool `schema:"rent_contract_id"`
	Rented            bool `schema:"rented"`
	Rentable          bool `schema:"rentable"`
	RentedByTwinID    bool `schema:"rented_by_twin_id"`
	SerialNumber      bool `schema:"serial_number"`
	Power             bool `schema:"power"`
	NumGPU            bool `schema:"num_gpu"`
	ExtraFee          bool `schema:"extra_fee"`
	Healthy           bool `schema:"healthy"`
	Dmi               bool `schema:"dmi"`
	Speed             bool `schema:"speed"`
	CpuBenchmark      bool `schema:"cpu_benchmark"`
	GPUs              bool `schema:"gpus"`
	PriceUsd          bool `schema:"price_usd"`
	FarmFreeIps       bool `schema:"farm_free_ips"`
	Features          bool `schema:"features"`
}

// HasSelection returns true if any field is selected
func (ns NodeSelect) HasSelection() bool {
	return ns.ID || ns.NodeID || ns.FarmID || ns.FarmName || ns.TwinID ||
		ns.Country || ns.GridVersion || ns.City || ns.Uptime || ns.Created ||
		ns.FarmingPolicyID || ns.UpdatedAt || ns.TotalResources || ns.UsedResources ||
		ns.Location || ns.PublicConfig || ns.Status || ns.CertificationType ||
		ns.Dedicated || ns.InDedicatedFarm || ns.RentContractID || ns.Rented ||
		ns.Rentable || ns.RentedByTwinID || ns.SerialNumber || ns.Power ||
		ns.NumGPU || ns.ExtraFee || ns.Healthy || ns.Dmi || ns.Speed ||
		ns.CpuBenchmark || ns.GPUs || ns.PriceUsd || ns.FarmFreeIps || ns.Features
}
