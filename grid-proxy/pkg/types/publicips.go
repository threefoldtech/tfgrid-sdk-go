package types

// PublicIP info about public ip in the farm
type PublicIP struct {
	ID         string `json:"id"`
	IP         string `json:"ip" sort:"ip"`
	Gateway    string `json:"gateway"`
	ContractID uint64 `json:"contract_id" sort:"contract_id"`
	FarmID     uint64 `json:"farm_id,omitempty" sort:"farm_id"`
}

type PublicIpFilter struct {
	FarmIDs []uint64 `schema:"farm_ids,omitempty"`
	Free    *bool    `schema:"free,omitempty"`
	Ip      *string  `schema:"ip,omitempty"`
	Gateway *string  `schema:"gateway,omitempty"`
}

// PublicIPSelect represents fields that can be selected in public IPs API response
type PublicIPSelect struct {
	ID         bool `schema:"id"`
	IP         bool `schema:"ip"`
	Gateway    bool `schema:"gateway"`
	ContractID bool `schema:"contract_id"`
	FarmID     bool `schema:"farm_id"`
}

// HasSelection returns true if any field is selected
func (ps PublicIPSelect) HasSelection() bool {
	return ps.ID || ps.IP || ps.Gateway || ps.ContractID || ps.FarmID
}
