package mock

import (
	"database/sql"
	"math"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

type DBData struct {
	NodeIDMap          map[string]uint32
	FarmIDMap          map[string]uint32
	FreeIPs            map[uint32]uint64
	TotalIPs           map[uint32]uint64
	NodeUsedResources  map[uint32]DBNodeResourcesTotal
	NodeRentedBy       map[uint32]uint32
	NodeRentContractID map[uint32]uint64

	Nodes               map[uint32]DBNode
	NodeTotalResources  map[uint32]DBNodeResourcesTotal
	Farms               map[uint32]DBFarm
	Twins               map[uint32]DBTwin
	PublicIPs           map[string]DBPublicIP
	PublicConfigs       map[uint32]DBPublicConfig
	NodeContracts       map[uint64]DBNodeContract
	RentContracts       map[uint64]DBRentContract
	NameContracts       map[uint64]DBNameContract
	Billings            map[uint64][]DBContractBillReport
	ContractResources   map[string]DBContractResources
	NonDeletedContracts map[uint32][]uint64
	DB                  *sql.DB
}

func loadNodes(database *sql.DB, dbData *DBData) error {
	rows, err := database.Query(`
	SELECT
		COALESCE(id, ''),
		COALESCE(grid_version, 0),
		COALESCE(node_id, 0),
		COALESCE(farm_id, 0),
		COALESCE(twin_id, 0),
		COALESCE(country, ''),
		COALESCE(city, ''),
		COALESCE(uptime, 0),
		COALESCE(created, 0),
		COALESCE(farming_policy_id, 0),
		COALESCE(certification, ''),
		COALESCE(secure, false),
		COALESCE(virtualized, false),
		COALESCE(serial_number, ''),
		COALESCE(created_at, 0),
		COALESCE(updated_at, 0),
		COALESCE(location_id, ''),
		COALESCE(has_gpu, false),
		COALESCE(extra_fee, 0),
		power
	FROM
		node;`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var node DBNode
		if err := rows.Scan(
			&node.ID,
			&node.GridVersion,
			&node.NodeID,
			&node.FarmID,
			&node.TwinID,
			&node.Country,
			&node.City,
			&node.Uptime,
			&node.Created,
			&node.FarmingPolicyID,
			&node.Certification,
			&node.Secure,
			&node.Virtualized,
			&node.SerialNumber,
			&node.CreateAt,
			&node.UpdatedAt,
			&node.LocationID,
			&node.HasGPU,
			&node.ExtraFee,
			&node.Power,
		); err != nil {
			return err
		}
		dbData.Nodes[node.NodeID] = node
		dbData.NodeIDMap[node.ID] = node.NodeID
	}
	return nil
}

func calcNodesUsedResources(dbData *DBData) error {

	for _, node := range dbData.Nodes {
		used := DBNodeResourcesTotal{
			MRU: uint64(2 * gridtypes.Gigabyte),
			SRU: uint64(100 * gridtypes.Gigabyte),
		}
		tenpercent := uint64(math.Round(float64(dbData.NodeTotalResources[node.NodeID].MRU) / 10))
		if used.MRU < tenpercent {
			used.MRU = tenpercent
		}
		dbData.NodeUsedResources[node.NodeID] = used
	}

	for _, contract := range dbData.NodeContracts {
		if contract.State == "Deleted" {
			continue
		}
		contratResourceID := contract.ResourcesUsedID
		dbData.NodeUsedResources[contract.NodeID] = DBNodeResourcesTotal{
			CRU: dbData.ContractResources[contratResourceID].CRU + dbData.NodeUsedResources[contract.NodeID].CRU,
			MRU: dbData.ContractResources[contratResourceID].MRU + dbData.NodeUsedResources[contract.NodeID].MRU,
			HRU: dbData.ContractResources[contratResourceID].HRU + dbData.NodeUsedResources[contract.NodeID].HRU,
			SRU: dbData.ContractResources[contratResourceID].SRU + dbData.NodeUsedResources[contract.NodeID].SRU,
		}

	}
	return nil
}

func calcRentInfo(dbData *DBData) error {
	for _, contract := range dbData.RentContracts {
		if contract.State == "Deleted" {
			continue
		}
		dbData.NodeRentedBy[contract.NodeID] = contract.TwinID
		dbData.NodeRentContractID[contract.NodeID] = contract.ContractID
	}
	return nil
}

func calcFreeIPs(dbData *DBData) error {
	for _, publicIP := range dbData.PublicIPs {
		if publicIP.ContractID == 0 {
			dbData.FreeIPs[dbData.FarmIDMap[publicIP.FarmID]]++
		}
		dbData.TotalIPs[dbData.FarmIDMap[publicIP.FarmID]]++
	}
	return nil
}

func loadNodesTotalResources(database *sql.DB, dbData *DBData) error {
	rows, err := database.Query(`
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
		var nodeResourcesTotal DBNodeResourcesTotal
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
		dbData.NodeTotalResources[dbData.NodeIDMap[nodeResourcesTotal.NodeID]] = nodeResourcesTotal
	}
	return nil
}

func loadFarms(database *sql.DB, dbData *DBData) error {
	rows, err := database.Query(`
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
		var farm DBFarm
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
		dbData.Farms[farm.FarmID] = farm
		dbData.FarmIDMap[farm.ID] = farm.FarmID
	}
	return nil
}

func loadTwins(database *sql.DB, dbData *DBData) error {
	rows, err := database.Query(`
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
		var twin DBTwin
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
		dbData.Twins[twin.TwinID] = twin
	}
	return nil
}

func loadPublicIPs(database *sql.DB, dbData *DBData) error {
	rows, err := database.Query(`
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
		var publicIP DBPublicIP
		if err := rows.Scan(
			&publicIP.ID,
			&publicIP.Gateway,
			&publicIP.IP,
			&publicIP.ContractID,
			&publicIP.FarmID,
		); err != nil {
			return err
		}
		dbData.PublicIPs[publicIP.ID] = publicIP
	}
	return nil
}

func loadPublicConfigs(database *sql.DB, dbData *DBData) error {
	rows, err := database.Query(`
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
		var publicConfig DBPublicConfig
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
		dbData.PublicConfigs[dbData.NodeIDMap[publicConfig.NodeID]] = publicConfig
	}
	return nil
}
func loadContracts(database *sql.DB, dbData *DBData) error {
	rows, err := database.Query(`
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
		var contract DBNodeContract
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
		dbData.NodeContracts[contract.ContractID] = contract
		if contract.State != "Deleted" {
			dbData.NonDeletedContracts[contract.NodeID] = append(dbData.NonDeletedContracts[contract.NodeID], contract.ContractID)
		}

	}
	return nil
}
func loadRentContracts(database *sql.DB, dbData *DBData) error {
	rows, err := database.Query(`
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
		var contract DBRentContract
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
		dbData.RentContracts[contract.ContractID] = contract
	}
	return nil
}
func loadNameContracts(database *sql.DB, dbData *DBData) error {
	rows, err := database.Query(`
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
		var contract DBNameContract
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
		dbData.NameContracts[contract.ContractID] = contract
	}
	return nil
}

func loadContractResources(database *sql.DB, dbData *DBData) error {
	rows, err := database.Query(`
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
		var contractResources DBContractResources
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
		dbData.ContractResources[contractResources.ID] = contractResources
	}
	return nil
}
func loadContractBillingReports(database *sql.DB, dbData *DBData) error {
	rows, err := database.Query(`
	SELECT
		COALESCE(id, ''),
		COALESCE(contract_id, 0),
		COALESCE(discount_received, ''),
		COALESCE(amount_billed, 0),
		COALESCE(timestamp, 0)
	FROM
		contract_bill_report;`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var contractBillReport DBContractBillReport
		if err := rows.Scan(
			&contractBillReport.ID,
			&contractBillReport.ContractID,
			&contractBillReport.DiscountReceived,
			&contractBillReport.AmountBilled,
			&contractBillReport.Timestamp,
		); err != nil {
			return err
		}
		dbData.Billings[contractBillReport.ContractID] = append(dbData.Billings[contractBillReport.ContractID], contractBillReport)
	}
	return nil
}

func NewDBData(database *sql.DB) (DBData, error) {
	data := DBData{
		NodeIDMap:           make(map[string]uint32),
		FarmIDMap:           make(map[string]uint32),
		FreeIPs:             make(map[uint32]uint64),
		TotalIPs:            make(map[uint32]uint64),
		Nodes:               make(map[uint32]DBNode),
		Farms:               make(map[uint32]DBFarm),
		Twins:               make(map[uint32]DBTwin),
		PublicIPs:           make(map[string]DBPublicIP),
		PublicConfigs:       make(map[uint32]DBPublicConfig),
		NodeContracts:       make(map[uint64]DBNodeContract),
		RentContracts:       make(map[uint64]DBRentContract),
		NameContracts:       make(map[uint64]DBNameContract),
		NodeRentedBy:        make(map[uint32]uint32),
		NodeRentContractID:  make(map[uint32]uint64),
		Billings:            make(map[uint64][]DBContractBillReport),
		ContractResources:   make(map[string]DBContractResources),
		NodeTotalResources:  make(map[uint32]DBNodeResourcesTotal),
		NodeUsedResources:   make(map[uint32]DBNodeResourcesTotal),
		NonDeletedContracts: make(map[uint32][]uint64),
		DB:                  database,
	}
	if err := loadNodes(database, &data); err != nil {
		return data, err
	}
	if err := loadFarms(database, &data); err != nil {
		return data, err
	}
	if err := loadTwins(database, &data); err != nil {
		return data, err
	}
	if err := loadPublicConfigs(database, &data); err != nil {
		return data, err
	}
	if err := loadPublicIPs(database, &data); err != nil {
		return data, err
	}
	if err := loadContracts(database, &data); err != nil {
		return data, err
	}
	if err := loadRentContracts(database, &data); err != nil {
		return data, err
	}
	if err := loadNameContracts(database, &data); err != nil {
		return data, err
	}
	if err := loadContractResources(database, &data); err != nil {
		return data, err
	}
	if err := loadContractBillingReports(database, &data); err != nil {
		return data, err
	}
	if err := loadNodesTotalResources(database, &data); err != nil {
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
