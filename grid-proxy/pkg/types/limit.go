package types

type SortOrder string

const (
	SortOrderAsc  SortOrder = "ASC"
	SortOrderDesc SortOrder = "DESC"
)

// Limit used for pagination
type Limit struct {
	Size      uint64    `schema:"size,omitempty"`
	Page      uint64    `schema:"page,omitempty"`
	RetCount  bool      `schema:"ret_count,omitempty"`
	Randomize bool      `schema:"randomize,omitempty"`
	SortBy    string    `schema:"sort_by,omitempty"`
	SortOrder SortOrder `schema:"sort_order,omitempty"`
}

// DefaultLimit returns the default values for the pagination
func DefaultLimit() Limit {
	return Limit{
		Size:      50,
		Page:      1,
		RetCount:  true,
		Randomize: false,
		SortBy:    "",
		SortOrder: "",
	}
}
