package db

import (
	"fmt"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"gorm.io/gorm"
)

func sort(q *gorm.DB, defaultField string, limit types.Limit) *gorm.DB {
	if limit.Randomize {
		q = q.Order("random()")
	} else if limit.SortBy != "" {
		order := types.SortOrderAsc
		if strings.EqualFold(string(limit.SortOrder), string(types.SortOrderDesc)) {
			order = types.SortOrderDesc
		}
		q = q.Order(fmt.Sprintf("%s %s", limit.SortBy, order))
	} else {
		q = q.Order(defaultField)
	}

	return q
}
