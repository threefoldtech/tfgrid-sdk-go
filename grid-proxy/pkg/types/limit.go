package types

import (
	"fmt"
	"reflect"
	"strings"
)

// SortOrder is the direction of sorting
type SortOrder string

// SortBy is the sorted by filed
type SortBy string

const (
	SortOrderAsc  SortOrder = "ASC"
	SortOrderDesc SortOrder = "DESC"
)

func (so SortOrder) valid() error {
	if so == "" {
		return nil
	}
	if strings.EqualFold(string(so), string(SortOrderAsc)) || strings.EqualFold(string(so), string(SortOrderDesc)) {
		return nil
	}

	return fmt.Errorf("%q is not a valid sort order", so)
}

func (sb SortBy) valid(typ interface{}) error {
	if sb == "" {
		return nil
	}

	objType := reflect.TypeOf(typ)
	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)
		sortTag, _ := field.Tag.Lookup("sort")

		// for nested fields, append the inner json tag to the sort tag (ex: "total_" + "cru")
		if field.Type.Kind() == reflect.Struct {
			innerType := field.Type
			for j := 0; j < innerType.NumField(); j++ {
				innerField := innerType.Field(j)
				jsonTag, hasJson := innerField.Tag.Lookup("json")
				if hasJson && sortTag+jsonTag == string(sb) {
					return nil
				}
			}
		} else {
			if sortTag == string(sb) {
				return nil
			}
		}
	}

	return fmt.Errorf("%q is not a sort filed for %v type", sb, objType.Name())
}

// Limit used for pagination
type Limit struct {
	Size      uint64    `schema:"size,omitempty"`
	Page      uint64    `schema:"page,omitempty"`
	RetCount  bool      `schema:"ret_count,omitempty"`
	Randomize bool      `schema:"randomize,omitempty"`
	SortBy    SortBy    `schema:"sort_by,omitempty"`
	SortOrder SortOrder `schema:"sort_order,omitempty"`
	Balance   float64   `schema:"balance,omitempty"`
}

// Valid validates the sorting values
func (l *Limit) Valid(typ interface{}) error {
	if err := l.SortBy.valid(typ); err != nil {
		return err
	}
	if err := l.SortOrder.valid(); err != nil {
		return err
	}
	return nil
}

// DefaultLimit returns the default values for the pagination
func DefaultLimit() Limit {
	return Limit{
		Size:      50,
		Page:      1,
		RetCount:  false,
		Randomize: false,
		SortBy:    "",
		SortOrder: "",
		Balance:   0,
	}
}
