package types

import "github.com/threefoldtech/zos/pkg/gridtypes"

// Location represent the geographic info about the node
type Location struct {
	Country   string   `json:"country"`
	City      string   `json:"city"`
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
	RentedByTwinID    uint         `json:"rentedByTwinId"`
	SerialNumber      string       `json:"serialNumber"`
	Power             NodePower    `json:"power"`
	NumGPU            int          `json:"num_gpu" sort:"num_gpu"`
	ExtraFee          uint64       `json:"extraFee" sort:"extra_fee"`
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
	SerialNumber      string         `json:"serialNumber"`
	Power             NodePower      `json:"power"`
	NumGPU            int            `json:"num_gpu"`
	ExtraFee          uint64         `json:"extraFee"`
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
	Status            *string  `schema:"status,omitempty"`
	FreeMRU           *uint64  `schema:"free_mru,omitempty"`
	FreeHRU           *uint64  `schema:"free_hru,omitempty"`
	FreeSRU           *uint64  `schema:"free_sru,omitempty"`
	TotalMRU          *uint64  `schema:"total_mru,omitempty"`
	TotalHRU          *uint64  `schema:"total_hru,omitempty"`
	TotalSRU          *uint64  `schema:"total_sru,omitempty"`
	TotalCRU          *uint64  `schema:"total_cru,omitempty"`
	Country           *string  `schema:"country,omitempty"`
	CountryContains   *string  `schema:"country_contains,omitempty"`
	City              *string  `schema:"city,omitempty"`
	CityContains      *string  `schema:"city_contains,omitempty"`
	Region            *string  `schema:"region,omitempty"`
	FarmName          *string  `schema:"farm_name,omitempty"`
	FarmNameContains  *string  `schema:"farm_name_contains,omitempty"`
	FarmIDs           []uint64 `schema:"farm_ids,omitempty"`
	FreeIPs           *uint64  `schema:"free_ips,omitempty"`
	IPv4              *bool    `schema:"ipv4,omitempty"`
	IPv6              *bool    `schema:"ipv6,omitempty"`
	Domain            *bool    `schema:"domain,omitempty"`
	Dedicated         *bool    `schema:"dedicated,omitempty"`
	InDedicatedFarm   *bool    `schema:"in_dedicated_farm,omitempty"`
	Rentable          *bool    `schema:"rentable,omitempty"`
	OwnedBy           *uint64  `schema:"owned_by,omitempty"`
	Rented            *bool    `schema:"rented,omitempty"`
	RentedBy          *uint64  `schema:"rented_by,omitempty"`
	AvailableFor      *uint64  `schema:"available_for,omitempty"`
	NodeID            *uint64  `schema:"node_id,omitempty"`
	TwinID            *uint64  `schema:"twin_id,omitempty"`
	CertificationType *string  `schema:"certification_type,omitempty"`
	HasGPU            *bool    `schema:"has_gpu,omitempty"`
	GpuDeviceID       *string  `schema:"gpu_device_id,omitempty"`
	GpuDeviceName     *string  `schema:"gpu_device_name,omitempty"`
	GpuVendorID       *string  `schema:"gpu_vendor_id,omitempty"`
	GpuVendorName     *string  `schema:"gpu_vendor_name,omitempty"`
	GpuAvailable      *bool    `schema:"gpu_available,omitempty"`
}

// NodeGPU holds the info about gpu card
type NodeGPU struct {
	NodeTwinID uint32 `json:"node_twin_id"`
	ID         string `json:"id"`
	Vendor     string `json:"vendor"`
	Device     string `json:"device"`
	Contract   int    `json:"contract"`
}
