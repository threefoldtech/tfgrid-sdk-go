package db

import (
	"context"
	"fmt"
	"strings"

	// to use for database/sql
	_ "github.com/lib/pq"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/nodestatus"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "embed"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var (
	// ErrNodeNotFound node not found
	ErrNodeNotFound = errors.New("node not found")
	// ErrFarmNotFound farm not found
	ErrFarmNotFound = errors.New("farm not found")
	//ErrViewNotFound
	ErrResourcesCacheTableNotFound = errors.New("ERROR: relation \"resources_cache\" does not exist (SQLSTATE 42P01)")
	// ErrContractNotFound contract not found
	ErrContractNotFound = errors.New("contract not found")
)

//go:embed setup.sql
var setupFile string

// PostgresDatabase postgres db client
type PostgresDatabase struct {
	gormDB     *gorm.DB
	connString string
}

func (d *PostgresDatabase) GetConnectionString() string {
	return d.connString
}

// NewPostgresDatabase returns a new postgres db client
func NewPostgresDatabase(host string, port int, user, password, dbname string, maxConns int, logLevel logger.LogLevel) (PostgresDatabase, error) {
	connString := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	gormDB, err := gorm.Open(postgres.Open(connString), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return PostgresDatabase{}, errors.Wrap(err, "failed to create orm wrapper around db")
	}
	sql, err := gormDB.DB()
	if err != nil {
		return PostgresDatabase{}, errors.Wrap(err, "failed to configure DB connection")
	}

	sql.SetMaxIdleConns(3)
	sql.SetMaxOpenConns(maxConns)

	res := PostgresDatabase{gormDB, connString}
	return res, nil
}

// Close the db connection
func (d *PostgresDatabase) Close() error {
	db, err := d.gormDB.DB()
	if err != nil {
		return err
	}
	return db.Close()
}

func (d *PostgresDatabase) Initialize() error {
	err := d.gormDB.AutoMigrate(
		&types.NodeGPU{},
		&types.HealthReport{},
		&types.Dmi{},
		&types.Speed{},
		&types.HasIpv6{},
		&types.NodesWorkloads{},
	)
	if err != nil {
		return errors.Wrap(err, "failed to migrate indexer tables")
	}

	err = d.gormDB.Exec(setupFile).Error
	if err != nil {
		return errors.Wrap(err, "failed to setup cache tables")
	}

	return nil
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
		condition = nodestatus.DecideNodeStatusCondition(filter.Status)
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

	if res := d.gormDB.WithContext(ctx).
		Table("node").
		Select(
			"count(COALESCE(public_config.ipv4, '') != '' OR COALESCE(public_config.ipv6, '') != '') as access_nodes",
			"count((COALESCE(public_config.ipv4, '') != '' OR COALESCE(public_config.ipv6, '') != '') AND COALESCE(public_config.domain, '') != '') as gateways",
		).
		Joins("RIGHT JOIN public_config ON node.id = public_config.node_id").
		Where(condition).
		Scan(&stats); res.Error != nil {
		return stats, errors.Wrap(res.Error, "couldn't get public config")
	}

	var distribution []NodesDistribution
	if res := d.gormDB.WithContext(ctx).Table("node").
		Select("country, count(node_id) as nodes").Where(condition).Group("country").Scan(&distribution); res.Error != nil {
		return stats, errors.Wrap(res.Error, "couldn't get nodes distribution")
	}
	nodesDistribution := map[string]int64{}
	for _, d := range distribution {
		nodesDistribution[d.Country] = d.Nodes
		stats.Nodes += d.Nodes
		stats.Countries++
	}
	stats.NodesDistribution = nodesDistribution

	if res := d.gormDB.WithContext(ctx).Table("node").Where(condition).
		Joins("LEFT JOIN resources_cache ON resources_cache.node_id = node.node_id").
		Where("node_gpu_count != 0").
		Count(&stats.GPUs); res.Error != nil {
		return stats, errors.Wrap(res.Error, "couldn't get node with GPU count")
	}

	res := d.gormDB.WithContext(ctx).Table("node").Where(condition).
		Joins(`
			LEFT JOIN resources_cache ON node.node_id = resources_cache.node_id
			LEFT JOIN farm ON node.farm_id = farm.farm_id
		`).
		Where(
			"farm.dedicated_farm = true OR resources_cache.node_contracts_count = 0 OR resources_cache.renter is not null",
		).
		Count(&stats.DedicatedNodes)
	if res.Error != nil {
		return stats, errors.Wrap(res.Error, "couldn't get dedicated nodes count")
	}

	if err := d.gormDB.WithContext(ctx).Table("node").
		Select("SUM(workloads_number) as workloads_number").
		Where(condition).
		Joins("LEFT JOIN node_workloads ON node.twin_id = node_workloads.node_twin_id").
		Scan(&stats.WorkloadsNumber).Error; err != nil {
		return stats, errors.Wrap(res.Error, "couldn't sum workloads number")
	}

	return stats, nil
}

// GetNode returns node info
func (d *PostgresDatabase) GetNode(ctx context.Context, nodeID uint32) (Node, error) {
	q := d.nodeTableQuery(ctx, types.NodeFilter{}, &gorm.DB{}, 0)
	q = q.Where("node.node_id = ?", nodeID)
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
	q := d.gormDB.WithContext(ctx).Table("farm")
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

func (d *PostgresDatabase) nodeTableQuery(ctx context.Context, filter types.NodeFilter, nodeGpuSubquery *gorm.DB, balance float64) *gorm.DB {
	calculatedDiscountColumn := fmt.Sprintf("calc_discount(resources_cache.price_usd, %f) AS price_usd", balance)
	q := d.gormDB.WithContext(ctx).
		Table("node").
		Select(
			"node.id",
			"node.node_id",
			"node.farm_id",
			"farm.name as farm_name",
			"node.twin_id",
			"node.country",
			"node.grid_version",
			"node.city",
			"node.uptime",
			"node.created",
			"node.farming_policy_id",
			"node.updated_at",
			"resources_cache.total_cru",
			"resources_cache.total_sru",
			"resources_cache.total_hru",
			"resources_cache.total_mru",
			"resources_cache.used_cru",
			"resources_cache.used_sru",
			"resources_cache.used_hru",
			"resources_cache.used_mru",
			"public_config.domain",
			"public_config.gw4",
			"public_config.gw6",
			"public_config.ipv4",
			"public_config.ipv6",
			"node.certification",
			"farm.dedicated_farm as farm_dedicated",
			"resources_cache.rent_contract_id as rent_contract_id",
			"(farm.dedicated_farm = true OR resources_cache.node_contracts_count = 0) AND resources_cache.renter is null as rentable",
			"resources_cache.renter is not null as rented",
			"resources_cache.renter",
			"node.serial_number",
			"convert_to_decimal(location.longitude) as longitude",
			"convert_to_decimal(location.latitude) as latitude",
			"node.power",
			"node.extra_fee",
			"resources_cache.node_contracts_count",
			"resources_cache.node_gpu_count AS num_gpu",
			"health_report.healthy",
			"node_ipv6.has_ipv6",
			"resources_cache.bios",
			"resources_cache.baseboard",
			"resources_cache.memory",
			"resources_cache.processor",
			"resources_cache.upload_speed",
			"resources_cache.download_speed",
			calculatedDiscountColumn,
		).
		Joins(`
			LEFT JOIN resources_cache ON node.node_id = resources_cache.node_id
			LEFT JOIN public_ips_cache ON public_ips_cache.farm_id = node.farm_id
			LEFT JOIN public_config ON node.id = public_config.node_id
			LEFT JOIN farm ON node.farm_id = farm.farm_id
			LEFT JOIN location ON node.location_id = location.id
			LEFT JOIN health_report ON node.twin_id = health_report.node_twin_id
			LEFT JOIN node_ipv6 ON node.twin_id = node_ipv6.node_twin_id
		`)

	if filter.HasGPU != nil || filter.GpuDeviceName != nil ||
		filter.GpuVendorName != nil || filter.GpuVendorID != nil ||
		filter.GpuDeviceID != nil || filter.GpuAvailable != nil {
		q.Joins(
			`RIGHT JOIN (?) AS gpu ON gpu.node_twin_id = node.twin_id`, nodeGpuSubquery,
		)
	}

	return q
}

func (d *PostgresDatabase) farmTableQuery(ctx context.Context, filter types.FarmFilter, nodeQuery *gorm.DB) *gorm.DB {
	q := d.gormDB.WithContext(ctx).
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
			"COALESCE(public_ips_cache.ips, '[]') as public_ips",
		).
		Joins(
			"LEFT JOIN public_ips_cache ON public_ips_cache.farm_id = farm.farm_id",
		)

	if filter.NodeAvailableFor != nil || filter.NodeFreeHRU != nil ||
		filter.NodeCertified != nil || filter.NodeFreeMRU != nil ||
		filter.NodeFreeSRU != nil || filter.NodeHasGPU != nil ||
		filter.NodeRentedBy != nil || (filter.NodeStatus != nil && len(filter.NodeStatus) != 0) ||
		filter.NodeTotalCRU != nil || filter.Country != nil ||
		filter.Region != nil {
		q.Joins(`RIGHT JOIN (?) AS resources_cache on resources_cache.farm_id = farm.farm_id`, nodeQuery).
			Group(`
				farm.id,
				farm.farm_id,
				farm.name,
				farm.twin_id,
				farm.pricing_policy_id,
				farm.certification,
				farm.stellar_address,
				farm.dedicated_farm,
				COALESCE(public_ips_cache.ips, '[]')
			`)
	}

	return q
}

// GetFarms return farms filtered and paginated
func (d *PostgresDatabase) GetFarms(ctx context.Context, filter types.FarmFilter, limit types.Limit) ([]Farm, uint, error) {
	nodeQuery := d.gormDB.Table("resources_cache").
		Select("resources_cache.farm_id", "renter").
		Joins("LEFT JOIN node ON node.node_id = resources_cache.node_id").
		Group(`resources_cache.farm_id, renter`)

	if filter.NodeFreeMRU != nil {
		nodeQuery = nodeQuery.Where("resources_cache.free_mru >= ?", *filter.NodeFreeMRU)
	}
	if filter.NodeFreeHRU != nil {
		nodeQuery = nodeQuery.Where("resources_cache.free_hru >= ?", *filter.NodeFreeHRU)
	}
	if filter.NodeFreeSRU != nil {
		nodeQuery = nodeQuery.Where("resources_cache.free_sru >= ?", *filter.NodeFreeSRU)
	}
	if filter.NodeTotalCRU != nil {
		nodeQuery = nodeQuery.Where("resources_cache.total_cru >= ?", *filter.NodeTotalCRU)
	}

	if filter.NodeHasGPU != nil {
		nodeQuery = nodeQuery.Where("(resources_cache.node_gpu_count > 0) = ?", *filter.NodeHasGPU)
	}

	if filter.NodeRentedBy != nil {
		nodeQuery = nodeQuery.Where("COALESCE(resources_cache.renter, 0) = ?", *filter.NodeRentedBy)
	}

	if filter.Country != nil {
		nodeQuery = nodeQuery.Where("LOWER(resources_cache.country) = LOWER(?)", *filter.Country)
	}

	if filter.Region != nil {
		nodeQuery = nodeQuery.Where("LOWER(resources_cache.region) = LOWER(?)", *filter.Region)
	}

	if filter.NodeStatus != nil && len(filter.NodeStatus) != 0 {
		condition := nodestatus.DecideNodeStatusCondition(filter.NodeStatus)
		nodeQuery = nodeQuery.Where(condition)
	}

	if filter.NodeCertified != nil {
		nodeQuery = nodeQuery.Where("(node.certification = 'Certified') = ?", *filter.NodeCertified)
	}

	q := d.farmTableQuery(ctx, filter, nodeQuery)

	if filter.NodeAvailableFor != nil {
		q = q.Where("COALESCE(resources_cache.renter, 0) = ? OR (resources_cache.renter IS NULL AND farm.dedicated_farm = false)", *filter.NodeAvailableFor)
	}

	if filter.FreeIPs != nil {
		q = q.Where("COALESCE(public_ips_cache.free_ips, 0) >= ?", *filter.FreeIPs)
	}
	if filter.TotalIPs != nil {
		q = q.Where("COALESCE(public_ips_cache.total_ips, 0) >= ?", *filter.TotalIPs)
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

	// Sorting
	if limit.Randomize {
		q = q.Order("random()")
	} else {
		if filter.NodeAvailableFor != nil {
			q = q.Order("(bool_or(resources_cache.renter IS NOT NULL)) DESC")
		}
		if limit.SortBy != "" {
			order := types.SortOrderAsc
			if strings.EqualFold(string(limit.SortOrder), string(types.SortOrderDesc)) {
				order = types.SortOrderDesc
			}
			q = q.Order(fmt.Sprintf("%s %s", limit.SortBy, order))
		} else {
			q = q.Order("farm.farm_id")
		}
	}

	var count int64
	if limit.RetCount {
		countQuery := q.Session(&gorm.Session{})
		if res := countQuery.Count(&count); res.Error != nil {
			return nil, 0, errors.Wrap(res.Error, "couldn't get farm count")
		}
	}

	// Pagination
	q = q.Limit(int(limit.Size)).Offset(int(limit.Page-1) * int(limit.Size))

	var farms []Farm
	if res := q.Scan(&farms); res.Error != nil {
		return farms, uint(count), errors.Wrap(res.Error, "failed to scan returned farm from database")
	}
	return farms, uint(count), nil
}

// GetNodes returns nodes filtered and paginated
func (d *PostgresDatabase) GetNodes(ctx context.Context, filter types.NodeFilter, limit types.Limit) ([]Node, uint, error) {
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

	q := d.nodeTableQuery(ctx, filter, nodeGpuSubquery, limit.Balance)

	condition := "TRUE"
	if filter.Status != nil {
		condition = nodestatus.DecideNodeStatusCondition(filter.Status)
	}

	q = q.Where(condition)

	if filter.NumGPU != nil {
		q = q.Where("COALESCE(resources_cache.node_gpu_count, 0) >= ?", *filter.NumGPU)
	}
	if filter.Healthy != nil {
		q = q.Where("health_report.healthy = ? ", *filter.Healthy)
	}
	if filter.HasIpv6 != nil {
		q = q.Where("COALESCE(node_ipv6.has_ipv6, false) = ? ", *filter.HasIpv6)
	}
	if filter.FreeMRU != nil {
		q = q.Where("resources_cache.free_mru >= ?", *filter.FreeMRU)
	}
	if filter.FreeHRU != nil {
		q = q.Where("resources_cache.free_hru >= ?", *filter.FreeHRU)
	}
	if filter.FreeSRU != nil {
		q = q.Where("resources_cache.free_sru >= ?", *filter.FreeSRU)
	}
	if filter.TotalCRU != nil {
		q = q.Where("resources_cache.total_cru >= ?", *filter.TotalCRU)
	}
	if filter.TotalHRU != nil {
		q = q.Where("resources_cache.total_hru >= ?", *filter.TotalHRU)
	}
	if filter.TotalMRU != nil {
		q = q.Where("resources_cache.total_mru >= ?", *filter.TotalMRU)
	}
	if filter.TotalSRU != nil {
		q = q.Where("resources_cache.total_sru >= ?", *filter.TotalSRU)
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
	if filter.Region != nil {
		q = q.Where("LOWER(resources_cache.region) = LOWER(?)", *filter.Region)
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
		q = q.Where("COALESCE(public_ips_cache.free_ips, 0) >= ?", *filter.FreeIPs)
	}
	if filter.IPv4 != nil {
		q = q.Where("(COALESCE(public_config.ipv4, '') != '') = ?", *filter.IPv4)
	}
	if filter.IPv6 != nil {
		q = q.Where("(COALESCE(public_config.ipv6, '') != '') = ?", *filter.IPv6)
	}
	if filter.Domain != nil {
		q = q.Where("(COALESCE(public_config.domain, '') != '') = ?", *filter.Domain)
	}
	if filter.CertificationType != nil {
		q = q.Where("node.certification ILIKE ?", *filter.CertificationType)
	}
	if filter.Excluded != nil {
		q = q.Where("node.node_id NOT IN ?", filter.Excluded)
	}

	// Dedicated nodes filters
	if filter.InDedicatedFarm != nil {
		q = q.Where(`farm.dedicated_farm = ?`, *filter.InDedicatedFarm)
	}
	if filter.Dedicated != nil {
		q = q.Where(`? = (farm.dedicated_farm = true OR resources_cache.node_contracts_count = 0 OR resources_cache.renter is not null)`, *filter.Dedicated)
	}
	if filter.Rentable != nil {
		q = q.Where(`? = ((farm.dedicated_farm = true OR resources_cache.node_contracts_count = 0) AND resources_cache.renter is null)`, *filter.Rentable)
	}
	if filter.AvailableFor != nil {
		q = q.Where(`COALESCE(resources_cache.renter, 0) = ? OR (resources_cache.renter is null AND farm.dedicated_farm = false)`, *filter.AvailableFor)
	}
	if filter.RentedBy != nil {
		q = q.Where(`COALESCE(resources_cache.renter, 0) = ?`, *filter.RentedBy)
	}
	if filter.Rented != nil {
		q = q.Where(`? = (resources_cache.renter is not null)`, *filter.Rented)
	}
	if filter.OwnedBy != nil {
		q = q.Where(`COALESCE(farm.twin_id, 0) = ?`, *filter.OwnedBy)
	}
	if filter.PriceMin != nil {
		q = q.Where(`calc_discount(resources_cache.price_usd, ?) >= ?`, limit.Balance, *filter.PriceMin)
	}
	if filter.PriceMax != nil {
		q = q.Where(`calc_discount(resources_cache.price_usd, ?) <= ?`, limit.Balance, *filter.PriceMax)
	}

	// Sorting
	if limit.Randomize {
		q = q.Order("random()")
	} else {
		if filter.AvailableFor != nil {
			q = q.Order("(case when resources_cache.renter is not null then 1 else 2 end)")
		}

		if limit.SortBy != "" {
			order := types.SortOrderAsc
			if strings.EqualFold(string(limit.SortOrder), string(types.SortOrderDesc)) {
				order = types.SortOrderDesc
			}

			if limit.SortBy == "status" {
				q = q.Order(nodestatus.DecideNodeStatusOrdering(order))
			} else if limit.SortBy == "free_cru" {
				q = q.Order(fmt.Sprintf("total_cru-used_cru %s", order))
			} else {
				q = q.Order(fmt.Sprintf("%s %s", limit.SortBy, order))
			}
		} else {
			q = q.Order("node.node_id")
		}
	}

	var count int64
	if limit.RetCount {
		countQuery := q.Session(&gorm.Session{})
		if res := countQuery.Count(&count); res.Error != nil {
			return nil, 0, errors.Wrap(res.Error, "couldn't get node count")
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
	if resError != nil && resError.Error() == ErrResourcesCacheTableNotFound.Error() {
		if err := d.Initialize(); err != nil {
			log.Logger.Err(err).Msg("failed to reinitialize database")
		} else {
			return true
		}
	}
	return false
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

	// Sorting
	if limit.Randomize {
		q = q.Order("random()")
	} else if limit.SortBy != "" {
		order := types.SortOrderAsc
		if strings.EqualFold(string(limit.SortOrder), string(types.SortOrderDesc)) {
			order = types.SortOrderDesc
		}
		q = q.Order(fmt.Sprintf("%s %s", limit.SortBy, order))
	} else {
		q = q.Order("twin.twin_id")
	}

	var count int64
	if limit.RetCount {
		countQuery := q.Session(&gorm.Session{})
		if res := countQuery.Count(&count); res.Error != nil {
			return nil, 0, errors.Wrap(res.Error, "couldn't get twin count")
		}
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
		UNION ALL
		SELECT contract_id, twin_id, state, created_at, '' AS name, node_id, '', '', 0, 'rent' AS type
		FROM rent_contract 
		UNION ALL
		SELECT contract_id, twin_id, state, created_at, name, 0, '', '', 0, 'name' AS type
		FROM name_contract
	) contracts`

	return d.gormDB.Table(contractTablesQuery).
		Select(
			"contracts.contract_id",
			"contracts.twin_id",
			"state",
			"created_at",
			"contracts.name",
			"node_id",
			"deployment_data",
			"deployment_hash",
			"number_of_public_i_ps as number_of_public_ips",
			"type",
			"farm.name as farm_name",
			"farm.farm_id",
		).
		Joins(`LEFT JOIN farm ON farm.farm_id = (
			SELECT farm_id from node WHERE node.node_id = contracts.node_id
		)`)
}

// GetContracts returns contracts filtered and paginated
func (d *PostgresDatabase) GetContracts(ctx context.Context, filter types.ContractFilter, limit types.Limit) ([]DBContract, uint, error) {
	q := d.contractTableQuery()
	q = q.WithContext(ctx)

	if filter.Type != nil {
		q = q.Where("type = ?", *filter.Type)
	}
	if len(filter.State) != 0 {
		states := []string{}
		for _, s := range filter.State {
			states = append(states, strings.ToLower(s))
		}

		q = q.Where("LOWER(state) IN ?", states)
	}
	if filter.TwinID != nil {
		q = q.Where("contracts.twin_id = ?", *filter.TwinID)
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
		q = q.Where("contracts.name = ?", *filter.Name)
	}
	if filter.DeploymentData != nil {
		q = q.Where("deployment_data = ?", *filter.DeploymentData)
	}
	if filter.DeploymentHash != nil {
		q = q.Where("deployment_hash = ?", *filter.DeploymentHash)
	}
	if filter.FarmName != nil {
		q = q.Where("farm.name ILIKE ?", *filter.FarmName)
	}
	if filter.FarmId != nil {
		q = q.Where("farm.farm_id = ?", *filter.FarmId)
	}

	// Sorting
	if limit.Randomize {
		q = q.Order("random()")
	} else if limit.SortBy != "" {
		order := types.SortOrderAsc
		if strings.EqualFold(string(limit.SortOrder), string(types.SortOrderDesc)) {
			order = types.SortOrderDesc
		}
		q = q.Order(fmt.Sprintf("%s %s", limit.SortBy, order))
	} else {
		q = q.Order("contracts.contract_id")
	}

	var count int64
	if limit.Randomize || limit.RetCount {
		countQuery := q.Session(&gorm.Session{})
		if res := countQuery.Count(&count); res.Error != nil {
			return nil, 0, errors.Wrap(res.Error, "couldn't get contract count")
		}
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

// GetContract return a single contract all billing reports
func (d *PostgresDatabase) GetContractBills(ctx context.Context, contractID uint32, limit types.Limit) ([]ContractBilling, uint, error) {
	q := d.gormDB.WithContext(ctx).Table("contract_bill_report").
		Select("amount_billed, discount_received, timestamp").
		Where("contract_id = ?", contractID)

	q = q.Limit(int(limit.Size)).
		Offset(int(limit.Page-1) * int(limit.Size)).
		Order("timestamp DESC")

	var count int64
	if limit.RetCount {
		countQuery := q.Session(&gorm.Session{})
		if res := countQuery.Count(&count); res.Error != nil {
			return nil, 0, errors.Wrap(res.Error, "couldn't get contract bills count")
		}
	}

	var bills []ContractBilling
	if res := q.Scan(&bills); res.Error != nil {
		return bills, 0, errors.Wrap(res.Error, "failed to scan returned contract from database")
	}

	return bills, uint(count), nil
}

// GetContractsLatestBillReports return latest reports for some contracts
func (d *PostgresDatabase) GetContractsLatestBillReports(ctx context.Context, contractsIds []uint32, limit uint) ([]ContractBilling, error) {
	// WITH: a CTE to create a tmp table
	// ROW_NUMBER(): function is a window function that assigns a sequential integer (rn) to each row
	// PARTITION BY: ranking is for each contract_id separately
	// ranking is done in desc order on timestamp
	q := d.gormDB.Raw(`
        WITH ranked_bill_reports AS (
            SELECT
				timestamp, amount_billed, contract_id,
				ROW_NUMBER() OVER (PARTITION BY contract_id ORDER BY timestamp DESC) as rn
            FROM
				contract_bill_report
            WHERE
                contract_id IN (?)
        )
		
        SELECT timestamp, amount_billed, contract_id
        FROM ranked_bill_reports
        WHERE rn <= ?;
		`, contractsIds, limit)

	var reports []ContractBilling
	if res := q.Scan(&reports); res.Error != nil {
		return reports, res.Error
	}

	return reports, nil
}

// GetContractsTotalBilledAmount return a sum of all billed amount
func (d *PostgresDatabase) GetContractsTotalBilledAmount(ctx context.Context, contractIds []uint32) (uint64, error) {
	q := d.gormDB.Raw(`SELECT sum(amount_billed) FROM contract_bill_report WHERE contract_id IN (?);`, contractIds)

	var total uint64
	if res := q.Scan(&total); res.Error != nil {
		return 0, res.Error
	}

	return total, nil
}
