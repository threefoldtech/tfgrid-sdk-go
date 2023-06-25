package db

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"gorm.io/gorm"
)

func (d *PostgresDatabase) farmTableQuery() *gorm.DB {
	return d.gormDB.
		Table("farm").
		Select(
			"DISTINCT ON (farm.farm_id) farm.farm_id",
			"farm.name",
			"farm.twin_id",
			"farm.pricing_policy_id",
			"farm.certification",
			"farm.stellar_address",
			"farm.dedicated_farm as dedicated",
			"COALESCE(public_ip.public_ips, '[]') as public_ips",
		).
		Joins(
			`LEFT JOIN
		(SELECT
			farm_id, 
			json_agg(json_build_object('id', id, 'ip', ip, 'contract_id', contract_id, 'gateway', gateway)) as public_ips
		FROM
			public_ip
		GROUP by farm_id) public_ip
		ON public_ip.farm_id = farm.id`,
		)
}

// GetFarm return farm info
func (d *PostgresDatabase) GetFarm(farmID uint32) (Farm, error) {
	q := d.farmTableQuery()
	q = q.Where("farm.farm_id = ?", farmID)
	var farm Farm
	if res := q.Scan(&farm); res.Error != nil {
		return farm, errors.Wrap(res.Error, "failed to scan returned farm from database")
	}
	return farm, nil
}

// GetFarms return farms filtered and paginated
func (d *PostgresDatabase) GetFarms(filter types.FarmFilter, limit types.Limit) ([]Farm, uint, error) {
	q := d.farmTableQuery()

	if filter.NodeFreeHRU != nil || filter.NodeFreeMRU != nil || filter.NodeFreeSRU != nil {
		q = q.Joins(
			"LEFT JOIN node on farm.farm_id = node.farm_id",
		).Joins(
			"LEFT JOIN nodes_resources_view on node.node_id = nodes_resources_view.node_id",
		)

		if filter.NodeFreeMRU != nil {
			q = q.Where("EXISTS( select node.node_id WHERE nodes_resources_view.free_mru >= ?)", *filter.NodeFreeMRU)
		}
		if filter.NodeFreeHRU != nil {
			q = q.Where("EXISTS( select node.node_id WHERE nodes_resources_view.free_hru >= ?)", *filter.NodeFreeHRU)
		}
		if filter.NodeFreeSRU != nil {
			q = q.Where("EXISTS( select node.node_id WHERE nodes_resources_view.free_sru >= ?)", *filter.NodeFreeSRU)
		}
	}

	if filter.FreeIPs != nil {
		q = q.Where("(SELECT count(id) from public_ip WHERE public_ip.farm_id = farm.id and public_ip.contract_id = 0) >= ?", *filter.FreeIPs)
	}
	if filter.TotalIPs != nil {
		q = q.Where("(SELECT count(id) from public_ip WHERE public_ip.farm_id = farm.id) >= ?", *filter.TotalIPs)
	}
	if filter.StellarAddress != nil {
		q = q.Where("farm.stellar_address = ?", *filter.StellarAddress)
	}
	if filter.PricingPolicyID != nil {
		q = q.Where("farm.pricing_policy_id = ?", *filter.PricingPolicyID)
	}
	if filter.FarmID != nil {
		q = q.Where("farm.farm_id = ?", *filter.FarmID)
	}
	if filter.TwinID != nil {
		q = q.Where("farm.twin_id = ?", *filter.TwinID)
	}
	if filter.Name != nil {
		q = q.Where("LOWER(farm.name) = LOWER(?)", *filter.Name)
	}

	if filter.NameContains != nil {
		escaped := strings.Replace(*filter.NameContains, "%", "\\%", -1)
		escaped = strings.Replace(escaped, "_", "\\_", -1)
		q = q.Where("farm.name ILIKE ?", fmt.Sprintf("%%%s%%", escaped))
	}

	if filter.CertificationType != nil {
		q = q.Where("farm.certification = ?", *filter.CertificationType)
	}

	if filter.Dedicated != nil {
		q = q.Where("farm.dedicated_farm = ?", *filter.Dedicated)
	}
	var count int64
	if limit.Randomize || limit.RetCount {
		if res := q.Count(&count); res.Error != nil {
			return nil, 0, errors.Wrap(res.Error, "couldn't get farm count")
		}
	}
	if limit.Randomize {
		q = q.Limit(int(limit.Size)).
			Offset(int(rand.Intn(int(count)) - int(limit.Size)))
	} else {
		q = q.Limit(int(limit.Size)).
			Offset(int(limit.Page-1) * int(limit.Size)).
			Order("farm.farm_id")
	}
	var farms []Farm
	if res := q.Scan(&farms); res.Error != nil {
		return farms, uint(count), errors.Wrap(res.Error, "failed to scan returned farm from database")
	}
	return farms, uint(count), nil
}
