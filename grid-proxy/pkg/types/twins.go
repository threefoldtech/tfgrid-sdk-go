package types

// Twin is the twin info
type Twin struct {
	TwinID    uint   `json:"twinId" sort:"twin_id"`
	AccountID string `json:"accountId" sort:"account_id"`
	Relay     string `json:"relay" sort:"relay"`
	PublicKey string `json:"publicKey" sort:"public_key"`
}

// TwinFilter twin filters
type TwinFilter struct {
	TwinID    *uint64 `schema:"twin_id,omitempty"`
	AccountID *string `schema:"account_id,omitempty"`
	Relay     *string `schema:"relay,omitempty"`
	PublicKey *string `schema:"public_key,omitempty"`
}

type TwinFee struct {
	LastHourSpent float64
	TotalSpend    float64
}
