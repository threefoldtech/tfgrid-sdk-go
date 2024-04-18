package mock

import (
	"database/sql"
	"encoding/json"
	"math"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"gorm.io/gorm"
)

const deleted = "Deleted"

type DBData struct {
	NodeIDMap          map[string]uint64
	FarmIDMap          map[string]uint64
	FreeIPs            map[uint64]uint64
	TotalIPs           map[uint64]uint64
	NodeUsedResources  map[uint64]NodeResourcesTotal
	NodeRentedBy       map[uint64]uint64
	NodeRentContractID map[uint64]uint64
	FarmHasRentedNode  map[uint64]map[uint64]bool

	Nodes               map[uint64]Node
	NodeTotalResources  map[uint64]NodeResourcesTotal
	Farms               map[uint64]Farm
	Twins               map[uint64]Twin
	PublicIPs           map[string]PublicIp
	PublicConfigs       map[uint64]PublicConfig
	NodeContracts       map[uint64]NodeContract
	RentContracts       map[uint64]RentContract
	NameContracts       map[uint64]NameContract
	Billings            map[uint64][]ContractBillReport
	BillReports         uint32
	ContractResources   map[string]ContractResources
	NonDeletedContracts map[uint64][]uint64
	GPUs                map[uint32][]types.NodeGPU
	Regions             map[string]string
	Locations           map[string]Location
	HealthReports       map[uint32]bool
	DMIs                map[uint32]types.Dmi
	Speeds              map[uint32]types.Speed
	PricingPolicies     map[uint]PricingPolicy

	DB *sql.DB
}

func loadNodes(db *sql.DB, gormDB *gorm.DB, data *DBData) error {
	var nodes []Node
	err := gormDB.Table("node").Scan(&nodes).Error
	if err != nil {
		return err
	}
	for _, node := range nodes {
		data.Nodes[node.NodeID] = node
		data.NodeIDMap[node.ID] = node.NodeID
	}
	return nil
}

func calcNodesUsedResources(data *DBData) error {

	for _, node := range data.Nodes {
		used := NodeResourcesTotal{
			MRU: uint64(2 * gridtypes.Gigabyte),
			SRU: uint64(20 * gridtypes.Gigabyte),
		}
		tenpercent := uint64(math.Round(float64(data.NodeTotalResources[node.NodeID].MRU) / 10))
		if used.MRU < tenpercent {
			used.MRU = tenpercent
		}
		data.NodeUsedResources[node.NodeID] = used
	}

	for _, contract := range data.NodeContracts {
		if contract.State == deleted {
			continue
		}
		contratResourceID := contract.ResourcesUsedID
		data.NodeUsedResources[contract.NodeID] = NodeResourcesTotal{
			CRU: data.ContractResources[contratResourceID].CRU + data.NodeUsedResources[contract.NodeID].CRU,
			MRU: data.ContractResources[contratResourceID].MRU + data.NodeUsedResources[contract.NodeID].MRU,
			HRU: data.ContractResources[contratResourceID].HRU + data.NodeUsedResources[contract.NodeID].HRU,
			SRU: data.ContractResources[contratResourceID].SRU + data.NodeUsedResources[contract.NodeID].SRU,
		}

	}
	return nil
}

func calcRentInfo(data *DBData) error {
	for _, contract := range data.RentContracts {
		if contract.State == deleted {
			continue
		}
		data.NodeRentedBy[contract.NodeID] = contract.TwinID
		data.NodeRentContractID[contract.NodeID] = contract.ContractID
		farmID := data.Nodes[contract.NodeID].FarmID
		data.FarmHasRentedNode[farmID][contract.TwinID] = true
	}
	return nil
}

func calcFreeIPs(data *DBData) error {
	for _, publicIP := range data.PublicIPs {
		if publicIP.ContractID == 0 {
			data.FreeIPs[data.FarmIDMap[publicIP.FarmID]]++
		}
		data.TotalIPs[data.FarmIDMap[publicIP.FarmID]]++
	}
	return nil
}

func loadNodesTotalResources(db *sql.DB, data *DBData) error {
	rows, err := db.Query(`
	SELECT
		COALESCE(id, ''),
		COALESCE(hru, 0),
		COALESCE(sru, 0),
		COALESCE(cru, 0),
		COALESCE(mru, 0),
		COALESCE(node_id, '')
	FROM
		node_resources_total;`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var nodeResourcesTotal NodeResourcesTotal
		if err := rows.Scan(
			&nodeResourcesTotal.ID,
			&nodeResourcesTotal.HRU,
			&nodeResourcesTotal.SRU,
			&nodeResourcesTotal.CRU,
			&nodeResourcesTotal.MRU,
			&nodeResourcesTotal.NodeID,
		); err != nil {
			return err
		}
		data.NodeTotalResources[data.NodeIDMap[nodeResourcesTotal.NodeID]] = nodeResourcesTotal
	}
	return nil
}

func loadFarms(db *sql.DB, data *DBData) error {
	rows, err := db.Query(`
	SELECT 
		COALESCE(id, ''),
		COALESCE(grid_version, 0),
		COALESCE(farm_id, 0),
		COALESCE(name, ''),
		COALESCE(twin_id, 0),
		COALESCE(pricing_policy_id, 0),
		COALESCE(certification, ''),
		COALESCE(stellar_address, ''),
		COALESCE(dedicated_farm, false)
	FROM
		farm;`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var farm Farm
		if err := rows.Scan(
			&farm.ID,
			&farm.GridVersion,
			&farm.FarmID,
			&farm.Name,
			&farm.TwinID,
			&farm.PricingPolicyID,
			&farm.Certification,
			&farm.StellarAddress,
			&farm.DedicatedFarm,
		); err != nil {
			return err
		}
		data.Farms[farm.FarmID] = farm
		data.FarmIDMap[farm.ID] = farm.FarmID
		data.FarmHasRentedNode[farm.FarmID] = map[uint64]bool{}
	}
	return nil
}

func loadTwins(db *sql.DB, data *DBData) error {
	rows, err := db.Query(`
	SELECT
	COALESCE(id, ''),
	COALESCE(grid_version, 0),
	COALESCE(twin_id, 0),
	COALESCE(account_id, ''),
	COALESCE(relay, ''),
	COALESCE(public_key, '')
	FROM
		twin;`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var twin Twin
		if err := rows.Scan(
			&twin.ID,
			&twin.GridVersion,
			&twin.TwinID,
			&twin.AccountID,
			&twin.Relay,
			&twin.PublicKey,
		); err != nil {
			return err
		}
		data.Twins[twin.TwinID] = twin
	}
	return nil
}

func loadPublicIPs(db *sql.DB, data *DBData) error {
	rows, err := db.Query(`
	SELECT 
		COALESCE(id, ''),
		COALESCE(gateway, ''),
		COALESCE(ip, ''),
		COALESCE(contract_id, 0),
		COALESCE(farm_id, '')
	FROM
		public_ip;`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var publicIP PublicIp
		if err := rows.Scan(
			&publicIP.ID,
			&publicIP.Gateway,
			&publicIP.IP,
			&publicIP.ContractID,
			&publicIP.FarmID,
		); err != nil {
			return err
		}
		data.PublicIPs[publicIP.ID] = publicIP
	}
	return nil
}

func loadPublicConfigs(db *sql.DB, data *DBData) error {
	rows, err := db.Query(`
	SELECT
	COALESCE(id, ''),
	COALESCE(ipv4, ''),
	COALESCE(ipv6, ''),
	COALESCE(gw4, ''),
	COALESCE(gw6, ''),
	COALESCE(domain, ''),
	COALESCE(node_id, '')
	FROM
		public_config;`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var publicConfig PublicConfig
		if err := rows.Scan(
			&publicConfig.ID,
			&publicConfig.IPv4,
			&publicConfig.IPv6,
			&publicConfig.GW4,
			&publicConfig.GW6,
			&publicConfig.Domain,
			&publicConfig.NodeID,
		); err != nil {
			return err
		}
		data.PublicConfigs[data.NodeIDMap[publicConfig.NodeID]] = publicConfig
	}
	return nil
}
func loadContracts(db *sql.DB, data *DBData) error {
	rows, err := db.Query(`
	SELECT
		COALESCE(id, ''),
		COALESCE(grid_version, 0),
		COALESCE(contract_id, 0),
		COALESCE(twin_id, 0),
		COALESCE(node_id, 0),
		COALESCE(deployment_data, ''),
		COALESCE(deployment_hash, ''),
		COALESCE(number_of_public_i_ps, 0),
		COALESCE(state, ''),
		COALESCE(created_at, 0),
		COALESCE(resources_used_id, '')
	FROM
		node_contract;`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var contract NodeContract
		if err := rows.Scan(
			&contract.ID,
			&contract.GridVersion,
			&contract.ContractID,
			&contract.TwinID,
			&contract.NodeID,
			&contract.DeploymentData,
			&contract.DeploymentHash,
			&contract.NumberOfPublicIPs,
			&contract.State,
			&contract.CreatedAt,
			&contract.ResourcesUsedID,
		); err != nil {
			return err
		}
		data.NodeContracts[contract.ContractID] = contract
		if contract.State != deleted {
			data.NonDeletedContracts[contract.NodeID] = append(data.NonDeletedContracts[contract.NodeID], contract.ContractID)
		}

	}
	return nil
}
func loadRentContracts(db *sql.DB, data *DBData) error {
	rows, err := db.Query(`
	SELECT
		COALESCE(id, ''),
		COALESCE(grid_version, 0),
		COALESCE(contract_id, 0),
		COALESCE(twin_id, 0),
		COALESCE(node_id, 0),
		COALESCE(state, ''),
		COALESCE(created_at, 0)
	FROM
		rent_contract;`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var contract RentContract
		if err := rows.Scan(

			&contract.ID,
			&contract.GridVersion,
			&contract.ContractID,
			&contract.TwinID,
			&contract.NodeID,
			&contract.State,
			&contract.CreatedAt,
		); err != nil {
			return err
		}
		data.RentContracts[contract.ContractID] = contract
	}
	return nil
}
func loadNameContracts(db *sql.DB, data *DBData) error {
	rows, err := db.Query(`
	SELECT
		COALESCE(id, ''),
		COALESCE(grid_version, 0),
		COALESCE(contract_id, 0),
		COALESCE(twin_id, 0),
		COALESCE(name, ''),
		COALESCE(state, ''),
		COALESCE(created_at, 0)
	FROM
		name_contract;`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var contract NameContract
		if err := rows.Scan(
			&contract.ID,
			&contract.GridVersion,
			&contract.ContractID,
			&contract.TwinID,
			&contract.Name,
			&contract.State,
			&contract.CreatedAt,
		); err != nil {
			return err
		}
		data.NameContracts[contract.ContractID] = contract
	}
	return nil
}

func loadContractResources(db *sql.DB, data *DBData) error {
	rows, err := db.Query(`
	SELECT
	COALESCE(id, ''),
	COALESCE(hru, 0),
	COALESCE(sru, 0),
	COALESCE(cru, 0),
	COALESCE(mru, 0),
	COALESCE(contract_id, '')
	FROM
		contract_resources;`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var contractResources ContractResources
		if err := rows.Scan(
			&contractResources.ID,
			&contractResources.HRU,
			&contractResources.SRU,
			&contractResources.CRU,
			&contractResources.MRU,
			&contractResources.ContractID,
		); err != nil {
			return err
		}
		data.ContractResources[contractResources.ID] = contractResources
	}
	return nil
}
func loadContractBillingReports(db *sql.DB, data *DBData) error {
	rows, err := db.Query(`
	SELECT
		COALESCE(id, ''),
		COALESCE(contract_id, 0),
		COALESCE(discount_received, ''),
		COALESCE(amount_billed, 0),
		COALESCE(timestamp, 0)
	FROM
		contract_bill_report
	ORDER BY
        timestamp DESC;`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var contractBillReport ContractBillReport
		if err := rows.Scan(
			&contractBillReport.ID,
			&contractBillReport.ContractID,
			&contractBillReport.DiscountReceived,
			&contractBillReport.AmountBilled,
			&contractBillReport.Timestamp,
		); err != nil {
			return err
		}
		data.Billings[contractBillReport.ContractID] = append(data.Billings[contractBillReport.ContractID], contractBillReport)
		data.BillReports++
	}
	return nil
}

func loadCountries(db *sql.DB, data *DBData) error {
	rows, err := db.Query(`
	SELECT
		COALESCE(id, ''),
		COALESCE(country_id, 0),
		COALESCE(code, ''),
		COALESCE(name, ''),
		COALESCE(region, ''),
		COALESCE(subregion, ''),
		COALESCE(lat, ''),
		COALESCE(long, '')
	FROM
		country;
	`)
	if err != nil {
		return err
	}

	for rows.Next() {
		var country Country
		if err := rows.Scan(
			&country.ID,
			&country.CountryID,
			&country.Code,
			&country.Name,
			&country.Region,
			&country.Subregion,
			&country.Lat,
			&country.Long,
		); err != nil {
			return err
		}
		data.Regions[strings.ToLower(country.Name)] = country.Region
	}

	return nil
}

func loadLocations(db *sql.DB, data *DBData) error {
	rows, err := db.Query(`
	SELECT 
		COALESCE(id, ''),
		CASE WHEN longitude = '' THEN NULL ELSE longitude END,
		CASE WHEN latitude = '' THEN NULL ELSE latitude END
	FROM
		location;
	`)

	if err != nil {
		return err
	}

	for rows.Next() {
		var location Location
		if err := rows.Scan(
			&location.ID,
			&location.Longitude,
			&location.Latitude,
		); err != nil {
			return err
		}

		data.Locations[location.ID] = location
	}

	return nil
}

func loadNodeGPUs(db *sql.DB, data *DBData) error {
	rows, err := db.Query(`
	SELECT 
		COALESCE(id, ''),
		COALESCE(contract, 0),
		COALESCE(node_twin_id, 0),
		COALESCE(vendor, ''),
		COALESCE(device, '')
	FROM
		node_gpu;`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var gpu types.NodeGPU
		if err := rows.Scan(
			&gpu.ID,
			&gpu.Contract,
			&gpu.NodeTwinID,
			&gpu.Vendor,
			&gpu.Device,
		); err != nil {
			return err
		}
		data.GPUs[gpu.NodeTwinID] = append(data.GPUs[gpu.NodeTwinID], gpu)
	}
	return nil
}

func loadHealthReports(db *sql.DB, data *DBData) error {
	rows, err := db.Query(`
	SELECT
		COALESCE(node_twin_id, 0),
		COALESCE(healthy, false)
	FROM
		health_report;`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var health types.HealthReport
		if err := rows.Scan(
			&health.NodeTwinId,
			&health.Healthy,
		); err != nil {
			return err
		}
		data.HealthReports[health.NodeTwinId] = health.Healthy
	}

	return nil
}

func loadDMIs(db *sql.DB, gormDB *gorm.DB, data *DBData) error {
	var dmis []types.Dmi
	err := gormDB.Table("dmi").Scan(&dmis).Error
	if err != nil {
		return err
	}
	for _, dmi := range dmis {
		twinId := dmi.NodeTwinId
		dmi.NodeTwinId = 0 // to omit it as empty, cleaner response
		data.DMIs[twinId] = dmi
	}

	return nil
}

func loadSpeeds(db *sql.DB, data *DBData) error {
	rows, err := db.Query(`
	SELECT 
		node_twin_id,
		upload,
		download
	FROM 
		speed;`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var speed types.Speed
		if err := rows.Scan(
			&speed.NodeTwinId,
			&speed.Upload,
			&speed.Download,
		); err != nil {
			return err
		}
		data.Speeds[speed.NodeTwinId] = speed
	}
	return nil
}

func loadPricingPolicies(db *sql.DB, data *DBData) error {
	rows, err := db.Query(`
		SELECT
			id,
			grid_version,
			pricing_policy_id,
			name,
			su,
			cu,
			nu,
			ipu,
			foundation_account,
			certified_sales_account,
			dedicated_node_discount
		FROM
			pricing_policy;`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var policy PricingPolicy
		var su, cu, nu, ipu string
		if err := rows.Scan(
			&policy.ID,
			&policy.GridVersion,
			&policy.PricingPolicyID,
			&policy.Name,
			&su,
			&cu,
			&nu,
			&ipu,
			&policy.FoundationAccount,
			&policy.CertifiedSalesAccount,
			&policy.DedicatedNodeDiscount,
		); err != nil {
			return err
		}

		policy.SU = parseUnit(su)
		policy.CU = parseUnit(cu)
		policy.NU = parseUnit(nu)
		policy.IPU = parseUnit(ipu)

		data.PricingPolicies[policy.ID] = policy
	}

	return nil
}

func parseUnit(unitString string) Unit {
	var unit Unit
	_ = json.Unmarshal([]byte(unitString), &unit)
	return unit
}

func Load(db *sql.DB, gormDB *gorm.DB) (DBData, error) {
	data := DBData{
		NodeIDMap:           make(map[string]uint64),
		FarmIDMap:           make(map[string]uint64),
		FreeIPs:             make(map[uint64]uint64),
		TotalIPs:            make(map[uint64]uint64),
		Nodes:               make(map[uint64]Node),
		Farms:               make(map[uint64]Farm),
		Twins:               make(map[uint64]Twin),
		PublicIPs:           make(map[string]PublicIp),
		PublicConfigs:       make(map[uint64]PublicConfig),
		NodeContracts:       make(map[uint64]NodeContract),
		RentContracts:       make(map[uint64]RentContract),
		NameContracts:       make(map[uint64]NameContract),
		NodeRentedBy:        make(map[uint64]uint64),
		NodeRentContractID:  make(map[uint64]uint64),
		Billings:            make(map[uint64][]ContractBillReport),
		BillReports:         0,
		ContractResources:   make(map[string]ContractResources),
		NodeTotalResources:  make(map[uint64]NodeResourcesTotal),
		NodeUsedResources:   make(map[uint64]NodeResourcesTotal),
		NonDeletedContracts: make(map[uint64][]uint64),
		GPUs:                make(map[uint32][]types.NodeGPU),
		FarmHasRentedNode:   make(map[uint64]map[uint64]bool),
		Regions:             make(map[string]string),
		Locations:           make(map[string]Location),
		HealthReports:       make(map[uint32]bool),
		DMIs:                make(map[uint32]types.Dmi),
		Speeds:              make(map[uint32]types.Speed),
		PricingPolicies:     make(map[uint]PricingPolicy),
		DB:                  db,
	}
	if err := loadNodes(db, gormDB, &data); err != nil {
		return data, err
	}
	if err := loadFarms(db, &data); err != nil {
		return data, err
	}
	if err := loadTwins(db, &data); err != nil {
		return data, err
	}
	if err := loadPublicConfigs(db, &data); err != nil {
		return data, err
	}
	if err := loadPublicIPs(db, &data); err != nil {
		return data, err
	}
	if err := loadContracts(db, &data); err != nil {
		return data, err
	}
	if err := loadRentContracts(db, &data); err != nil {
		return data, err
	}
	if err := loadNameContracts(db, &data); err != nil {
		return data, err
	}
	if err := loadContractResources(db, &data); err != nil {
		return data, err
	}
	if err := loadContractBillingReports(db, &data); err != nil {
		return data, err
	}
	if err := loadNodesTotalResources(db, &data); err != nil {
		return data, err
	}
	if err := loadNodeGPUs(db, &data); err != nil {
		return data, err
	}
	if err := loadCountries(db, &data); err != nil {
		return data, err
	}
	if err := loadLocations(db, &data); err != nil {
		return data, err
	}
	if err := loadHealthReports(db, &data); err != nil {
		return data, err
	}
	if err := loadDMIs(db, gormDB, &data); err != nil {
		return data, err
	}
	if err := loadSpeeds(db, &data); err != nil {
		return data, err
	}
	if err := loadPricingPolicies(db, &data); err != nil {
		return data, err
	}
	if err := calcNodesUsedResources(&data); err != nil {
		return data, err
	}
	if err := calcRentInfo(&data); err != nil {
		return data, err
	}
	if err := calcFreeIPs(&data); err != nil {
		return data, err
	}
	return data, nil
}
