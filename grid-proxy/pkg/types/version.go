package types

// Version represent the deployed version of gridproxy
type Version struct {
	Version string `json:"version"`
}

type IndexerState struct {
	UpdatedAt int64 `json:"updated_at"`
}

type IndexersState struct {
	Gpu       IndexerState `json:"gpu"`
	Health    IndexerState `json:"health"`
	Dmi       IndexerState `json:"dmi"`
	Speed     IndexerState `json:"speed"`
	Ipv6      IndexerState `json:"ipv6"`
	Workloads IndexerState `json:"workloads"`
}

// Health represent the healthiness of the server and connections
type Health struct {
	TotalStateOk bool          `json:"total_state_ok"`
	DBConn       string        `json:"db_conn"`
	RMBConn      string        `json:"rmb_conn"`
	Indexers     IndexersState `json:"indexers"`
}
