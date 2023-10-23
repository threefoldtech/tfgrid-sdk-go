package db

import (
	"context"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// Database interface for storing and fetching grid info
type Database interface {
	GetCounters(ctx context.Context, filter types.StatsFilter) (types.Counters, error)
	GetNode(ctx context.Context, nodeID uint32) (Node, error)
	GetFarm(ctx context.Context, farmID uint32) (Farm, error)
	GetNodes(ctx context.Context, filter types.NodeFilter, limit types.Limit) ([]Node, uint, error)
	GetFarms(ctx context.Context, filter types.FarmFilter, limit types.Limit) ([]Farm, uint, error)
	GetTwins(ctx context.Context, filter types.TwinFilter, limit types.Limit) ([]types.Twin, uint, error)
	GetContracts(ctx context.Context, filter types.ContractFilter, limit types.Limit) ([]DBContract, uint, error)
	GetContract(ctx context.Context, contractID uint32) (DBContract, error)
	GetContractBills(ctx context.Context, contractID uint32, limit types.Limit) ([]ContractBilling, uint, error)
	UpsertNodesGPU(ctx context.Context, nodesGPU []types.NodeGPU) error
}

type ContractBilling types.ContractBilling

// DBContract is contract info
type DBContract struct {
	ContractID        uint
	TwinID            uint
	State             string
	CreatedAt         uint
	Name              string
	NodeID            uint
	DeploymentData    string
	DeploymentHash    string
	NumberOfPublicIps uint
	Type              string
}

// Node data about a node which is calculated from the chain
type Node struct {
	ID              string
	NodeID          int64
	FarmID          int64
	TwinID          int64
	Country         string
	GridVersion     int64
	City            string
	Uptime          int64
	Created         int64
	FarmingPolicyID int64
	UpdatedAt       int64
	TotalCru        int64
	TotalMru        int64
	TotalSru        int64
	TotalHru        int64
	UsedCru         int64
	UsedMru         int64
	UsedSru         int64
	UsedHru         int64
	Domain          string
	Gw4             string
	Gw6             string
	Ipv4            string
	Ipv6            string
	Certification   string
	Dedicated       bool
	RentContractID  int64
	RentedByTwinID  int64
	SerialNumber    string
	Longitude       *float64
	Latitude        *float64
	Power           NodePower `gorm:"type:jsonb"`
	NumGPU          int       `gorm:"num_gpu"`
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
	FarmID          int
	TwinID          int
	PricingPolicyID int
	Certification   string
	StellarAddress  string
	Dedicated       bool
	PublicIps       string
}

// NodesDistribution is the number of nodes per each country
type NodesDistribution struct {
	Country string `json:"country"`
	Nodes   int64  `json:"nodes"`
}

type NodeGPU struct {
	NodeTwinID int    `gorm:"primaryKey;autoIncrement:false"`
	ID         string `gorm:"primaryKey"`
	Vendor     string
	Device     string
	Contract   int
}

func (NodeGPU) TableName() string {
	return "node_gpu"
}
