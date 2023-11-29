package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	// to use for database/sql
	_ "github.com/lib/pq"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/nodestatus"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var (
	// ErrNodeNotFound node not found
	ErrNodeNotFound = errors.New("node not found")
	// ErrFarmNotFound farm not found
	ErrFarmNotFound = errors.New("farm not found")
	//ErrViewNotFound
	ErrNodeResourcesViewNotFound = errors.New("ERROR: relation \"nodes_resources_view\" does not exist (SQLSTATE 42P01)")
	// ErrContractNotFound contract not found
	ErrContractNotFound = errors.New("contract not found")
)

const (
	setupPostgresql = `
	CREATE OR REPLACE VIEW nodes_resources_view AS SELECT
		node.node_id,
		COALESCE(sum(contract_resources.cru), 0) as used_cru,
		COALESCE(sum(contract_resources.mru), 0) + GREATEST(CAST((node_resources_total.mru / 10) AS bigint), 2147483648) as used_mru,
		COALESCE(sum(contract_resources.hru), 0) as used_hru,
		COALESCE(sum(contract_resources.sru), 0) + 21474836480 as used_sru,
		node_resources_total.mru - COALESCE(sum(contract_resources.mru), 0) - GREATEST(CAST((node_resources_total.mru / 10) AS bigint), 2147483648) as free_mru,
		node_resources_total.hru - COALESCE(sum(contract_resources.hru), 0) as free_hru,
		node_resources_total.sru - COALESCE(sum(contract_resources.sru), 0) - 21474836480 as free_sru,
		COALESCE(node_resources_total.cru, 0) as total_cru,
		COALESCE(node_resources_total.mru, 0) as total_mru,
		COALESCE(node_resources_total.hru, 0) as total_hru,
		COALESCE(node_resources_total.sru, 0) as total_sru,
		COALESCE(COUNT(DISTINCT state), 0) as states
	FROM contract_resources
	JOIN node_contract as node_contract
	ON node_contract.resources_used_id = contract_resources.id AND node_contract.state IN ('Created', 'GracePeriod')
	RIGHT JOIN node as node
	ON node.node_id = node_contract.node_id
	JOIN node_resources_total AS node_resources_total
	ON node_resources_total.node_id = node.id
	GROUP BY node.node_id, node_resources_total.mru, node_resources_total.sru, node_resources_total.hru, node_resources_total.cru;

	DROP FUNCTION IF EXISTS node_resources(query_node_id INTEGER);
	CREATE OR REPLACE function node_resources(query_node_id INTEGER)
	returns table (node_id INTEGER, used_cru NUMERIC, used_mru NUMERIC, used_hru NUMERIC, used_sru NUMERIC, free_mru NUMERIC, free_hru NUMERIC, free_sru NUMERIC, total_cru NUMERIC, total_mru NUMERIC, total_hru NUMERIC, total_sru NUMERIC, states BIGINT)
	as
	$body$
	SELECT
		node.node_id,
		COALESCE(sum(contract_resources.cru), 0) as used_cru,
		COALESCE(sum(contract_resources.mru), 0) + GREATEST(CAST((node_resources_total.mru / 10) AS bigint), 2147483648) as used_mru,
		COALESCE(sum(contract_resources.hru), 0) as used_hru,
		COALESCE(sum(contract_resources.sru), 0) + 21474836480 as used_sru,
		node_resources_total.mru - COALESCE(sum(contract_resources.mru), 0) - GREATEST(CAST((node_resources_total.mru / 10) AS bigint), 2147483648) as free_mru,
		node_resources_total.hru - COALESCE(sum(contract_resources.hru), 0) as free_hru,
		node_resources_total.sru - COALESCE(sum(contract_resources.sru), 0) - 21474836480 as free_sru,
		COALESCE(node_resources_total.cru, 0) as total_cru,
		COALESCE(node_resources_total.mru, 0) as total_mru,
		COALESCE(node_resources_total.hru, 0) as total_hru,
		COALESCE(node_resources_total.sru, 0) as total_sru,
		COALESCE(COUNT(DISTINCT state), 0) as states
	FROM contract_resources
	JOIN node_contract as node_contract
	ON node_contract.resources_used_id = contract_resources.id AND node_contract.state IN ('Created', 'GracePeriod')
	RIGHT JOIN node as node
	ON node.node_id = node_contract.node_id
	JOIN node_resources_total AS node_resources_total
	ON node_resources_total.node_id = node.id
	WHERE node.node_id = query_node_id
	GROUP BY node.node_id, node_resources_total.mru, node_resources_total.sru, node_resources_total.hru, node_resources_total.cru;
	$body$
	language sql;

	DROP FUNCTION IF EXISTS convert_to_decimal(v_input text);
	CREATE OR REPLACE FUNCTION convert_to_decimal(v_input text)
	RETURNS DECIMAL AS $$
	DECLARE v_dec_value DECIMAL DEFAULT NULL;
	BEGIN
		BEGIN
			v_dec_value := v_input::DECIMAL;
		EXCEPTION WHEN OTHERS THEN
			RAISE NOTICE 'Invalid decimal value: "%".  Returning NULL.', v_input;
			RETURN NULL;
		END;
	RETURN v_dec_value;
	END;
	$$ LANGUAGE plpgsql;

	DROP TRIGGER IF EXISTS node_added ON node;
	`
)

// PostgresDatabase postgres db client
type PostgresDatabase struct {
	gormDB     *gorm.DB
	connString string
}

func (d *PostgresDatabase) GetConnectionString() string {
	return d.connString
}

// NewPostgresDatabase returns a new postgres db client
func NewPostgresDatabase(host string, port int, user, password, dbname string, maxConns int) (Database, error) {
	connString := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	gormDB, err := gorm.Open(postgres.Open(connString), &gorm.Config{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create orm wrapper around db")
	}
	sql, err := gormDB.DB()
	if err != nil {
		return nil, errors.Wrap(err, "failed to configure DB connection")
	}

	sql.SetMaxIdleConns(3)
	sql.SetMaxOpenConns(maxConns)

	err = gormDB.AutoMigrate(&NodeGPU{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to auto migrate DB")
	}

	res := PostgresDatabase{gormDB, connString}
	if err := res.initialize(); err != nil {
		return nil, errors.Wrap(err, "failed to setup tables")
	}
	return &res, nil
}

// Close the db connection
func (d *PostgresDatabase) Close() error {
	db, err := d.gormDB.DB()
	if err != nil {
		return err
	}
	return db.Close()
}

func (d *PostgresDatabase) initialize() error {
	res := d.gormDB.Exec(setupPostgresql)
	return res.Error
}

// GetStats returns aggregate info about the grid
func (d *PostgresDatabase) GetStats(ctx context.Context, filter types.StatsFilter) (types.Stats, error) {
	var stats types.Stats
	if res := d.gormDB.WithContext(ctx).Table("twin").Count(&stats.Twins); res.Error != nil {
		return stats, errors.Wrap(res.Error, "couldn't get twin count")
	}
	if res := d.gormDB.WithContext(ctx).Table("public_ip").Count(&stats.PublicIPs); res.Error != nil {
		return stats, errors.Wrap(res.Error, "couldn't get public ip count")
	}
	var count int64
	if res := d.gormDB.WithContext(ctx).Table("node_contract").Count(&count); res.Error != nil {
		return stats, errors.Wrap(res.Error, "couldn't get node contract count")
	}
	stats.Contracts += count
	if res := d.gormDB.WithContext(ctx).Table("rent_contract").Count(&count); res.Error != nil {
		return stats, errors.Wrap(res.Error, "couldn't get rent contract count")
	}
	stats.Contracts += count
	if res := d.gormDB.WithContext(ctx).Table("name_contract").Count(&count); res.Error != nil {
		return stats, errors.Wrap(res.Error, "couldn't get name contract count")
	}
	stats.Contracts += count
	if res := d.gormDB.WithContext(ctx).Table("farm").Distinct("farm_id").Count(&stats.Farms); res.Error != nil {
		return stats, errors.Wrap(res.Error, "couldn't get farm count")
	}

	condition := "TRUE"
	if filter.Status != nil {
		condition = nodestatus.DecideNodeStatusCondition(*filter.Status)
	}

	if res := d.gormDB.WithContext(ctx).
		Table("node").
		Select(
			"sum(node_resources_total.cru) as total_cru",
			"sum(node_resources_total.sru) as total_sru",
			"sum(node_resources_total.hru) as total_hru",
			"sum(node_resources_total.mru) as total_mru",
		).
		Joins("LEFT JOIN node_resources_total ON node.id = node_resources_total.node_id").
		Where(condition).
		Scan(&stats); res.Error != nil {
		return stats, errors.Wrap(res.Error, "couldn't get nodes total resources")
	}
	if res := d.gormDB.WithContext(ctx).Table("node").
		Where(condition).Count(&stats.Nodes); res.Error != nil {
		return stats, errors.Wrap(res.Error, "couldn't get node count")
	}
	if res := d.gormDB.WithContext(ctx).Table("node").
		Where(condition).Distinct("country").Count(&stats.Countries); res.Error != nil {
		return stats, errors.Wrap(res.Error, "couldn't get country count")
	}
	query := d.gormDB.WithContext(ctx).
		Table("node").
		Joins(
			`RIGHT JOIN public_config
			ON node.id = public_config.node_id
			`,
		)

	if res := query.Where(condition).Where("COALESCE(public_config.ipv4, '') != '' OR COALESCE(public_config.ipv6, '') != ''").Count(&stats.AccessNodes); res.Error != nil {
		return stats, errors.Wrap(res.Error, "couldn't get access node count")
	}
	if res := query.Where(condition).Where("COALESCE(public_config.domain, '') != '' AND (COALESCE(public_config.ipv4, '') != '' OR COALESCE(public_config.ipv6, '') != '')").Count(&stats.Gateways); res.Error != nil {
		return stats, errors.Wrap(res.Error, "couldn't get gateway count")
	}
	var distribution []NodesDistribution
	if res := d.gormDB.WithContext(ctx).Table("node").
		Select("country, count(node_id) as nodes").Where(condition).Group("country").Scan(&distribution); res.Error != nil {
		return stats, errors.Wrap(res.Error, "couldn't get nodes distribution")
	}
	if res := d.gormDB.WithContext(ctx).Table("node").Where(condition).Where("EXISTS( select node_gpu.id FROM node_gpu WHERE node_gpu.node_twin_id = node.twin_id)").Count(&stats.GPUs); res.Error != nil {
		return stats, errors.Wrap(res.Error, "couldn't get node with GPU count")
	}
	nodesDistribution := map[string]int64{}
	for _, d := range distribution {
		nodesDistribution[d.Country] = d.Nodes
	}
	stats.NodesDistribution = nodesDistribution

	nonDeletedNodeContracts := d.gormDB.Table("node_contract").
		Select("DISTINCT ON (node_id) node_id, contract_id").
		Where("state IN ('Created', 'GracePeriod')")

	res := d.gormDB.WithContext(ctx).Table("node").Where(condition).
		Joins(
			"LEFT JOIN rent_contract ON rent_contract.state IN ('Created', 'GracePeriod') AND rent_contract.node_id = node.node_id",
		).
		Joins(
			"LEFT JOIN (?) AS node_contract ON node_contract.node_id = node.node_id", nonDeletedNodeContracts,
		).
		Joins(
			"LEFT JOIN farm ON node.farm_id = farm.farm_id",
		).
		Where(
			"farm.dedicated_farm = true OR COALESCE(node_contract.contract_id, 0) = 0 OR COALESCE(rent_contract.contract_id, 0) != 0",
		).
		Count(&stats.DedicatedNodes)
	if res.Error != nil {
		return stats, errors.Wrap(res.Error, "couldn't get dedicated nodes count")
	}

	return stats, nil
}

// Scan is a custom decoder for jsonb filed. executed while scanning the node.
func (np *NodePower) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	if data, ok := value.([]byte); ok {
		return json.Unmarshal(data, np)
	}
	return fmt.Errorf("failed to unmarshal NodePower")
}

// GetNode returns node info
func (d *PostgresDatabase) GetNode(ctx context.Context, nodeID uint32) (Node, error) {
	q := d.nodeTableQuery()
	q = q.WithContext(ctx)
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

// GetFarm return farm info
func (d *PostgresDatabase) GetFarm(ctx context.Context, farmID uint32) (Farm, error) {
	q := d.farmTableQuery()
	q = q.WithContext(ctx)
	q = q.Where("farm.farm_id = ?", farmID)
	var farm Farm
	if res := q.Scan(&farm); res.Error != nil {
		return farm, errors.Wrap(res.Error, "failed to scan returned farm from database")
	}
	return farm, nil
}

//lint:ignore U1000 used for debugging
func convertParam(p interface{}) string {
	if v, ok := p.(string); ok {
		return fmt.Sprintf("'%s'", v)
	} else if v, ok := p.(uint64); ok {
		return fmt.Sprintf("%d", v)
	} else if v, ok := p.(int64); ok {
		return fmt.Sprintf("%d", v)
	} else if v, ok := p.(uint32); ok {
		return fmt.Sprintf("%d", v)
	} else if v, ok := p.(int); ok {
		return fmt.Sprintf("%d", v)
	} else if v, ok := p.(gridtypes.Unit); ok {
		return fmt.Sprintf("%d", v)
	}
	log.Error().Msgf("can't recognize type %s", fmt.Sprintf("%v", p))
	return "0"
}

// nolint
//
//lint:ignore U1000 used for debugging
func printQuery(query string, args ...interface{}) {
	for i, e := range args {
		query = strings.ReplaceAll(query, fmt.Sprintf("$%d", i+1), convertParam(e))
	}
	fmt.Printf("node query: %s", query)
}

func (d *PostgresDatabase) farmTableQuery() *gorm.DB {
	return d.gormDB.
		Table("farm").
		Select(
			"farm.id",
			"farm.farm_id",
			"farm.name",
			"farm.twin_id",
			"farm.pricing_policy_id",
			"farm.certification",
			"farm.stellar_address",
			"farm.dedicated_farm as dedicated",
			"COALESCE(public_ips.public_ips, '[]') as public_ips",
			"bool_or(node.rent_contract_id != 0)",
		).
		Joins(
			`left join(
				SELECT
					node.node_id,
					node.twin_id,
					node.farm_id,
					node.power,
					node.updated_at,
					node.certification,
					node.country,
					nodes_resources_view.free_mru,
					nodes_resources_view.free_hru,
					nodes_resources_view.free_sru,
					COALESCE(rent_contract.contract_id, 0) rent_contract_id,
					COALESCE(rent_contract.twin_id, 0) renter,
					COALESCE(node_gpu.id, '') gpu_id
				FROM node
				LEFT JOIN nodes_resources_view ON node.node_id = nodes_resources_view.node_id
				LEFT JOIN rent_contract ON node.node_id = rent_contract.node_id AND rent_contract.state IN ('Created', 'GracePeriod')
				LEFT JOIN node_gpu ON node.twin_id = node_gpu.node_twin_id
			) node on node.farm_id = farm.farm_id`,
		).
		Joins(`left join(
			SELECT
				p1.farm_id,
				COUNT(p1.id) total_ips,
				COUNT(CASE WHEN p2.contract_id = 0 THEN 1 END) free_ips
			FROM public_ip p1
			LEFT JOIN public_ip p2 ON p1.id = p2.id
			GROUP BY p1.farm_id
		) public_ip_count on public_ip_count.farm_id = farm.id`).
		Joins(`left join (
			select 
				farm_id,
				jsonb_agg(jsonb_build_object('id', id, 'ip', ip, 'contract_id', contract_id, 'gateway', gateway)) as public_ips
			from public_ip
			GROUP BY farm_id
		) public_ips on public_ips.farm_id = farm.id`).
		Group(
			`farm.id,
			farm.farm_id,
			farm.name,
			farm.twin_id,
			farm.pricing_policy_id,
			farm.certification,
			farm.stellar_address,
			farm.dedicated_farm,
			COALESCE(public_ips.public_ips, '[]')`,
		)
}

func (d *PostgresDatabase) nodeTableQuery() *gorm.DB {
	subquery := d.gormDB.Table("node_contract").
		Select("DISTINCT ON (node_id) node_id, contract_id").
		Where("state IN ('Created', 'GracePeriod')")

	nodeGPUQuery := `(SELECT count(node_gpu.id) FROM node_gpu WHERE node_gpu.node_twin_id = node.twin_id) as num_gpu`

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
			"farm.dedicated_farm as farm_dedicated",
			"rent_contract.contract_id as rent_contract_id",
			"rent_contract.twin_id as rented_by_twin_id",
			"node.serial_number",
			"convert_to_decimal(location.longitude) as longitude",
			"convert_to_decimal(location.latitude) as latitude",
			"node.power",
			"node.extra_fee",
			"CASE WHEN node_contract.contract_id IS NOT NULL THEN 1 ELSE 0 END AS has_node_contract",
			nodeGPUQuery,
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

// GetNodes returns nodes filtered and paginated
func (d *PostgresDatabase) GetNodes(ctx context.Context, filter types.NodeFilter, limit types.Limit) ([]Node, uint, error) {
	q := d.nodeTableQuery()
	q = q.WithContext(ctx)

	condition := "TRUE"
	if filter.Status != nil {
		condition = nodestatus.DecideNodeStatusCondition(*filter.Status)
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
		q = q.Where("(COALESCE(public_config.ipv4, '') = '') != ?", *filter.IPv4)
	}
	if filter.IPv6 != nil {
		q = q.Where("(COALESCE(public_config.ipv6, '') = '') != ?", *filter.IPv6)
	}
	if filter.Domain != nil {
		q = q.Where("(COALESCE(public_config.domain, '') = '') != ?", *filter.Domain)
	}
	if filter.CertificationType != nil {
		q = q.Where("node.certification ILIKE ?", *filter.CertificationType)
	}

	// Dedicated nodes filters
	if filter.InDedicatedFarm != nil {
		q = q.Where(`farm.dedicated_farm = ?`, *filter.InDedicatedFarm)
	}
	if filter.Dedicated != nil {
		q = q.Where(`? = (farm.dedicated_farm = true OR COALESCE(node_contract.contract_id, 0) = 0 OR COALESCE(rent_contract.contract_id, 0) != 0)`, *filter.Dedicated)
	}
	if filter.Rentable != nil {
		q = q.Where(`? = ((farm.dedicated_farm = true OR COALESCE(node_contract.contract_id, 0) = 0) AND COALESCE(rent_contract.contract_id, 0) = 0)`, *filter.Rentable)
	}
	if filter.AvailableFor != nil {
		q = q.Where(`COALESCE(rent_contract.twin_id, 0) = ? OR (COALESCE(rent_contract.twin_id, 0) = 0 AND farm.dedicated_farm = false)`, *filter.AvailableFor)
	}
	if filter.RentedBy != nil {
		q = q.Where(`COALESCE(rent_contract.twin_id, 0) = ?`, *filter.RentedBy)
	}
	if filter.Rented != nil {
		q = q.Where(`? = (COALESCE(rent_contract.contract_id, 0) != 0)`, *filter.Rented)
	}
	if filter.CertificationType != nil {
		q = q.Where("node.certification ILIKE ?", *filter.CertificationType)
	}
	if filter.OwnedBy != nil {
		q = q.Where(`COALESCE(farm.twin_id, 0) = ?`, *filter.OwnedBy)
	}

	/*
		used distinct selecting to avoid duplicated node after the join.
		- postgres apply WHERE before DISTINCT so filters will still filter on the whole data.
		- we don't return any gpu info on the node object so no worries of losing the data because DISTINCT.
	*/
	nodeGpuSubquery := d.gormDB.Table("node_gpu").
		Select("DISTINCT ON (node_twin_id) node_twin_id")

	if filter.HasGPU != nil {
		nodeGpuSubquery = nodeGpuSubquery.Where("(COALESCE(node_gpu.id, '') != '') = ?", *filter.HasGPU)
	}

	if filter.GpuDeviceName != nil {
		nodeGpuSubquery = nodeGpuSubquery.Where("COALESCE(node_gpu.device, '') ILIKE '%' || ? || '%'", *filter.GpuDeviceName)
	}

	if filter.GpuVendorName != nil {
		nodeGpuSubquery = nodeGpuSubquery.Where("COALESCE(node_gpu.vendor, '') ILIKE '%' || ? || '%'", *filter.GpuVendorName)
	}

	if filter.GpuVendorID != nil {
		nodeGpuSubquery = nodeGpuSubquery.Where("COALESCE(node_gpu.id, '') ILIKE '%' || ? || '%'", *filter.GpuVendorID)
	}

	if filter.GpuDeviceID != nil {
		nodeGpuSubquery = nodeGpuSubquery.Where("COALESCE(node_gpu.id, '') ILIKE '%' || ? || '%'", *filter.GpuDeviceID)
	}

	if filter.GpuAvailable != nil {
		nodeGpuSubquery = nodeGpuSubquery.Where("(COALESCE(node_gpu.contract, 0) = 0) = ?", *filter.GpuAvailable)
	}

	if filter.HasGPU != nil || filter.GpuDeviceName != nil || filter.GpuVendorName != nil || filter.GpuVendorID != nil ||
		filter.GpuDeviceID != nil || filter.GpuAvailable != nil {

		q.Joins(
			`INNER JOIN (?) AS gpu ON gpu.node_twin_id = node.twin_id`, nodeGpuSubquery,
		)
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

	// Sorting
	if limit.Randomize {
		q = q.Order("random()")
	} else {
		if filter.AvailableFor != nil {
			q = q.Order("(case when rent_contract is not null then 1 else 2 end)")
		}

		if limit.SortBy != "" {
			order := types.SortOrderAsc
			if strings.ToUpper(string(limit.SortOrder)) == string(types.SortOrderDesc) {
				order = types.SortOrderDesc
			}
			q = q.Order(fmt.Sprintf("%s %s", limit.SortBy, order))
		} else {
			q = q.Order("node.node_id")
		}
	}
	// Pagination
	q = q.Limit(int(limit.Size)).Offset(int(limit.Page-1) * int(limit.Size))

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

func (d *PostgresDatabase) shouldRetry(resError error) bool {
	if resError != nil && resError.Error() == ErrNodeResourcesViewNotFound.Error() {
		if err := d.initialize(); err != nil {
			log.Logger.Err(err).Msg("failed to reinitialize database")
		} else {
			return true
		}
	}
	return false
}

// GetFarms return farms filtered and paginated
func (d *PostgresDatabase) GetFarms(ctx context.Context, filter types.FarmFilter, limit types.Limit) ([]Farm, uint, error) {
	q := d.farmTableQuery()
	q = q.WithContext(ctx)
	if filter.NodeFreeMRU != nil {
		q = q.Where("node.free_mru >= ?", *filter.NodeFreeMRU)
	}
	if filter.NodeFreeHRU != nil {
		q = q.Where("node.free_hru >= ?", *filter.NodeFreeHRU)
	}
	if filter.NodeFreeSRU != nil {
		q = q.Where("node.free_sru >= ?", *filter.NodeFreeSRU)
	}

	if filter.NodeAvailableFor != nil {
		q = q.Where("node.renter = ? OR (node.renter = 0 AND farm.dedicated_farm = false)", *filter.NodeAvailableFor)
		q = q.Order("CASE WHEN bool_or(node.rent_contract_id != 0) = TRUE THEN 1 ELSE 2 END")
	}

	if filter.NodeHasGPU != nil {
		q = q.Where("(node.gpu_id != '') = ?", *filter.NodeHasGPU)
	}

	if filter.NodeRentedBy != nil {
		q = q.Where("node.renter = ?", *filter.NodeRentedBy)
	}

	if filter.Country != nil {
		q = q.Where("LOWER(node.country) = LOWER(?)", *filter.Country)
	}

	if filter.NodeStatus != nil {
		condition := nodestatus.DecideNodeStatusCondition(*filter.NodeStatus)
		q = q.Where(condition)
	}

	if filter.NodeCertified != nil {
		q = q.Where("(node.certification = 'Certified') = ?", *filter.NodeCertified)
	}

	if filter.FreeIPs != nil {
		q = q.Where("public_ip_count.free_ips >= ?", *filter.FreeIPs)
	}
	if filter.TotalIPs != nil {
		q = q.Where("public_ip_count.total_ips >= ?", *filter.TotalIPs)
	}
	if filter.StellarAddress != nil {
		q = q.Where("COALESCE(farm.stellar_address, '') = ?", *filter.StellarAddress)
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

	// Sorting
	if limit.Randomize {
		q = q.Order("random()")
	} else if limit.SortBy != "" {
		order := types.SortOrderAsc
		if strings.ToUpper(string(limit.SortOrder)) == string(types.SortOrderDesc) {
			order = types.SortOrderDesc
		}
		q = q.Order(fmt.Sprintf("%s %s", limit.SortBy, order))
	} else {
		q = q.Order("farm.farm_id")
	}

	// Pagination
	q = q.Limit(int(limit.Size)).Offset(int(limit.Page-1) * int(limit.Size))

	var farms []Farm
	if res := q.Scan(&farms); res.Error != nil {
		return farms, uint(count), errors.Wrap(res.Error, "failed to scan returned farm from database")
	}
	return farms, uint(count), nil
}

// GetTwins returns twins filtered and paginated
func (d *PostgresDatabase) GetTwins(ctx context.Context, filter types.TwinFilter, limit types.Limit) ([]types.Twin, uint, error) {
	q := d.gormDB.WithContext(ctx).
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

	// Sorting
	if limit.Randomize {
		q = q.Order("random()")
	} else if limit.SortBy != "" {
		order := types.SortOrderAsc
		if strings.ToUpper(string(limit.SortOrder)) == string(types.SortOrderDesc) {
			order = types.SortOrderDesc
		}
		q = q.Order(fmt.Sprintf("%s %s", limit.SortBy, order))
	} else {
		q = q.Order("twin.twin_id")
	}

	// Pagination
	q = q.Limit(int(limit.Size)).Offset(int(limit.Page-1) * int(limit.Size))

	twins := []types.Twin{}
	if res := q.Scan(&twins); res.Error != nil {
		return twins, uint(count), errors.Wrap(res.Error, "failed to scan returned twins from database")
	}
	return twins, uint(count), nil
}

// contractTableQuery union a contracts table from node/rent/name contracts tables
func (d *PostgresDatabase) contractTableQuery() *gorm.DB {
	contractTablesQuery := `(
		SELECT contract_id, twin_id, state, created_at, '' AS name, node_id, deployment_data, deployment_hash, number_of_public_i_ps, 'node' AS type
		FROM node_contract 
		UNION 
		SELECT contract_id, twin_id, state, created_at, '' AS name, node_id, '', '', 0, 'rent' AS type
		FROM rent_contract 
		UNION 
		SELECT contract_id, twin_id, state, created_at, name, 0, '', '', 0, 'name' AS type
		FROM name_contract
	) contracts`

	return d.gormDB.Table(contractTablesQuery).
		Select(
			"contracts.contract_id",
			"twin_id",
			"state",
			"created_at",
			"name",
			"node_id",
			"deployment_data",
			"deployment_hash",
			"number_of_public_i_ps as number_of_public_ips",
			"type",
		)
}

// GetContracts returns contracts filtered and paginated
func (d *PostgresDatabase) GetContracts(ctx context.Context, filter types.ContractFilter, limit types.Limit) ([]DBContract, uint, error) {
	q := d.contractTableQuery()
	q = q.WithContext(ctx)

	if filter.Type != nil {
		q = q.Where("type = ?", *filter.Type)
	}
	if filter.State != nil {
		q = q.Where("state ILIKE ?", *filter.State)
	}
	if filter.TwinID != nil {
		q = q.Where("twin_id = ?", *filter.TwinID)
	}
	if filter.ContractID != nil {
		q = q.Where("contracts.contract_id = ?", *filter.ContractID)
	}
	if filter.NodeID != nil {
		q = q.Where("node_id = ?", *filter.NodeID)
	}
	if filter.NumberOfPublicIps != nil {
		q = q.Where("number_of_public_i_ps >= ?", *filter.NumberOfPublicIps)
	}
	if filter.Name != nil {
		q = q.Where("name = ?", *filter.Name)
	}
	if filter.DeploymentData != nil {
		q = q.Where("deployment_data = ?", *filter.DeploymentData)
	}
	if filter.DeploymentHash != nil {
		q = q.Where("deployment_hash = ?", *filter.DeploymentHash)
	}
	var count int64
	if limit.Randomize || limit.RetCount {
		if res := q.Count(&count); res.Error != nil {
			return nil, 0, errors.Wrap(res.Error, "couldn't get contract count")
		}
	}

	// Sorting
	if limit.Randomize {
		q = q.Order("random()")
	} else if limit.SortBy != "" {
		order := types.SortOrderAsc
		if strings.ToUpper(string(limit.SortOrder)) == string(types.SortOrderDesc) {
			order = types.SortOrderDesc
		}
		q = q.Order(fmt.Sprintf("%s %s", limit.SortBy, order))
	} else {
		q = q.Order("contracts.contract_id")
	}

	// Pagination
	q = q.Limit(int(limit.Size)).Offset(int(limit.Page-1) * int(limit.Size))

	var contracts []DBContract
	if res := q.Scan(&contracts); res.Error != nil {
		return contracts, uint(count), errors.Wrap(res.Error, "failed to scan returned contracts from database")
	}
	return contracts, uint(count), nil
}

// GetContract return a single contract info
func (d *PostgresDatabase) GetContract(ctx context.Context, contractID uint32) (DBContract, error) {
	q := d.contractTableQuery()
	q = q.WithContext(ctx)

	q = q.Where("contracts.contract_id = ?", contractID)

	var contract DBContract
	res := q.Scan(&contract)

	if res.Error != nil {
		return DBContract{}, res.Error
	}
	if contract.ContractID == 0 {
		return DBContract{}, ErrContractNotFound
	}
	return contract, nil
}

// GetContract return a single contract info
func (d *PostgresDatabase) GetContractBills(ctx context.Context, contractID uint32, limit types.Limit) ([]ContractBilling, uint, error) {
	q := d.gormDB.WithContext(ctx).Table("contract_bill_report").
		Select("amount_billed, discount_received, timestamp").
		Where("contract_id = ?", contractID)

	q = q.Limit(int(limit.Size)).
		Offset(int(limit.Page-1) * int(limit.Size)).
		Order("timestamp DESC")

	var count int64
	if limit.RetCount {
		if res := q.Count(&count); res.Error != nil {
			return nil, 0, errors.Wrap(res.Error, "couldn't get contract bills count")
		}
	}

	var bills []ContractBilling
	if res := q.Scan(&bills); res.Error != nil {
		return bills, 0, errors.Wrap(res.Error, "failed to scan returned contract from database")
	}

	return bills, uint(count), nil
}

func (p *PostgresDatabase) UpsertNodesGPU(ctx context.Context, nodesGPU []types.NodeGPU) error {
	// For upsert operation
	conflictClause := clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}, {Name: "node_twin_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"vendor", "device", "contract"}),
	}
	err := p.gormDB.WithContext(ctx).Table("node_gpu").Clauses(conflictClause).Create(&nodesGPU).Error
	if err != nil {
		return fmt.Errorf("failed to upsert nodes GPU details: %w", err)
	}
	return nil
}
