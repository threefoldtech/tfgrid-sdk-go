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
