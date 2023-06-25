package types

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// ContractBilling is contract billing info
type ContractBilling struct {
	AmountBilled     uint64 `json:"amount_billed"`
	DiscountReceived string `json:"discount_received"`
	Timestamp        uint64 `json:"timestamp"`
}

// Counters contains aggregate info about the grid
type Counters struct {
	Nodes             uint64            `json:"nodes"`
	Farms             uint64            `json:"farms"`
	Countries         uint64            `json:"countries"`
	TotalCRU          uint64            `json:"total_cru"`
	TotalSRU          uint64            `json:"total_sru"`
	TotalMRU          uint64            `json:"total_mru"`
	TotalHRU          uint64            `json:"total_hru"`
	PublicIPs         uint64            `json:"public_ips"`
	AccessNodes       uint64            `json:"access_nodes"`
	Gateways          uint64            `json:"gateways"`
	Twins             uint64            `json:"twins"`
	Contracts         uint64            `json:"contracts"`
	NodesDistribution map[string]uint64 `json:"nodes_distribution" gorm:"-:all"`
	GPUs              uint64            `json:"gpus"`
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
type Farm struct {
	Name              string     `json:"name"`
	FarmID            uint32     `json:"farm_id"`
	TwinID            uint32     `json:"twin_id"`
	PricingPolicyID   uint32     `json:"pricing_policy_id"`
	CertificationType string     `json:"certification_type"`
	StellarAddress    string     `json:"stellar_address"`
	Dedicated         bool       `json:"dedicated"`
	PublicIps         []PublicIP `json:"public_ips"`
}

// PublicIP info about public ip in the farm
type PublicIP struct {
	ID         string `json:"id"`
	IP         string `json:"ip"`
	FarmID     string `json:"farm_id"`
	ContractID uint64 `json:"contract_id"`
	Gateway    string `json:"gateway"`
}

// StatsFilter statistics filters
type StatsFilter struct {
	Status *string `json:"status"`
}

// Limit used for pagination
type Limit struct {
	Size      uint64 `json:"size"`
	Page      uint64 `json:"page"`
	RetCount  bool   `json:"ret_count"`
	Randomize bool   `json:"randomize"`
}

// NodeFilter node filters
type NodeFilter struct {
	Status            *string  `json:"status"`
	FreeMRU           *uint64  `json:"free_mru"`
	FreeHRU           *uint64  `json:"free_hru"`
	FreeSRU           *uint64  `json:"free_sru"`
	TotalMRU          *uint64  `json:"total_mru"`
	TotalHRU          *uint64  `json:"total_hru"`
	TotalSRU          *uint64  `json:"total_sru"`
	TotalCRU          *uint64  `json:"total_cru"`
	Country           *string  `json:"country"`
	CountryContains   *string  `json:"country_contains"`
	City              *string  `json:"city"`
	CityContains      *string  `json:"city_contains"`
	FarmName          *string  `json:"farm_name"`
	FarmNameContains  *string  `json:"farm_name_contains"`
	FarmIDs           []uint32 `json:"farm_ids"`
	FreeIPs           *uint64  `json:"free_ips"`
	IPv4              *bool    `json:"ipv4"`
	IPv6              *bool    `json:"ipv6"`
	Domain            *bool    `json:"domain"`
	Dedicated         *bool    `json:"dedicated"`
	Rentable          *bool    `json:"rentable"`
	Rented            *bool    `json:"rented"`
	RentedBy          *uint32  `json:"rented_by"`
	AvailableFor      *uint32  `json:"available_for"`
	NodeID            *uint32  `json:"node_id"`
	TwinID            *uint32  `json:"twin_id"`
	CertificationType *string  `json:"certification_type"`
	HasGPU            *bool    `json:"has_gpu"`
}

// FarmFilter farm filters
type FarmFilter struct {
	FreeIPs           *uint64 `json:"free_ips"`
	TotalIPs          *uint64 `json:"total_ips"`
	StellarAddress    *string `json:"stellar_address"`
	PricingPolicyID   *uint32 `json:"pricing_policy_id"`
	FarmID            *uint32 `json:"farm_id"`
	TwinID            *uint32 `json:"twin_id"`
	Name              *string `json:"name"`
	NameContains      *string `json:"name_contains"`
	CertificationType *string `json:"certification_type"`
	Dedicated         *bool   `json:"dedicated"`
	NodeFreeMRU       *uint64 `json:"node_free_mru"`
	NodeFreeHRU       *uint64 `json:"node_free_hru"`
	NodeFreeSRU       *uint64 `json:"node_free_sru"`
}

// TwinFilter twin filters
type TwinFilter struct {
	TwinID    *uint32 `json:"twin_id"`
	AccountID *string `json:"account_id"`
	Relay     *string `json:"relay"`
	PublicKey *string `json:"public_key"`
}

// ContractFilter contract filters
type ContractFilter struct {
	ContractID        *uint64 `json:"contract_id"`
	TwinID            *uint32 `json:"twin_id"`
	NodeID            *uint32 `json:"node_id"`
	Type              *string `json:"type"`
	State             *string `json:"state"`
	Name              *string `json:"name"`
	NumberOfPublicIps *uint64 `json:"number_of_public_ips"`
	DeploymentData    *string `json:"deployment_data"`
	DeploymentHash    *string `json:"deployment_hash"`
}

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
	NodeID            uint32       `json:"node_id"`
	FarmID            uint32       `json:"farm_id"`
	TwinID            uint32       `json:"twin_id"`
	Country           string       `json:"country"`
	GridVersion       uint32       `json:"grid_version"`
	City              string       `json:"city"`
	Uptime            uint64       `json:"uptime"`
	Created           uint64       `json:"created"`
	FarmingPolicyID   uint32       `json:"farmin_policy_id"`
	UpdatedAt         int64        `json:"updated_at"`
	TotalResources    Capacity     `json:"total_resources"`
	UsedResources     Capacity     `json:"used_resources"`
	Location          Location     `json:"location"`
	PublicConfig      PublicConfig `json:"public_config"`
	Status            string       `json:"status"` // added node status field for up or down
	CertificationType string       `json:"certification_type"`
	Dedicated         bool         `json:"dedicated"`
	RentContractID    uint64       `json:"rent_contract_id"`
	RentedByTwinID    uint32       `json:"rented_by_twin_id"`
	SerialNumber      string       `json:"serial_number"`
	Power             NodePower    `json:"power"`
	NumGPU            uint8        `json:"num_gpu"`
	ExtraFee          uint64       `json:"extra_fee"`
}

// CapacityResult is the NodeData capacity results to unmarshal json in it
type CapacityResult struct {
	Total Capacity `json:"total_resources"`
	Used  Capacity `json:"used_resources"`
}

// Node to be compatible with old view
type NodeWithNestedCapacity struct {
	ID                string         `json:"id"`
	NodeID            uint32         `json:"node_id"`
	FarmID            uint32         `json:"farm_id"`
	TwinID            uint32         `json:"twin_id"`
	Country           string         `json:"country"`
	GridVersion       uint32         `json:"grid_version"`
	City              string         `json:"city"`
	Uptime            uint64         `json:"uptime"`
	Created           uint64         `json:"created"`
	FarmingPolicyID   uint32         `json:"farmin_policy_id"`
	UpdatedAt         int64          `json:"updated_at"`
	Capacity          CapacityResult `json:"capacity"`
	Location          Location       `json:"location"`
	PublicConfig      PublicConfig   `json:"public_config"`
	Status            string         `json:"status"` // added node status field for up or down
	CertificationType string         `json:"certification_type"`
	Dedicated         bool           `json:"dedicated"`
	RentContractID    uint64         `json:"rent_contract_id"`
	RentedByTwinID    uint32         `json:"rented_by_twin_id"`
	SerialNumber      string         `json:"serial_number"`
	Power             NodePower      `json:"power"`
	NumGPU            uint8          `json:"num_gpu"`
	ExtraFee          uint64         `json:"extra_fee"`
}

type Twin struct {
	TwinID    uint32 `json:"twin_id"`
	AccountID string `json:"account_id"`
	Relay     string `json:"relay"`
	PublicKey string `json:"public_key"`
}

type NodeContractDetails struct {
	NodeID            uint32 `json:"node_id"`
	DeploymentData    string `json:"deployment_data"`
	DeploymentHash    string `json:"deployment_hash"`
	NumberOfPublicIps uint64 `json:"number_of_public_ips"`
}

type NameContractDetails struct {
	Name string `json:"name"`
}

type RentContractDetails struct {
	NodeID uint32 `json:"node_id"`
}

type Contract struct {
	ContractID uint64            `json:"contract_id"`
	TwinID     uint32            `json:"twin_id"`
	State      string            `json:"state"`
	CreatedAt  int64             `json:"created_at"`
	Type       string            `json:"type"`
	Details    interface{}       `json:"details"`
	Billing    []ContractBilling `json:"billing"`
}

type Version struct {
	Version string `json:"version"`
}

type NodeStatisticsResources struct {
	CRU   uint64 `json:"cru"`
	HRU   uint64 `json:"hru"`
	IPV4U uint64 `json:"ipv4u"`
	MRU   uint64 `json:"mru"`
	SRU   uint64 `json:"sru"`
}

type NodeStatisticsUsers struct {
	Deployments uint64 `json:"deployments"`
	Workloads   uint64 `json:"workloads"`
}

type NodeStatistics struct {
	System NodeStatisticsResources `json:"system"`
	Total  NodeStatisticsResources `json:"total"`
	Used   NodeStatisticsResources `json:"used"`
	Users  NodeStatisticsUsers     `json:"users"`
}

// NodeStatus is used for status endpoint to decode json in
type NodeStatus struct {
	Status string `json:"status"`
}

type NodeGPU struct {
	ID     string `json:"id"`
	Vendor string `json:"vendor"`
	Device string `json:"device"`
}

// Serialize is the serializer for node status struct
func (n *NodeStatus) Serialize() (json.RawMessage, error) {
	bytes, err := json.Marshal(n)
	if err != nil {
		return json.RawMessage{}, errors.Wrap(err, "failed to serialize json data for node status struct")
	}
	return json.RawMessage(bytes), nil
}

// Deserialize is the deserializer for node status struct
func (n *NodeStatus) Deserialize(data []byte) error {
	err := json.Unmarshal(data, n)
	if err != nil {
		return errors.Wrap(err, "failed to deserialize json data for node status struct")
	}
	return nil
}
