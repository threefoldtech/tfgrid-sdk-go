package mock

import (
	"database/sql"
)

type DBData struct {
	NodeIDMap          map[string]uint64
	FarmIDMap          map[string]uint64
	FreeIPs            map[uint64]uint64
	TotalIPs           map[uint64]uint64
	NodeRentedBy       map[uint64]uint64
	NodeRentContractID map[uint64]uint64
	FarmHasRentedNode  map[uint64]bool

	Nodes               map[uint64]Node
	NodesCacheMap       map[uint64]NodesCache
	NodeTotalResources  map[uint64]NodeResourcesTotal
	Farms               map[uint64]Farm
	FarmsCacheMap       map[uint64]FarmsCache
	Twins               map[uint64]Twin
	PublicIPs           map[string]PublicIp
	PublicConfigs       map[uint64]PublicConfig
	GenericContracts    map[uint64]GenericContract
	NodeContracts       map[uint64]GenericContract
	RentContracts       map[uint64]GenericContract
	NameContracts       map[uint64]GenericContract
	Billings            map[uint64][]ContractBillReport
	ContractResources   map[string]ContractResources
	NonDeletedContracts map[uint64][]uint64
	GPUs                map[uint64]NodeGPU
	DB                  *sql.DB
}

func loadNodes(db *sql.DB, data *DBData) error {
	rows, err := db.Query(`
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
		COALESCE(extra_fee, 0),
		power,
		total_hru,
		total_sru,
		total_cru,
		total_mru
	FROM
		node;`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var node Node
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
			&node.CreatedAt,
			&node.UpdatedAt,
			&node.LocationID,
			&node.ExtraFee,
			&node.Power,
			&node.TotalHRU,
			&node.TotalSRU,
			&node.TotalCRU,
			&node.TotalMRU,
		); err != nil {
			return err
		}
		data.Nodes[node.NodeID] = node
		data.NodeIDMap[node.ID] = node.NodeID
	}
	return nil
}

func loadNodesCache(db *sql.DB, data *DBData) error {
	rows, err := db.Query(`
	SELECT
		node_id,
		farm_id,
		node_twin_id,
		free_hru,
		free_sru,
		free_mru,
		free_cru,
		COALESCE(renter, 0),
		COALESCE(rent_contract_id, 0),
		node_contracts,
		free_gpus
	FROM
		nodes_cache;`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var cache NodesCache
		if err := rows.Scan(
			&cache.NodeID,
			&cache.FarmID,
			&cache.NodeTwinID,
			&cache.FreeHRU,
			&cache.FreeSRU,
			&cache.FreeMRU,
			&cache.FreeCRU,
			&cache.Renter,
			&cache.RentContractID,
			&cache.NodeContracts,
			&cache.FreeGPUs,
		); err != nil {
			return err
		}
		data.NodesCacheMap[cache.NodeID] = cache
	}
	return nil
}

func loadFarmsCache(db *sql.DB, data *DBData) error {
	rows, err := db.Query(`
	SELECT
		farm_id,
		free_ips,
		total_ips,
		COALESCE(ips, '[]')
	FROM
		farms_cache;`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var cache FarmsCache
		if err := rows.Scan(
			&cache.FarmID,
			&cache.FreeIPs,
			&cache.TotalIPs,
			&cache.IPs,
		); err != nil {
			return err
		}
		data.FarmsCacheMap[cache.FarmID] = cache
	}
	return nil
}

func calcRentInfo(data *DBData) error {
	for _, contract := range data.RentContracts {
		if contract.State == "Deleted" {
			continue
		}
		data.NodeRentedBy[contract.NodeID] = contract.TwinID
		data.NodeRentContractID[contract.NodeID] = contract.ContractID
		farmID := data.Nodes[contract.NodeID].FarmID
		data.FarmHasRentedNode[farmID] = true
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
func loadGenericContracts(db *sql.DB, data *DBData) error {
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
		COALESCE(resources_used_id, ''),
		COALESCE(name, ''),
		type
	FROM
		generic_contract;`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var contract GenericContract
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
			&contract.Name,
			&contract.Type,
		); err != nil {
			return err
		}

		switch contract.Type {
		case "node":
			data.NodeContracts[contract.ContractID] = contract
		case "name":
			data.NameContracts[contract.ContractID] = contract
		case "rent":
			data.RentContracts[contract.ContractID] = contract
		}

		if contract.State != "Deleted" {
			data.NonDeletedContracts[contract.NodeID] = append(data.NonDeletedContracts[contract.NodeID], contract.ContractID)
		}

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
		var gpu NodeGPU
		if err := rows.Scan(
			&gpu.ID,
			&gpu.Contract,
			&gpu.NodeTwinID,
			&gpu.Vendor,
			&gpu.Device,
		); err != nil {
			return err
		}
		data.GPUs[gpu.NodeTwinID] = gpu
	}
	return nil
}

func Load(db *sql.DB) (DBData, error) {
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
		NodeContracts:       make(map[uint64]GenericContract),
		RentContracts:       make(map[uint64]GenericContract),
		NameContracts:       make(map[uint64]GenericContract),
		NodeRentedBy:        make(map[uint64]uint64),
		NodeRentContractID:  make(map[uint64]uint64),
		Billings:            make(map[uint64][]ContractBillReport),
		ContractResources:   make(map[string]ContractResources),
		NodeTotalResources:  make(map[uint64]NodeResourcesTotal),
		NonDeletedContracts: make(map[uint64][]uint64),
		GPUs:                make(map[uint64]NodeGPU),
		FarmHasRentedNode:   make(map[uint64]bool),
		NodesCacheMap:       make(map[uint64]NodesCache),
		FarmsCacheMap:       make(map[uint64]FarmsCache),
		GenericContracts:    make(map[uint64]GenericContract),
		DB:                  db,
	}
	if err := loadNodes(db, &data); err != nil {
		return data, err
	}
	if err := loadNodesCache(db, &data); err != nil {
		return data, err
	}
	if err := loadFarms(db, &data); err != nil {
		return data, err
	}
	if err := loadFarmsCache(db, &data); err != nil {
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
	if err := loadGenericContracts(db, &data); err != nil {
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
	// if err := calcNodesUsedResources(&data); err != nil {
	// 	return data, err
	// }
	if err := calcRentInfo(&data); err != nil {
		return data, err
	}
	if err := calcFreeIPs(&data); err != nil {
		return data, err
	}
	return data, nil
}
