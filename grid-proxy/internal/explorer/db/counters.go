package db

import (
	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// GetCounters returns aggregate info about the grid
func (d *PostgresDatabase) GetCounters(filter types.StatsFilter) (types.Counters, error) {
	var counters types.Counters

	var twins int64
	if res := d.gormDB.Table("twin").Count(&twins); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get twin count")
	}
	counters.Twins = uint64(twins)

	var publicIPs int64
	if res := d.gormDB.Table("public_ip").Count(&publicIPs); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get public ip count")
	}
	counters.PublicIPs = uint64(publicIPs)

	var contractsCount int64
	if res := d.gormDB.Table("node_contract").Count(&contractsCount); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get node contract count")
	}
	counters.Contracts += uint64(contractsCount)

	if res := d.gormDB.Table("rent_contract").Count(&contractsCount); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get rent contract count")
	}
	counters.Contracts += uint64(contractsCount)

	if res := d.gormDB.Table("name_contract").Count(&contractsCount); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get name contract count")
	}
	counters.Contracts += uint64(contractsCount)

	var farms int64
	if res := d.gormDB.Table("farm").Distinct("farm_id").Count(&farms); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get farm count")
	}
	counters.Farms = uint64(farms)

	condition := "TRUE"
	if filter.Status != nil {
		condition = decideNodeStatusCondition(*filter.Status)
	}

	if res := d.gormDB.
		Table("node").
		Select(
			"sum(node_resources_total.cru) as total_cru",
			"sum(node_resources_total.sru) as total_sru",
			"sum(node_resources_total.hru) as total_hru",
			"sum(node_resources_total.mru) as total_mru",
		).
		Joins("LEFT JOIN node_resources_total ON node.id = node_resources_total.node_id").
		Where(condition).
		Scan(&counters); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get nodes total resources")
	}

	var nodes int64
	if res := d.gormDB.Table("node").
		Where(condition).Count(&nodes); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get node count")
	}
	counters.Nodes = uint64(nodes)

	var countries int64
	if res := d.gormDB.Table("node").
		Where(condition).Distinct("country").Count(&countries); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get country count")
	}
	counters.Countries = uint64(countries)

	query := d.gormDB.
		Table("node").
		Joins(
			`RIGHT JOIN public_config
			ON node.id = public_config.node_id
			`,
		)

	var accessNodes int64
	if res := query.Where(condition).Where("COALESCE(public_config.ipv4, '') != '' OR COALESCE(public_config.ipv6, '') != ''").Count(&accessNodes); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get access node count")
	}
	counters.AccessNodes = uint64(accessNodes)

	var gateways int64
	if res := query.Where(condition).Where("COALESCE(public_config.domain, '') != '' AND (COALESCE(public_config.ipv4, '') != '' OR COALESCE(public_config.ipv6, '') != '')").Count(&gateways); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get gateway count")
	}
	counters.Gateways = uint64(gateways)

	var distribution []NodesDistribution
	if res := d.gormDB.Table("node").
		Select("country, count(node_id) as nodes").Where(condition).Group("country").Scan(&distribution); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get nodes distribution")
	}

	var gpus int64
	if res := d.gormDB.Table("node").Where(condition).Where("node.has_gpu = true").Count(&gpus); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get node with GPU count")
	}
	counters.GPUs = uint64(gpus)

	nodesDistribution := map[string]uint64{}
	for _, d := range distribution {
		nodesDistribution[d.Country] = d.Nodes
	}
	counters.NodesDistribution = nodesDistribution

	return counters, nil
}
