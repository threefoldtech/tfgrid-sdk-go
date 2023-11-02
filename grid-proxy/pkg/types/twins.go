package types

// Twin is the twin info
type Twin struct {
	TwinID    uint   `json:"twin_id"`
	AccountID string `json:"account_id"`
	Relay     string `json:"relay"`
	PublicKey string `json:"public_key"`
}

// TwinFilter twin filters
type TwinFilter struct {
	TwinID    *uint64 `schema:"twin_id,omitempty"`
	AccountID *string `schema:"account_id,omitempty"`
	Relay     *string `schema:"relay,omitempty"`
	PublicKey *string `schema:"public_key,omitempty"`
}
