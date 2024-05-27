package types

// Version represent the deployed version of gridproxy
type Version struct {
	Version string `json:"version"`
}

// Healthiness represent the healthiness of the server and connections
type Healthiness struct {
	TotalStateOk bool   `json:"total_state_ok"`
	DBConn       string `json:"db_conn"`
	RMBConn      string `json:"rmb_conn"`
}
