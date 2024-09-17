package db

import (
	"context"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// Database interface for storing and fetching grid info
type Database interface {
	GetConnectionString() string
	Ping() error
	Initialized() error
	GetRandomHealthyTwinIds(length int) ([]uint32, error)
	GetLastUpsertsTimestamp() (types.IndexersState, error)

	// server getters
	GetStats(ctx context.Context, filter types.StatsFilter) (types.Stats, error)
	GetNode(ctx context.Context, nodeID uint32) (Node, error)
	GetFarm(ctx context.Context, farmID uint32) (Farm, error)
	GetNodes(ctx context.Context, filter types.NodeFilter, limit types.Limit) ([]Node, uint, error)
	GetFarms(ctx context.Context, filter types.FarmFilter, limit types.Limit) ([]Farm, uint, error)
	GetTwins(ctx context.Context, filter types.TwinFilter, limit types.Limit) ([]types.Twin, uint, error)
	GetContracts(ctx context.Context, filter types.ContractFilter, limit types.Limit) ([]DBContract, uint, error)
	GetContract(ctx context.Context, contractID uint32) (DBContract, error)
	GetContractBills(ctx context.Context, contractID uint32, limit types.Limit) ([]ContractBilling, uint, error)
	GetContractsLatestBillReports(ctx context.Context, contractsIds []uint32, limit uint) ([]ContractBilling, error)
	GetContractsTotalBilledAmount(ctx context.Context, contractIds []uint32) (uint64, error)

	// indexer utils
	DeleteOldGpus(ctx context.Context, nodeTwinIds []uint32, expiration int64) error
	GetLastNodeTwinID(ctx context.Context) (uint32, error)
	GetNodeTwinIDsAfter(ctx context.Context, twinID uint32) ([]uint32, error)
	GetHealthyNodeTwinIds(ctx context.Context) ([]uint32, error)

	// indexer upserters
	UpsertNodesGPU(ctx context.Context, gpus []types.NodeGPU) error
	UpsertNodeHealth(ctx context.Context, healthReports []types.HealthReport) error
	UpsertNodeDmi(ctx context.Context, dmis []types.Dmi) error
	UpsertNetworkSpeed(ctx context.Context, speeds []types.Speed) error
	UpsertNodeIpv6Report(ctx context.Context, ips []types.HasIpv6) error
	UpsertNodeWorkloads(ctx context.Context, workloads []types.NodesWorkloads) error
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
	FarmName          string
	FarmId            uint64
}

// Node data about a node which is calculated from the chain
type Node struct {
	ID                 string
	NodeID             int64
	FarmID             int64
	FarmName           string
	TwinID             int64
	Country            string
	GridVersion        int64
	City               string
	Uptime             int64
	Created            int64
	FarmingPolicyID    int64
	UpdatedAt          int64
	TotalCru           int64
	TotalMru           int64
	TotalSru           int64
	TotalHru           int64
	UsedCru            int64
	UsedMru            int64
	UsedSru            int64
	UsedHru            int64
	Domain             string
	Gw4                string
	Gw6                string
	Ipv4               string
	Ipv6               string
	Certification      string
	FarmDedicated      bool `gorm:"farm_dedicated"`
	RentContractID     int64
	Renter             int64
	Rented             bool
	Rentable           bool
	SerialNumber       string
	Longitude          *float64
	Latitude           *float64
	Power              NodePower `gorm:"type:jsonb;serializer:json"`
	NumGPU             int       `gorm:"num_gpu"`
	ExtraFee           uint64
	NodeContractsCount uint64 `gorm:"node_contracts_count"`
	Healthy            bool
	Bios               types.BIOS        `gorm:"type:jsonb;serializer:json"`
	Baseboard          types.Baseboard   `gorm:"type:jsonb;serializer:json"`
	Memory             []types.Memory    `gorm:"type:jsonb;serializer:json"`
	Processor          []types.Processor `gorm:"type:jsonb;serializer:json"`
	Gpus               []types.NodeGPU   `gorm:"type:jsonb;serializer:json"`
	UploadSpeed        float64
	DownloadSpeed      float64
	PriceUsd           float64
	FarmFreeIps        uint
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
