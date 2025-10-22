package types

// NodeGpu holds info for a node gpu
// used as both gorm model and server json response
type NodeGPU struct {
	NodeTwinID uint32 `gorm:"uniqueIndex:uni_gpu_node_twin_id" json:"node_twin_id,omitempty" `
	ID         string `gorm:"uniqueIndex:uni_gpu_node_twin_id" json:"id"`
	Vendor     string `json:"vendor"`
	Device     string `json:"device"`
	Vram       uint64 `json:"vram"`
	Contract   int    `json:"contract"`
	UpdatedAt  int64  `json:"updated_at,omitempty"`
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
	NodeTwinId      uint32  `json:"node_twin_id,omitempty" gorm:"unique;not null;column:node_twin_id"`
	Upload          float64 `json:"upload" gorm:"column:upload"`                       // in bit/sec let's suppose default is ipv4/tcp
	Download        float64 `json:"download" gorm:"column:download"`                   // in bit/sec
	UDPDownloadIPv4 float64 `json:"udp_download_ipv4" gorm:"column:udp_download_ipv4"` // in bit/sec
	UDPUploadIPv4   float64 `json:"udp_upload_ipv4" gorm:"column:udp_upload_ipv4"`     // in bit/sec
	TCPDownloadIPv6 float64 `json:"tcp_download_ipv6" gorm:"column:tcp_download_ipv6"` // in bit/sec
	TCPUploadIPv6   float64 `json:"tcp_upload_ipv6" gorm:"column:tcp_upload_ipv6"`     // in bit/sec
	UDPDownloadIPv6 float64 `json:"udp_download_ipv6" gorm:"column:udp_download_ipv6"` // in bit/sec
	UDPUploadIPv6   float64 `json:"udp_upload_ipv6" gorm:"column:udp_upload_ipv6"`     // in bit/sec
	UpdatedAt       int64   `json:"updated_at,omitempty" gorm:"column:updated_at"`
}

func (Speed) TableName() string {
	return "speed"
}

// CpuBenchmark holds measures to the performance of the CPU for a node
// used as both gorm model and server json response
type CpuBenchmark struct {
	NodeTwinId     uint32  `json:"node_twin_id,omitempty" gorm:"unique;not null"`
	SingleThreaded float64 `json:"single" gorm:"column:single_threaded"`
	MultiThreaded  float64 `json:"multi" gorm:"column:multi_threaded"`
	Threads        int     `json:"threads" gorm:"column:threads"`
	Workloads      int     `json:"workloads" gorm:"column:workloads"`
	UpdatedAt      int64   `json:"updated_at,omitempty" gorm:"column:updated_at"`
}

func (CpuBenchmark) TableName() string {
	return "cpu_benchmark"
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
	UpdatedAt  int64       `json:"updated_at,omitempty"`
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

type NodeFeatures struct {
	NodeTwinId uint32   `json:"node_twin_id,omitempty" gorm:"unique;not null"`
	UpdatedAt  int64    `json:"updated_at,omitempty"`
	Features   []string `json:"features" gorm:"type:jsonb;serializer:json"`
}

func (NodeFeatures) TableName() string {
	return "node_features"
}

type NodeLocation struct {
	Country   string `json:"country" gorm:"unique;not null"`
	Continent string `json:"continent"`
	UpdatedAt int64  `json:"updated_at"`
}

func (NodeLocation) TableName() string {
	return "node_location"
}

// SystemOverheadUsage represents the system resources consumed/used by ZOS itself
type SystemOverheadUsage struct {
	NodeTwinID uint32 `gorm:"column:node_twin_id;primaryKey"`
	SystemCRU  uint64 `gorm:"column:system_cru"`
	SystemHRU  uint64 `gorm:"column:system_hru"`
	SystemMRU  uint64 `gorm:"column:system_mru"`
	SystemSRU  uint64 `gorm:"column:system_sru"`
	UpdatedAt  int64  `gorm:"column:updated_at"`
}

func (SystemOverheadUsage) TableName() string {
	return "node_system_overhead_resources"
}
