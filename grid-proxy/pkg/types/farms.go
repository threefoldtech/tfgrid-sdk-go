package types

// Farm info about the farm
type Farm struct {
	Name              string     `json:"name" sort:"name"`
	FarmID            int        `json:"farmId" sort:"farm_id"`
	TwinID            int        `json:"twinId" sort:"twin_id"`
	PricingPolicyID   int        `json:"pricingPolicyId"`
	CertificationType string     `json:"certificationType"`
	StellarAddress    string     `json:"stellarAddress"`
	Dedicated         bool       `json:"dedicated" sort:"dedicated"`
	PublicIps         []PublicIP `json:"publicIps" sort:"public_ips"`
	// TODO: maybe separate the sorting params in a different struct
	_ string `sort:"free_ips"`
	_ string `sort:"total_ips"`
	_ string `sort:"used_ips"`
}

// FarmFilter farm filters
type FarmFilter struct {
	FreeIPs           *uint64  `schema:"free_ips,omitempty"`
	TotalIPs          *uint64  `schema:"total_ips,omitempty"`
	StellarAddress    *string  `schema:"stellar_address,omitempty"`
	PricingPolicyID   *uint64  `schema:"pricing_policy_id,omitempty"`
	FarmID            *uint64  `schema:"farm_id,omitempty"`
	TwinID            *uint64  `schema:"twin_id,omitempty"`
	Name              *string  `schema:"name,omitempty"`
	NameContains      *string  `schema:"name_contains,omitempty"`
	CertificationType *string  `schema:"certification_type,omitempty"`
	Dedicated         *bool    `schema:"dedicated,omitempty"`
	NodeFreeMRU       *uint64  `schema:"node_free_mru,omitempty"`
	NodeFreeHRU       *uint64  `schema:"node_free_hru,omitempty"`
	NodeFreeSRU       *uint64  `schema:"node_free_sru,omitempty"`
	NodeTotalCRU      *uint64  `schema:"node_total_cru,omitempty"`
	NodeStatus        []string `schema:"node_status,omitempty"`
	NodeRentedBy      *uint64  `schema:"node_rented_by,omitempty"`
	NodeAvailableFor  *uint64  `schema:"node_available_for,omitempty"`
	NodeHasGPU        *bool    `schema:"node_has_gpu,omitempty"`
	NodeHasIpv6       *bool    `schema:"node_has_ipv6,omitempty"`
	NodeCertified     *bool    `schema:"node_certified,omitempty"`
	NodeFeatures      []string `schema:"node_features,omitempty"`
	Country           *string  `schema:"country,omitempty"`
	Region            *string  `schema:"region,omitempty"`
}

func (f FarmFilter) Validate() error {
	return validateNodeFeatures(f.NodeFeatures)
}

func (f FarmFilter) IsNodeFilterRequested() bool {
	return f.NodeAvailableFor != nil || f.NodeFreeHRU != nil ||
		f.NodeCertified != nil || f.NodeFreeMRU != nil ||
		f.NodeFreeSRU != nil || f.NodeHasGPU != nil ||
		f.NodeRentedBy != nil || len(f.NodeStatus) != 0 ||
		f.NodeTotalCRU != nil || f.Country != nil ||
		f.Region != nil || f.NodeHasIpv6 != nil ||
		len(f.NodeFeatures) != 0
}

// FarmSelect represents fields that can be selected in farms API response
type FarmSelect struct {
	Name              bool `schema:"name"`
	FarmID            bool `schema:"farm_id"`
	TwinID            bool `schema:"twin_id"`
	PricingPolicyID   bool `schema:"pricing_policy_id"`
	CertificationType bool `schema:"certification_type"`
	StellarAddress    bool `schema:"stellar_address"`
	Dedicated         bool `schema:"dedicated"`
	PublicIps         bool `schema:"public_ips"`
}

// HasSelection returns true if any field is selected
func (fs FarmSelect) HasSelection() bool {
	return fs.Name || fs.FarmID || fs.TwinID || fs.PricingPolicyID ||
		fs.CertificationType || fs.StellarAddress || fs.Dedicated || fs.PublicIps
}
