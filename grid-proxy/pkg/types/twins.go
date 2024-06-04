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

// TwinConsumption show a report of user spent in TFT
type TwinConsumption struct {
	LastHourConsumption float64 `json:"last_hour_consumption"`
	OverallConsumption  float64 `json:"overall_consumption"`
}
