package models

// Farm of the farmer
type Farm struct {
	ID          uint32 `json:"id" yaml:"id" toml:"id"`
	Description string `json:"description,omitempty" yaml:"description,omitempty" toml:"description,omitempty"`
	PublicIPs   uint64 `json:"public_ips,omitempty" yaml:"public_ips,omitempty" toml:"public_ips,omitempty"`
}
