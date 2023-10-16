// nolint
package mock

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// TODO: the one in tools/db/types.go is unexported but it's the same file
type JSON json.RawMessage

type ContractResources struct {
	ID         string `gorm:"column:id"`
	HRU        uint64 `gorm:"column:hru"`
	SRU        uint64 `gorm:"column:sru"`
	CRU        uint64 `gorm:"column:cru"`
	MRU        uint64 `gorm:"column:mru"`
	ContractID string `gorm:"column:contract_id"`
}

func (ContractResources) TableName() string {
	return "contract_resources"
}

type Farm struct {
	ID              string `gorm:"column:id"`
	GridVersion     uint64 `gorm:"column:grid_version"`
	FarmID          uint64 `gorm:"column:farm_id"`
	Name            string `gorm:"column:name"`
	TwinID          uint64 `gorm:"column:twin_id"`
	PricingPolicyID uint64 `gorm:"column:pricing_policy_id"`
	Certification   string `gorm:"column:certification"`
	StellarAddress  string `gorm:"column:stellar_address"`
	DedicatedFarm   bool   `gorm:"column:dedicated_farm"`
}

func (Farm) TableName() string {
	return "farm"
}

type Node struct {
	ID              string    `gorm:"column:id"`
	GridVersion     uint64    `gorm:"column:grid_version"`
	NodeID          uint64    `gorm:"column:node_id"`
	FarmID          uint64    `gorm:"column:farm_id"`
	TwinID          uint64    `gorm:"column:twin_id"`
	Country         string    `gorm:"column:country"`
	City            string    `gorm:"column:city"`
	Uptime          uint64    `gorm:"column:uptime"`
	Created         uint64    `gorm:"column:created"`
	FarmingPolicyID uint64    `gorm:"column:farming_policy_id"`
	Certification   string    `gorm:"column:certification"`
	Secure          bool      `gorm:"column:secure"`
	Virtualized     bool      `gorm:"column:virtualized"`
	SerialNumber    string    `gorm:"column:serial_number"`
	CreatedAt       uint64    `gorm:"column:created_at"`
	UpdatedAt       uint64    `gorm:"column:updated_at"`
	LocationID      string    `gorm:"column:location_id"`
	Power           NodePower `gorm:"type:jsonb"`
	ExtraFee        uint64    `gorm:"column:extra_fee"`
	TotalHRU        uint64    `gorm:"column:total_hru"`
	TotalCRU        uint64    `gorm:"column:total_cru"`
	TotalMRU        uint64    `gorm:"column:total_mru"`
	TotalSRU        uint64    `gorm:"column:total_sru"`
}

func (Node) TableName() string {
	return "node"
}

type NodesCache struct {
	ID             string `gorm:"column:id"`
	NodeID         uint64 `gorm:"column:node_id"`
	NodeTwinID     uint64 `gorm:"column:node_twin_id"`
	FreeHRU        uint64 `gorm:"column:free_hru"`
	FreeMRU        uint64 `gorm:"column:free_mru"`
	FreeSRU        uint64 `gorm:"column:free_sru"`
	FreeCRU        uint64 `gorm:"column:free_cru"`
	Renter         uint64 `gorm:"column:renter"`
	RentContractID uint64 `gorm:"column:rent_contract_id"`
	NodeContracts  uint64 `gorm:"column:node_contracts"`
	FarmID         uint64 `gorm:"column:farm_id"`
	DedicatedFarm  bool   `gorm:"column:dedicated_farm"`
	FreeGPUs       uint64 `gorm:"column:free_gpus"`
}

func (NodesCache) TableName() string {
	return "nodes_cache"
}

type FarmsCache struct {
	ID       string `gorm:"column:id"`
	FarmID   uint64 `gorm:"column:farm_id"`
	FreeIPs  uint64 `gorm:"column:free_ips"`
	TotalIPs uint64 `gorm:"column:total_ips"`
	IPs      string `gorm:"column:ips"`
}

func (FarmsCache) TableName() string {
	return "farms_cache"
}

type NodePower struct {
	State  string `json:"state"`
	Target string `json:"target"`
}

// Scan is a custom decoder for jsonb filed. executed while scanning the node.
func (np *NodePower) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	if data, ok := value.([]byte); ok {
		return json.Unmarshal(data, np)
	}
	return fmt.Errorf("failed to unmarshal NodePower")
}

func (np NodePower) Value() (driver.Value, error) {
	return json.Marshal(np)
}

type Twin struct {
	ID          string `gorm:"column:id"`
	GridVersion uint64 `gorm:"column:grid_version"`
	TwinID      uint64 `gorm:"column:twin_id"`
	AccountID   string `gorm:"column:account_id"`
	Relay       string `gorm:"column:relay"`
	PublicKey   string `gorm:"column:public_key"`
}

func (Twin) TableName() string {
	return "twin"
}

type PublicIp struct {
	ID         string `gorm:"column:id" json:"id"`
	Gateway    string `gorm:"column:gateway" json:"gateway"`
	IP         string `gorm:"column:ip" json:"ip"`
	ContractID uint64 `gorm:"column:contract_id" json:"contract_id"`
	FarmID     string `gorm:"column:farm_id" json:"farm_id"`
}

func (PublicIp) TableName() string {
	return "public_ip"
}

type GenericContract struct {
	ID                string `json:"id" gorm:"column:id"`
	GridVersion       uint64 `json:"grid_version" gorm:"column:grid_version"`
	ContractID        uint64 `json:"contract_id" gorm:"column:contract_id"`
	TwinID            uint64 `json:"twin_id" gorm:"column:twin_id"`
	NodeID            uint64 `json:"node_id" gorm:"column:node_id"`
	DeploymentData    string `json:"deployment_data" gorm:"column:deployment_data"`
	DeploymentHash    string `json:"deployment_hash" gorm:"column:deployment_hash"`
	NumberOfPublicIPs uint64 `json:"number_of_public_ips" gorm:"column:number_of_public_i_ps"`
	State             string `json:"state" gorm:"column:state"`
	CreatedAt         uint64 `json:"created_at" gorm:"column:created_at"`
	ResourcesUsedID   string `json:"resources_used_id" gorm:"column:resources_used_id"`
	Name              string `json:"name" gorm:"column:name"`
	Type              string `json:"type" gorm:"column:type"`
}

func (GenericContract) TableName() string {
	return "generic_contract"
}

type NodeResourcesTotal struct {
	ID     string `gorm:"column:id"`
	HRU    uint64 `gorm:"column:hru"`
	SRU    uint64 `gorm:"column:sru"`
	CRU    uint64 `gorm:"column:cru"`
	MRU    uint64 `gorm:"column:mru"`
	NodeID string `gorm:"column:node_id"`
}

func (NodeResourcesTotal) TableName() string {
	return "nodes_resources_total"
}

type PublicConfig struct {
	ID     string `gorm:"column:id"`
	IPv4   string `gorm:"column:ipv4"`
	IPv6   string `gorm:"column:ipv6"`
	GW4    string `gorm:"column:gw4"`
	GW6    string `gorm:"column:gw6"`
	Domain string `gorm:"column:domain"`
	NodeID string `gorm:"column:node_id"`
}

func (PublicConfig) TableName() string {
	return "public_config"
}

type ContractBillReport struct {
	ID               string `gorm:"column:id"`
	ContractID       uint64 `gorm:"column:contract_id"`
	DiscountReceived string `gorm:"column:discount_received"`
	AmountBilled     uint64 `gorm:"column:amount_billed"`
	Timestamp        uint64 `gorm:"column:timestamp"`
}

func (ContractBillReport) TableName() string {
	return "contract_bill_report"
}

type NodeGPU struct {
	NodeTwinID uint64 `gorm:"column:node_twin_id"`
	ID         string `gorm:"column:id"`
	Vendor     string `gorm:"column:vendor"`
	Device     string `gorm:"column:device"`
	Contract   int    `gorm:"column:contract"`
}

func (NodeGPU) TableName() string {
	return "node_gpu"
}

type Location struct {
	ID        string `gorm:"column:id"`
	Longitude string `gorm:"column:longitude"`
	Latitude  string `gorm:"column:latitude"`
}

func (Location) TableName() string {
	return "location"
}
