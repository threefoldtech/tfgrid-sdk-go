package types

// Limit used for pagination
type Limit struct {
	Size      uint64 `schema:"size"`
	Page      uint64 `schema:"page"`
	RetCount  bool   `schema:"ret_count"`
	Randomize bool   `schema:"randomize"`
}

func DefaultLimit() Limit {
	return Limit{
		Size:      50,
		Page:      1,
		RetCount:  true,
		Randomize: false,
	}
}
