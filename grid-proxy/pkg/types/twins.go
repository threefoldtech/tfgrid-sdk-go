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

// TwinSelect represents fields that can be selected in twins API response
type TwinSelect struct {
	TwinID    bool `schema:"twin_id"`
	AccountID bool `schema:"account_id"`
	Relay     bool `schema:"relay"`
	PublicKey bool `schema:"public_key"`
}

// HasSelection returns true if any field is selected
func (ts TwinSelect) HasSelection() bool {
	return ts.TwinID || ts.AccountID || ts.Relay || ts.PublicKey
}

// TwinConsumption show a report of user spent in TFT
type TwinConsumption struct {
	LastHourConsumption float64 `json:"last_hour_consumption"`
	OverallConsumption  float64 `json:"overall_consumption"`
}
