// nolint
package mock

import "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"

// TODO: the one in tools/db/types.go is unexported but it's the same file

var (
	POSTGRES_HOST      string
	POSTGRES_PORT      int
	POSTGRES_USER      string
	POSTGRES_PASSSWORD string
	POSTGRES_DB        string
	ENDPOINT           string
	SEED               int
	STATUS_DOWN        = "down"
	STATUS_UP          = "up"
)

type DBContractResources struct {
	ID         string
	HRU        uint64
	SRU        uint64
	CRU        uint64
	MRU        uint64
	ContractID string
}

type DBFarm struct {
	ID              string
	GridVersion     uint32
	FarmID          uint32
	Name            string
	TwinID          uint32
	PricingPolicyID uint32
	Certification   string
	StellarAddress  string
	DedicatedFarm   bool
}

type DBNode struct {
	ID              string
	GridVersion     uint32
	NodeID          uint32
	FarmID          uint32
	TwinID          uint32
	Country         string
	City            string
	Uptime          uint64
	Created         uint64
	FarmingPolicyID uint32
	Certification   string
	Secure          bool
	Virtualized     bool
	SerialNumber    string
	CreateAt        int64
	UpdatedAt       int64
	LocationID      string
	Power           db.NodePower `gorm:"type:jsonb"`
	HasGPU          bool
	ExtraFee        uint64
}

type DBTwin struct {
	ID          string
	GridVersion uint32
	TwinID      uint32
	AccountID   string
	Relay       string
	PublicKey   string
}
type DBPublicIP struct {
	ID         string
	Gateway    string
	IP         string
	ContractID uint64
	FarmID     string
}
type DBNodeContract struct {
	ID                string
	GridVersion       uint32
	ContractID        uint64
	TwinID            uint32
	NodeID            uint32
	DeploymentData    string
	DeploymentHash    string
	NumberOfPublicIPs uint64
	State             string
	CreatedAt         int64
	ResourcesUsedID   string
}
type DBNodeResourcesTotal struct {
	ID     string
	HRU    uint64
	SRU    uint64
	CRU    uint64
	MRU    uint64
	NodeID string
}
type DBPublicConfig struct {
	ID     string
	IPv4   string
	IPv6   string
	GW4    string
	GW6    string
	Domain string
	NodeID string
}
type DBRentContract struct {
	ID          string
	GridVersion uint32
	ContractID  uint64
	TwinID      uint32
	NodeID      uint32
	State       string
	CreatedAt   int64
}

type DBContractBillReport struct {
	ID               string
	ContractID       uint64
	DiscountReceived string
	AmountBilled     uint64
	Timestamp        uint64
}

type DBNameContract struct {
	ID          string
	GridVersion uint32
	ContractID  uint64
	TwinID      uint32
	Name        string
	State       string
	CreatedAt   int64
}
