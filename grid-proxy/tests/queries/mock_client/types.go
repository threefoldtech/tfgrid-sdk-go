// nolint
package mock

import (
	"encoding/json"
	"fmt"
)

// TODO: the one in tools/db/types.go is unexported but it's the same file

type ContractResources struct {
	ID         string
	HRU        uint64
	SRU        uint64
	CRU        uint64
	MRU        uint64
	ContractID string
}
type Farm struct {
	ID              string
	GridVersion     uint64
	FarmID          uint64
	Name            string
	TwinID          uint64
	PricingPolicyID uint64
	Certification   string
	StellarAddress  string
	DedicatedFarm   bool
}

type Node struct {
	ID              string
	GridVersion     uint64
	NodeID          uint64
	FarmID          uint64
	TwinID          uint64
	Country         string
	City            string
	Uptime          uint64
	Created         uint64
	FarmingPolicyID uint64
	Certification   string
	Secure          bool
	Virtualized     bool
	SerialNumber    string
	CreatedAt       uint64
	UpdatedAt       uint64
	LocationID      string
	Power           NodePower `gorm:"type:jsonb"`
	HasGPU          bool
	ExtraFee        uint64
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

type Twin struct {
	ID          string
	GridVersion uint64
	TwinID      uint64
	AccountID   string
	Relay       string
	PublicKey   string
}
type PublicIp struct {
	ID         string
	Gateway    string
	IP         string
	ContractID uint64
	FarmID     string
}
type NodeContract struct {
	ID                string
	GridVersion       uint64
	ContractID        uint64
	TwinID            uint64
	NodeID            uint64
	DeploymentData    string
	DeploymentHash    string
	NumberOfPublicIPs uint64
	State             string
	CreatedAt         uint64
	ResourcesUsedID   string
}
type NodeResourcesTotal struct {
	ID     string
	HRU    uint64
	SRU    uint64
	CRU    uint64
	MRU    uint64
	NodeID string
}
type PublicConfig struct {
	ID     string
	IPv4   string
	IPv6   string
	GW4    string
	GW6    string
	Domain string
	NodeID string
}
type RentContract struct {
	ID          string
	GridVersion uint64
	ContractID  uint64
	TwinID      uint64
	NodeID      uint64
	State       string
	CreatedAt   uint64
}

type ContractBillReport struct {
	ID               string
	ContractID       uint64
	DiscountReceived string
	AmountBilled     uint64
	Timestamp        uint64
}

type NameContract struct {
	ID          string
	GridVersion uint64
	ContractID  uint64
	TwinID      uint64
	Name        string
	State       string
	CreatedAt   uint64
}

type NodeGPU struct {
	NodeTwinID uint64
	ID         string
	Vendor     string
	Device     string
	Contract   int
}

type Country struct {
	ID        string
	CountryID uint64
	Code      string
	Name      string
	Region    string
	Subregion string
	Lat       string
	Long      string
}

type Location struct {
	ID        string
	Longitude *float64
	Latitude  *float64
}
