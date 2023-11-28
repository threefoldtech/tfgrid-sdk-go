package types

type SortOrder string
type SortBy string

const (
	// order
	SortOrderAsc  SortOrder = "ASC"
	SortOrderDesc SortOrder = "DESC"

	// node
	SortByPower    SortBy = "power"
	SortByNodeID   SortBy = "node_id"
	SortByCountry  SortBy = "country"
	SortByCity     SortBy = "city"
	SortByUptime   SortBy = "uptime"
	SortByTotalCRU SortBy = "totalcru"
	SortByTotalHRU SortBy = ""
	SortByTotalMRU SortBy = ""
	SortByTotalSRU SortBy = ""
	SortByFreeCRU  SortBy = ""
	SortByFreeHRU  SortBy = ""
	SortByFreeMRU  SortBy = ""
	SortByFreeSRU  SortBy = ""
	SortByDomain   SortBy = ""

	// twin
	SortByAccountID SortBy = ""
	SortByRelay     SortBy = ""
	SortByPublicIP  SortBy = ""

	// farm
	SortByName            SortBy = ""
	SortByPricingPolicyID SortBy = ""
	SortByCertification   SortBy = ""
	SortByStellarAddress  SortBy = ""
	SortByPublicIPsCount  SortBy = ""

	// contract
	SortByType              SortBy = ""
	SortByState             SortBy = ""
	SortByCreatedAt         SortBy = ""
	SortByNumberOfPublicIps SortBy = ""

	// common
	SortByDedicated SortBy = ""
	SortByID        SortBy = ""
	SortByFarmID    SortBy = ""
	SortByTwinID    SortBy = ""
)

// Limit used for pagination
type Limit struct {
	Size      uint64
	Page      uint64
	RetCount  bool
	Randomize bool
	SortBy    string
	SortOrder SortOrder
}

// DefaultLimit returns the default values for the pagination
func DefaultLimit() Limit {
	return Limit{
		Size:      50,
		Page:      1,
		RetCount:  true,
		Randomize: false,
	}
}
