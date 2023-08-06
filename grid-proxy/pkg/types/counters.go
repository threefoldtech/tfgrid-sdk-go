package types

// Counters contains aggregate info about the grid
type Counters struct {
	Nodes             int64            `json:"nodes"`
	Farms             int64            `json:"farms"`
	Countries         int64            `json:"countries"`
	TotalCRU          int64            `json:"total_cru"`
	TotalSRU          int64            `json:"total_sru"`
	TotalMRU          int64            `json:"total_mru"`
	TotalHRU          int64            `json:"total_hru"`
	PublicIPs         int64            `json:"public_ips"`
	AccessNodes       int64            `json:"access_nodes"`
	Gateways          int64            `json:"gateways"`
	Twins             int64            `json:"twins"`
	Contracts         int64            `json:"contracts"`
	NodesDistribution map[string]int64 `json:"nodes_distribution" gorm:"-:all"`
	GPUs              int64            `json:"gpus"`
}

// StatsFilter statistics filters
type StatsFilter struct {
	Status *string `schema:"status,omitempty"`
}

type NodeStatisticsResources struct {
	CRU   int `json:"cru"`
	HRU   int `json:"hru"`
	IPV4U int `json:"ipv4u"`
	MRU   int `json:"mru"`
	SRU   int `json:"sru"`
}

type NodeStatisticsUsers struct {
	Deployments int `json:"deployments"`
	Workloads   int `json:"workloads"`
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
