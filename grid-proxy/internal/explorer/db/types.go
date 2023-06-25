package db

import (
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// Database interface for storing and fetching grid info
type Database interface {
	GetCounters(filter types.StatsFilter) (types.Counters, error)
	GetNode(nodeID uint32) (Node, error)
	GetFarm(farmID uint32) (Farm, error)
	GetNodes(filter types.NodeFilter, limit types.Limit) ([]Node, uint, error)
	GetFarms(filter types.FarmFilter, limit types.Limit) ([]Farm, uint, error)
	GetTwins(filter types.TwinFilter, limit types.Limit) ([]types.Twin, uint, error)
	GetContracts(filter types.ContractFilter, limit types.Limit) ([]DBContract, uint, error)
}

// DBContract is contract info
type DBContract struct {
	ContractID        uint64
	TwinID            uint32
	State             string
	CreatedAt         uint64
	Name              string
	NodeID            uint32
	DeploymentData    string
	DeploymentHash    string
	NumberOfPublicIps uint64
	Type              string
	ContractBillings  string
}

// Node data about a node which is calculated from the chain
type Node struct {
	ID              string
	NodeID          uint32
	FarmID          uint32
	TwinID          uint32
	Country         string
	GridVersion     uint32
	City            string
	Uptime          uint64
	Created         uint64
	FarmingPolicyID uint32
	UpdatedAt       uint64
	TotalCru        uint64
	TotalMru        uint64
	TotalSru        uint64
	TotalHru        uint64
	UsedCru         uint64
	UsedMru         uint64
	UsedSru         uint64
	UsedHru         uint64
	Domain          string
	Gw4             string
	Gw6             string
	Ipv4            string
	Ipv6            string
	Certification   string
	Dedicated       bool
	RentContractID  uint64
	RentedByTwinID  uint32
	SerialNumber    string
	Longitude       *float64
	Latitude        *float64
	Power           NodePower `gorm:"type:jsonb"`
	HasGPU          bool
	ExtraFee        uint64
}

// NodePower struct is the farmerbot report for node status
type NodePower struct {
	State  string `json:"state"`
	Target string `json:"target"`
}

// Farm data about a farm which is calculated from the chain
type Farm struct {
	Name            string
	FarmID          uint32
	TwinID          uint32
	PricingPolicyID uint32
	Certification   string
	StellarAddress  string
	Dedicated       bool
	PublicIps       string
}

// NodesDistribution is the number of nodes per each country
type NodesDistribution struct {
	Country string `json:"country"`
	Nodes   uint64 `json:"nodes"`
}
