package models

// Farm of the farmer
type Farm struct {
	ID          uint32 `json:"id"`
	Description string `json:"description,omitempty"`
	PublicIPs   uint64 `json:"public_ips,omitempty"`
}
