package db

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func (d *PostgresDatabase) nodeTableQuery() *gorm.DB {
	subquery := d.gormDB.Table("node_contract").
		Select("DISTINCT ON (node_id) node_id, contract_id").
		Where("state IN ('Created', 'GracePeriod')")

	return d.gormDB.
		Table("node").
		Select(
			"node.id",
			"node.node_id",
			"node.farm_id",
			"node.twin_id",
			"node.country",
			"node.grid_version",
			"node.city",
			"node.uptime",
			"node.created",
			"node.farming_policy_id",
			"updated_at",
			"nodes_resources_view.total_cru",
			"nodes_resources_view.total_sru",
			"nodes_resources_view.total_hru",
			"nodes_resources_view.total_mru",
			"nodes_resources_view.used_cru",
			"nodes_resources_view.used_sru",
			"nodes_resources_view.used_hru",
			"nodes_resources_view.used_mru",
			"public_config.domain",
			"public_config.gw4",
			"public_config.gw6",
			"public_config.ipv4",
			"public_config.ipv6",
			"node.certification",
			"farm.dedicated_farm as dedicated",
			"rent_contract.contract_id as rent_contract_id",
			"rent_contract.twin_id as rented_by_twin_id",
			"node.serial_number",
			"convert_to_decimal(location.longitude) as longitude",
			"convert_to_decimal(location.latitude) as latitude",
			"node.power",
			"node.has_gpu",
			"node.extra_fee",
		).
		Joins(
			"LEFT JOIN nodes_resources_view ON node.node_id = nodes_resources_view.node_id",
		).
		Joins(
			"LEFT JOIN public_config ON node.id = public_config.node_id",
		).
		Joins(
			"LEFT JOIN rent_contract ON rent_contract.state IN ('Created', 'GracePeriod') AND rent_contract.node_id = node.node_id",
		).
		Joins(
			"LEFT JOIN (?) AS node_contract ON node_contract.node_id = node.node_id", subquery,
		).
		Joins(
			"LEFT JOIN farm ON node.farm_id = farm.farm_id",
		).
		Joins(
			"LEFT JOIN location ON node.location_id = location.id",
		)
}

// GetNode returns node info
func (d *PostgresDatabase) GetNode(nodeID uint32) (Node, error) {
	q := d.nodeTableQuery()
	q = q.Where("node.node_id = ?", nodeID)
	q = q.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
	var node Node
	res := q.Scan(&node)
	if d.shouldRetry(res.Error) {
		res = q.Scan(&node)
	}
	if res.Error != nil {
		return Node{}, res.Error
	}
	if node.ID == "" {
		return Node{}, ErrNodeNotFound
	}
	return node, nil
}

// GetNodes returns nodes filtered and paginated
func (d *PostgresDatabase) GetNodes(filter types.NodeFilter, limit types.Limit) ([]Node, uint, error) {
	q := d.nodeTableQuery()
	q = q.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})

	condition := "TRUE"
	if filter.Status != nil {
		condition = decideNodeStatusCondition(*filter.Status)
	}

	q = q.Where(condition)

	if filter.FreeMRU != nil {
		q = q.Where("nodes_resources_view.free_mru >= ?", *filter.FreeMRU)
	}
	if filter.FreeHRU != nil {
		q = q.Where("nodes_resources_view.free_hru >= ?", *filter.FreeHRU)
	}
	if filter.FreeSRU != nil {
		q = q.Where("nodes_resources_view.free_sru >= ?", *filter.FreeSRU)
	}
	if filter.TotalCRU != nil {
		q = q.Where("nodes_resources_view.total_cru >= ?", *filter.TotalCRU)
	}
	if filter.TotalHRU != nil {
		q = q.Where("nodes_resources_view.total_hru >= ?", *filter.TotalHRU)
	}
	if filter.TotalMRU != nil {
		q = q.Where("nodes_resources_view.total_mru >= ?", *filter.TotalMRU)
	}
	if filter.TotalSRU != nil {
		q = q.Where("nodes_resources_view.total_sru >= ?", *filter.TotalSRU)
	}
	if filter.Country != nil {
		q = q.Where("LOWER(node.country) = LOWER(?)", *filter.Country)
	}
	if filter.CountryContains != nil {
		q = q.Where("node.country ILIKE '%' || ? || '%'", *filter.CountryContains)
	}
	if filter.City != nil {
		q = q.Where("LOWER(node.city) = LOWER(?)", *filter.City)
	}
	if filter.CityContains != nil {
		q = q.Where("node.city ILIKE '%' || ? || '%'", *filter.CityContains)
	}
	if filter.NodeID != nil {
		q = q.Where("node.node_id = ?", *filter.NodeID)
	}
	if filter.TwinID != nil {
		q = q.Where("node.twin_id = ?", *filter.TwinID)
	}
	if filter.FarmIDs != nil {
		q = q.Where("node.farm_id IN ?", filter.FarmIDs)
	}
	if filter.FarmName != nil {
		q = q.Where("LOWER(farm.name) = LOWER(?)", *filter.FarmName)
	}
	if filter.FarmNameContains != nil {
		q = q.Where("farm.name ILIKE '%' || ? || '%'", *filter.FarmNameContains)
	}
	if filter.FreeIPs != nil {
		q = q.Where("(SELECT count(id) from public_ip WHERE public_ip.farm_id = farm.id AND public_ip.contract_id = 0) >= ?", *filter.FreeIPs)
	}
	if filter.IPv4 != nil {
		q = q.Where("COALESCE(public_config.ipv4, '') != ''")
	}
	if filter.IPv6 != nil {
		q = q.Where("COALESCE(public_config.ipv6, '') != ''")
	}
	if filter.Domain != nil {
		q = q.Where("COALESCE(public_config.domain, '') != ''")
	}
	if filter.Dedicated != nil {
		q = q.Where("farm.dedicated_farm = ?", *filter.Dedicated)
	}
	if filter.Rentable != nil {
		q = q.Where(`? = ((farm.dedicated_farm = true OR COALESCE(node_contract.contract_id, 0) = 0) AND COALESCE(rent_contract.contract_id, 0) = 0)`, *filter.Rentable)
	}
	if filter.RentedBy != nil {
		q = q.Where(`COALESCE(rent_contract.twin_id, 0) = ?`, *filter.RentedBy)
	}
	if filter.AvailableFor != nil {
		q = q.Where(`COALESCE(rent_contract.twin_id, 0) = ? OR (COALESCE(rent_contract.twin_id, 0) = 0 AND farm.dedicated_farm = false)`, *filter.AvailableFor)
	}
	if filter.Rented != nil {
		q = q.Where(`? = (COALESCE(rent_contract.contract_id, 0) != 0)`, *filter.Rented)
	}
	if filter.CertificationType != nil {
		q = q.Where("node.certification ILIKE ?", *filter.CertificationType)
	}
	if filter.HasGPU != nil {
		q = q.Where("node.has_gpu = ?", *filter.HasGPU)
	}

	var count int64
	if limit.Randomize || limit.RetCount {
		q = q.Session(&gorm.Session{})
		res := q.Count(&count)
		if d.shouldRetry(res.Error) {
			res = q.Count(&count)
		}
		if res.Error != nil {
			return nil, 0, res.Error
		}
	}
	if limit.Randomize {
		q = q.Limit(int(limit.Size)).
			Offset(int(rand.Intn(int(count)) - int(limit.Size)))
	} else {
		if filter.AvailableFor != nil {
			q = q.Order("(case when rent_contract is not null then 1 else 2 end)")
		}
		q = q.Limit(int(limit.Size)).
			Offset(int(limit.Page-1) * int(limit.Size)).
			Order("node_id")
	}

	var nodes []Node
	q = q.Session(&gorm.Session{})
	res := q.Scan(&nodes)
	if d.shouldRetry(res.Error) {
		res = q.Scan(&nodes)
	}
	if res.Error != nil {
		return nil, 0, res.Error
	}
	return nodes, uint(count), nil
}

func decideNodeStatusCondition(status string) string {
	condition := "TRUE"
	nodeUpInterval := time.Now().Unix() - nodeStateFactor*int64(reportInterval.Seconds())

	if status == "up" {
		condition = fmt.Sprintf(`node.updated_at >= %d`, nodeUpInterval)
	} else if status == "down" {
		condition = fmt.Sprintf(`node.updated_at < %d 
				OR node.updated_at IS NULL
				OR node.power->> 'target' = 'Up' AND node.power->> 'state' = 'Down'`, nodeUpInterval)
	} else if status == "standby" {
		condition = `node.power->> 'target' = 'Down'`
	}

	return condition
}
