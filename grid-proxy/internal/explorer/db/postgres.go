package db

import (
	"encoding/json"
	"fmt"
	"math/rand"
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
	CREATE INDEX IF NOT EXISTS idx_node_id ON public.node(node_id);
	CREATE INDEX IF NOT EXISTS idx_twin_id ON public.twin(twin_id);
	CREATE INDEX IF NOT EXISTS idx_farm_id ON public.farm(farm_id);
	CREATE INDEX IF NOT EXISTS idx_contract_id ON public.generic_contract(contract_id);
	CREATE INDEX IF NOT EXISTS idx_farms_cache_farm_id ON public.farms_cache(farm_id);
	CREATE INDEX IF NOT EXISTS idx_nodes_cache_node_id ON public.nodes_cache(node_id);
	
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
	$$ LANGUAGE plpgsql;`
)

// PostgresDatabase postgres db client
type PostgresDatabase struct {
	gormDB *gorm.DB
}

// NewPostgresDatabase returns a new postgres db client
func NewPostgresDatabase(host string, port int, user, password, dbname string, maxConns int) (Database, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	gormDB, err := gorm.Open(postgres.Open(psqlInfo), &gorm.Config{})
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

	res := PostgresDatabase{gormDB}
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

// GetCounters returns aggregate info about the grid
func (d *PostgresDatabase) GetCounters(filter types.StatsFilter) (types.Counters, error) {
	var counters types.Counters
	if res := d.gormDB.Table("twin").Count(&counters.Twins); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get twin count")
	}
	if res := d.gormDB.Table("public_ip").Count(&counters.PublicIPs); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get public ip count")
	}

	if res := d.gormDB.Table("generic_contract").Count(&counters.Contracts); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get contracts count")
	}

	if res := d.gormDB.Table("farm").Distinct("farm_id").Count(&counters.Farms); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get farm count")
	}

	condition := "TRUE"
	if filter.Status != nil {
		condition = nodestatus.DecideNodeStatusCondition(*filter.Status)
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
	if res := d.gormDB.Table("node").
		Where(condition).Count(&counters.Nodes); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get node count")
	}
	if res := d.gormDB.Table("node").
		Where(condition).Distinct("country").Count(&counters.Countries); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get country count")
	}
	query := d.gormDB.
		Table("node").
		Joins(
			`RIGHT JOIN public_config
			ON node.id = public_config.node_id
			`,
		)

	if res := query.Where(condition).Where("COALESCE(public_config.ipv4, '') != '' OR COALESCE(public_config.ipv6, '') != ''").Count(&counters.AccessNodes); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get access node count")
	}
	if res := query.Where(condition).Where("COALESCE(public_config.domain, '') != '' AND (COALESCE(public_config.ipv4, '') != '' OR COALESCE(public_config.ipv6, '') != '')").Count(&counters.Gateways); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get gateway count")
	}
	var distribution []NodesDistribution
	if res := d.gormDB.Table("node").
		Select("country, count(node_id) as nodes").Where(condition).Group("country").Scan(&distribution); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get nodes distribution")
	}
	if res := d.gormDB.Table("node").Where(condition).Where("EXISTS( select node_gpu.id FROM node_gpu WHERE node_gpu.node_twin_id = node.twin_id)").Count(&counters.GPUs); res.Error != nil {
		return counters, errors.Wrap(res.Error, "couldn't get node with GPU count")
	}
	nodesDistribution := map[string]int64{}
	for _, d := range distribution {
		nodesDistribution[d.Country] = d.Nodes
	}
	counters.NodesDistribution = nodesDistribution
	return counters, nil
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

// GetFarm return farm info
func (d *PostgresDatabase) GetFarm(farmID uint32) (Farm, error) {
	q := d.gormDB.Table("farm").Select(
		"farm.id",
		"farm.farm_id",
		"farm.name",
		"farm.twin_id",
		"farm.pricing_policy_id",
		"farm.certification",
		"farm.stellar_address",
		"farm.dedicated_farm as dedicated",
		"COALESCE(farms_cache.ips, '[]') as public_ips",
	).
		Joins(`
		LEFT JOIN farms_cache ON farm.farm_id = farms_cache.farm_id
	`)

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

func (d *PostgresDatabase) farmTableQuery(nodeQuery *gorm.DB) *gorm.DB {
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
			"COALESCE(farms_cache.ips, '[]') as public_ips",
		).
		Joins("RIGHT JOIN (?) AS node ON node.farm_id = farm.farm_id", nodeQuery).
		Joins(`
			LEFT JOIN farms_cache ON farm.farm_id = farms_cache.farm_id
		`)
}

func (d *PostgresDatabase) nodeTableQuery() *gorm.DB {
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
			"node.total_cru",
			"node.total_sru",
			"node.total_hru",
			"node.total_mru",
			"nodes_cache.free_cru",
			"nodes_cache.free_sru",
			"nodes_cache.free_hru",
			"nodes_cache.free_mru",
			"public_config.domain",
			"public_config.gw4",
			"public_config.gw6",
			"public_config.ipv4",
			"public_config.ipv6",
			"node.certification",
			"farm.dedicated_farm as dedicated",
			"nodes_cache.renter",
			"nodes_cache.rent_contract_id",
			"node.serial_number",
			"convert_to_decimal(location.longitude) as longitude",
			"convert_to_decimal(location.latitude) as latitude",
			"node.power",
			"node.extra_fee",
			nodeGPUQuery,
		).
		Joins(
			"LEFT JOIN nodes_cache ON node.node_id = nodes_cache.node_id",
		).
		Joins(
			"LEFT JOIN public_config ON node.id = public_config.node_id",
		).
		Joins(
			"LEFT JOIN farm ON node.farm_id = farm.farm_id",
		).
		Joins(
			"LEFT JOIN location ON node.location_id = location.id",
		).
		Joins(
			"LEFT JOIN node_gpu on node_gpu.node_twin_id = node.twin_id",
		).
		Joins(
			"LEFT JOIN farms_cache ON node.farm_id = farms_cache.farm_id",
		)
}

// GetNodes returns nodes filtered and paginated
func (d *PostgresDatabase) GetNodes(filter types.NodeFilter, limit types.Limit) ([]Node, uint, error) {
	q := d.nodeTableQuery()
	q = q.Session(&gorm.Session{})

	condition := "TRUE"
	if filter.Status != nil {
		condition = nodestatus.DecideNodeStatusCondition(*filter.Status)
	}

	q = q.Where(condition)

	if filter.FreeMRU != nil {
		q = q.Where("nodes_cache.free_mru >= ?", *filter.FreeMRU)
	}
	if filter.FreeHRU != nil {
		q = q.Where("nodes_cache.free_hru >= ?", *filter.FreeHRU)
	}
	if filter.FreeSRU != nil {
		q = q.Where("nodes_cache.free_sru >= ?", *filter.FreeSRU)
	}
	if filter.TotalCRU != nil {
		q = q.Where("node.total_cru >= ?", *filter.TotalCRU)
	}
	if filter.TotalHRU != nil {
		q = q.Where("node.total_hru >= ?", *filter.TotalHRU)
	}
	if filter.TotalMRU != nil {
		q = q.Where("node.total_mru >= ?", *filter.TotalMRU)
	}
	if filter.TotalSRU != nil {
		q = q.Where("node.total_sru >= ?", *filter.TotalSRU)
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
		q = q.Where("farms_cache.free_ips >= ?", *filter.FreeIPs)
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
	if filter.Dedicated != nil {
		q = q.Where("farm.dedicated_farm = ?", *filter.Dedicated)
	}
	if filter.Rentable != nil {
		q = q.Where(`? = ((farm.dedicated_farm = true OR COALESCE(nodes_cache.node_contracts, 0) = 0) AND COALESCE(nodes_cache.renter, 0) = 0)`, *filter.Rentable)
	}
	if filter.RentedBy != nil {
		q = q.Where(`COALESCE(nodes_cache.renter, 0) = ?`, *filter.RentedBy)
	}
	if filter.AvailableFor != nil {
		q = q.Where(`COALESCE(nodes_cache.renter, 0) = ? OR (COALESCE(nodes_cache.renter, 0) = 0 AND farm.dedicated_farm = false)`, *filter.AvailableFor)
	}
	if filter.Rented != nil {
		q = q.Where(`? = (COALESCE(nodes_cache.renter, 0) != 0)`, *filter.Rented)
	}
	if filter.CertificationType != nil {
		q = q.Where("node.certification ILIKE ?", *filter.CertificationType)
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
	if limit.Randomize {
		q = q.Limit(int(limit.Size)).
			Offset(int(rand.Intn(int(count)) - int(limit.Size)))
	} else {
		if filter.AvailableFor != nil {
			q = q.Order("(case when COALESCE(nodes_cache.renter, 0) != 0 then 1 else 2 end)")
		}
		q = q.Limit(int(limit.Size)).
			Offset(int(limit.Page-1) * int(limit.Size)).
			Order("node.node_id")
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
func (d *PostgresDatabase) GetFarms(filter types.FarmFilter, limit types.Limit) ([]Farm, uint, error) {
	nodeQuery := d.gormDB.Table("node").Select(
		"node.farm_id",
		"bool_or(COALESCE(nodes_cache.rent_contract_id, 0) != 0) as rented",
	).Joins(`
		LEFT JOIN nodes_cache ON node.node_id = nodes_cache.node_id
	`).Joins(`
		LEFT JOIN node_gpu ON node.twin_id = node_gpu.node_twin_id
	`).Group("node.farm_id")

	if filter.NodeFreeMRU != nil {
		nodeQuery = nodeQuery.Where("nodes_cache.free_mru >= ?", *filter.NodeFreeMRU)
	}
	if filter.NodeFreeHRU != nil {
		nodeQuery = nodeQuery.Where("nodes_cache.free_hru >= ?", *filter.NodeFreeHRU)
	}
	if filter.NodeFreeSRU != nil {
		nodeQuery = nodeQuery.Where("nodes_cache.free_sru >= ?", *filter.NodeFreeSRU)
	}

	if filter.NodeAvailableFor != nil {
		nodeQuery = nodeQuery.Where("nodes_cache.renter = ? OR (nodes_cache.renter = 0 AND nodes_cache.dedicated_farm = false)", *filter.NodeAvailableFor)
	}

	if filter.NodeHasGPU != nil {
		nodeQuery = nodeQuery.Where("(COALESCE(node_gpu.id, '') != '') = ?", *filter.NodeHasGPU)
	}

	if filter.NodeRentedBy != nil {
		nodeQuery = nodeQuery.Where("nodes_cache.renter = ?", *filter.NodeRentedBy)
	}

	if filter.Country != nil {
		nodeQuery = nodeQuery.Where("LOWER(node.country) = LOWER(?)", *filter.Country)
	}

	if filter.NodeStatus != nil {
		condition := nodestatus.DecideNodeStatusCondition(*filter.NodeStatus)
		nodeQuery = nodeQuery.Where(condition)
	}

	if filter.NodeCertified != nil {
		nodeQuery = nodeQuery.Where("(node.certification = 'Certified') = ?", *filter.NodeCertified)
	}

	q := d.farmTableQuery(nodeQuery)

	if filter.FreeIPs != nil {
		q = q.Where("farms_cache.free_ips >= ?", *filter.FreeIPs)
	}
	if filter.TotalIPs != nil {
		q = q.Where("farms_cache.total_ips >= ?", *filter.TotalIPs)
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
		if filter.NodeAvailableFor != nil {
			q = q.Order("CASE WHEN node.rented = TRUE THEN 1 ELSE 2 END")
		}
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

func (d *PostgresDatabase) contractTableQuery() *gorm.DB {
	return d.gormDB.Table("generic_contract").
		Select(
			"generic_contract.contract_id",
			"twin_id",
			"state",
			"created_at",
			"COALESCE(name, '') as name",
			"COALESCE(node_id, 0) as node_id",
			"COALESCE(deployment_data, '') as deployment_data",
			"COALESCE(deployment_hash, '') as deployment_hash",
			"COALESCE(number_of_public_i_ps, 0) as number_of_public_ips",
			"type",
		)
}

// GetContracts returns contracts filtered and paginated
func (d *PostgresDatabase) GetContracts(filter types.ContractFilter, limit types.Limit) ([]DBContract, uint, error) {
	q := d.contractTableQuery()

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
		q = q.Where("generic_contract.contract_id = ?", *filter.ContractID)
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
	if limit.Randomize {
		q = q.Limit(int(limit.Size)).
			Offset(int(rand.Intn(int(count)) - int(limit.Size)))
	} else {
		q = q.Limit(int(limit.Size)).
			Offset(int(limit.Page-1) * int(limit.Size)).
			Order("generic_contract.contract_id")
	}
	var contracts []DBContract
	if res := q.Scan(&contracts); res.Error != nil {
		return contracts, uint(count), errors.Wrap(res.Error, "failed to scan returned contracts from database")
	}

	return contracts, uint(count), nil
}

// GetContract return a single contract info
func (d *PostgresDatabase) GetContract(contractID uint32) (DBContract, error) {
	q := d.contractTableQuery()
	q = q.Where("generic_contract.contract_id = ?", contractID)

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
func (d *PostgresDatabase) GetContractBills(contractID uint32, limit types.Limit) ([]ContractBilling, uint, error) {
	q := d.gormDB.Table("contract_bill_report").
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

func (p *PostgresDatabase) UpsertNodesGPU(nodesGPU []types.NodeGPU) error {
	// For upsert operation
	conflictClause := clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}, {Name: "node_twin_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"vendor", "device", "contract"}),
	}
	err := p.gormDB.Table("node_gpu").Clauses(conflictClause).Create(&nodesGPU).Error
	if err != nil {
		return fmt.Errorf("failed to upsert nodes GPU details: %w", err)
	}
	return nil
}
