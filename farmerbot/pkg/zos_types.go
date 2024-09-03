package pkg

type PoolMetrics struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Size uint64 `json:"size"`
	Used uint64 `json:"used"`
}

type Capacity struct {
	CRU   uint64 `json:"cru"`
	SRU   uint64 `json:"sru"`
	HRU   uint64 `json:"hru"`
	MRU   uint64 `json:"mru"`
	IPV4U uint64 `json:"ipv4u"`
}

type Counters struct {
	// Total system capacity
	Total Capacity `json:"total"`
	// Used capacity this include user + system resources
	Used Capacity `json:"used"`
	// System resource reserved by zos
	System Capacity `json:"system"`
}

type GPU struct {
	ID       string `json:"id"`
	Vendor   string `json:"vendor"`
	Device   string `json:"device"`
	Contract uint64 `json:"contract"`
}
