package db

import (
	"math/rand"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// GetTwins returns twins filtered and paginated
func (d *PostgresDatabase) GetTwins(filter types.TwinFilter, limit types.Limit) ([]types.Twin, uint, error) {
	q := d.gormDB.
		Table("twin").
		Select(
			"twin_id",
			"account_id",
			"relay",
			"public_key",
		)
	if filter.TwinID != nil {
		q = q.Where("twin_id = ?", *filter.TwinID)
	}
	if filter.AccountID != nil {
		q = q.Where("account_id = ?", *filter.AccountID)
	}
	if filter.Relay != nil {
		q = q.Where("relay = ?", *filter.Relay)
	}
	if filter.PublicKey != nil {
		q = q.Where("public_key = ?", *filter.PublicKey)
	}
	var count int64
	if limit.Randomize || limit.RetCount {
		if res := q.Count(&count); res.Error != nil {
			return nil, 0, errors.Wrap(res.Error, "couldn't get twin count")
		}
	}
	if limit.Randomize {
		q = q.Limit(int(limit.Size)).
			Offset(int(rand.Intn(int(count)) - int(limit.Size)))
	} else {
		q = q.Limit(int(limit.Size)).
			Offset(int(limit.Page-1) * int(limit.Size)).
			Order("twin.twin_id")
	}
	twins := []types.Twin{}

	if res := q.Scan(&twins); res.Error != nil {
		return twins, uint(count), errors.Wrap(res.Error, "failed to scan returned twins from database")
	}
	return twins, uint(count), nil
}
