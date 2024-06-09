package types

// NodeGpu holds info for a node gpu
// used as both gorm model and server json response
type NodeGPU struct {
	NodeTwinID uint32 `gorm:"uniqueIndex:uni_gpu_node_twin_id" json:"node_twin_id,omitempty" `
	ID         string `gorm:"uniqueIndex:uni_gpu_node_twin_id" json:"id"`
	Vendor     string `json:"vendor"`
	Device     string `json:"device"`
	Contract   int    `json:"contract"`
	UpdatedAt  int64  `json:"updated_at"`
}

func (NodeGPU) TableName() string {
	return "node_gpu"
}

// HealthReport holds the state of node healthiness
// used as gorm model
type HealthReport struct {
	NodeTwinId uint32 `gorm:"unique;not null"`
	Healthy    bool
	UpdatedAt  int64
}

func (HealthReport) TableName() string {
	return "health_report"
}

// HasIpv6 holds the state of node having ipv6
// used as gorm model
type HasIpv6 struct {
	NodeTwinId uint32 `gorm:"unique;not null"`
	HasIpv6    bool
	UpdatedAt  int64
}

func (HasIpv6) TableName() string {
	return "node_ipv6"
}

// Speed holds upload/download speeds in `bit/sec` for a node
// used as both gorm model and server json response
type Speed struct {
	NodeTwinId uint32  `json:"node_twin_id,omitempty" gorm:"unique;not null"`
	Upload     float64 `json:"upload"`   // in bit/sec
	Download   float64 `json:"download"` // in bit/sec
	UpdatedAt  int64
}

func (Speed) TableName() string {
	return "speed"
}

// NodesWorkloads holds the number of workloads on a node
type NodesWorkloads struct {
	NodeTwinId      uint32 `json:"node_twin_id,omitempty" gorm:"unique;not null"`
	WorkloadsNumber uint32 `json:"workloads_number"`
	UpdatedAt       int64
}

func (NodesWorkloads) TableName() string {
	return "node_workloads"
}

// Dmi holds hardware dmi info for a node
// used as both gorm model and server json response
type Dmi struct {
	NodeTwinId uint32      `json:"node_twin_id,omitempty" gorm:"unique;not null"`
	BIOS       BIOS        `json:"bios" gorm:"type:jsonb;serializer:json"`
	Baseboard  Baseboard   `json:"baseboard" gorm:"type:jsonb;serializer:json"`
	Processor  []Processor `json:"processor" gorm:"type:jsonb;serializer:json"`
	Memory     []Memory    `json:"memory" gorm:"type:jsonb;serializer:json"`
	UpdatedAt  int64
}

func (Dmi) TableName() string {
	return "dmi"
}

type BIOS struct {
	Vendor  string `json:"vendor"`
	Version string `json:"version"`
}

type Baseboard struct {
	Manufacturer string `json:"manufacturer"`
	ProductName  string `json:"product_name"`
}

type Processor struct {
	Version     string `json:"version"`
	ThreadCount string `json:"thread_count"`
}

type Memory struct {
	Manufacturer string `json:"manufacturer"`
	Type         string `json:"type"`
}
