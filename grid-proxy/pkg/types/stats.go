package types

// Stats contains aggregate info about the grid
type Stats struct {
	Nodes             int64            `json:"nodes"`
	Farms             int64            `json:"farms"`
	Countries         int64            `json:"countries"`
	TotalCRU          int64            `json:"totalCru"`
	TotalSRU          int64            `json:"totalSru"`
	TotalMRU          int64            `json:"totalMru"`
	TotalHRU          int64            `json:"totalHru"`
	PublicIPs         int64            `json:"publicIps"`
	AccessNodes       int64            `json:"accessNodes"`
	Gateways          int64            `json:"gateways"`
	Twins             int64            `json:"twins"`
	Contracts         int64            `json:"contracts"`
	NodesDistribution map[string]int64 `json:"nodesDistribution" gorm:"-:all"`
	GPUs              int64            `json:"gpus"`
	DedicatedNodes    int64            `json:"dedicatedNodes"`
}

// StatsFilter statistics filters
type StatsFilter struct {
	Status *string `schema:"status,omitempty"`
}

// NodeStatisticsResources resources returned on node statistics
type NodeStatisticsResources struct {
	CRU   int `json:"cru"`
	HRU   int `json:"hru"`
	IPV4U int `json:"ipv4u"`
	MRU   int `json:"mru"`
	SRU   int `json:"sru"`
}

// NodeStatisticsUsers users info returned on node statistics
type NodeStatisticsUsers struct {
	Deployments             int    `json:"deployments"`
	Workloads               int    `json:"workloads"`
	LastDeploymentTimestamp uint64 `json:"last_deployment_timestamp"`
}

// NodeStatistics node statistics info
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
