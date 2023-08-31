package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

var (
	nodesMRU               = make(map[uint64]uint64)
	nodesSRU               = make(map[uint64]uint64)
	nodesHRU               = make(map[uint64]uint64)
	nodeUP                 = make(map[uint64]bool)
	createdNodeContracts   = make([]uint64, 0)
	dedicatedFarms         = make(map[uint64]struct{})
	availableRentNodes     = make(map[uint64]struct{})
	availableRentNodesList = make([]uint64, 0)
	renter                 = make(map[uint64]uint64)
	billCnt                = 1
	contractCnt            = uint64(1)
)

const (
	contractCreatedRatio = .1 // from devnet
	usedPublicIPsRatio   = .9
	nodeUpRatio          = .5
	nodeCount            = 1000
	farmCount            = 100
	normalUsers          = 2000
	publicIPCount        = 1000
	twinCount            = nodeCount + farmCount + normalUsers
	contractCount        = 3000
	rentContractCount    = 100
	nameContractCount    = 300

	maxContractHRU = 1024 * 1024 * 1024 * 300
	maxContractSRU = 1024 * 1024 * 1024 * 300
	maxContractMRU = 1024 * 1024 * 1024 * 16
	maxContractCRU = 16
	minContractHRU = 0
	minContractSRU = 1024 * 1024 * 256
	minContractMRU = 1024 * 1024 * 256
	minContractCRU = 1
)

var (
	countries = []string{"Belgium", "United States", "Egypt", "United Kingdom"}
	cities    = map[string][]string{
		"Belgium":        {"Brussels", "Antwerp", "Ghent", "Charleroi"},
		"United States":  {"New York", "Chicago", "Los Angeles", "San Francisco"},
		"Egypt":          {"Cairo", "Giza", "October", "Nasr City"},
		"United Kingdom": {"London", "Liverpool", "Manchester", "Cambridge"},
	}
)

func initSchema(db *sql.DB) error {
	schema, err := os.ReadFile("./schema.sql")
	if err != nil {
		return err
	}
	_, err = db.Exec(string(schema))
	if err != nil {
		return err
	}
	return nil
}

func generateTwins(db *sql.DB) error {
	var values []string

	for i := uint64(1); i <= twinCount; i++ {
		twin := twin{
			id:           fmt.Sprintf("twin-%d", i),
			account_id:   fmt.Sprintf("account-id-%d", i),
			relay:        fmt.Sprintf("relay-%d", i),
			public_key:   fmt.Sprintf("public-key-%d", i),
			twin_id:      i,
			grid_version: 3,
		}

		value, err := extractValuesFromInsertQuery(insertQuery(&twin))
		if err != nil {
			return err
		}
		values = append(values, value)
	}

	if len(values) != 0 {
		query := "INSERT INTO twin (id, grid_version, twin_id, account_id, relay, public_key) VALUES"
		query += strings.Join(values, ",") + ";"

		if _, err := db.Exec(query); err != nil {
			return err
		}

	}

	fmt.Println("twins generated")
	return nil
}

func generatePublicIPs(db *sql.DB) error {
	var publicIPValues []string
	var nodeContractUpdateValues []string

	for i := uint64(1); i <= publicIPCount; i++ {
		contract_id := uint64(0)
		if flip(usedPublicIPsRatio) {
			contract_id = createdNodeContracts[rnd(0, uint64(len(createdNodeContracts))-1)]
		}
		ip := randomIPv4()
		public_ip := public_ip{
			id:          fmt.Sprintf("public-ip-%d", i),
			gateway:     ip.String(),
			ip:          IPv4Subnet(ip).String(),
			contract_id: contract_id,
			farm_id:     fmt.Sprintf("farm-%d", rnd(1, farmCount)),
		}

		value, err := extractValuesFromInsertQuery(insertQuery(&public_ip))
		if err != nil {
			return err
		}
		publicIPValues = append(publicIPValues, value)
		nodeContractUpdateValues = append(nodeContractUpdateValues, fmt.Sprintf("%d", contract_id))

	}

	if len(publicIPValues) != 0 {
		query := "INSERT INTO public_ip (id, gateway, ip, contract_id, farm_id) VALUES"
		query += strings.Join(publicIPValues, ",") + ";"

		if _, err := db.Exec(query); err != nil {
			return err
		}
	}

	if len(nodeContractUpdateValues) != 0 {
		query := "UPDATE node_contract set number_of_public_i_ps = number_of_public_i_ps + 1 WHERE contract_id IN ("
		query += strings.Join(nodeContractUpdateValues, ",") + ");"
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	fmt.Println("public IPs generated")

	return nil
}

func generateFarms(db *sql.DB) error {
	var values []string
	for i := uint64(1); i <= farmCount; i++ {
		farm := farm{
			id:                fmt.Sprintf("farm-%d", i),
			farm_id:           i,
			name:              fmt.Sprintf("farm-name-%d", i),
			certification:     "Diy",
			dedicated_farm:    flip(.1),
			twin_id:           i,
			pricing_policy_id: 1,
			grid_version:      3,
			stellar_address:   "",
		}
		if farm.dedicated_farm {
			dedicatedFarms[farm.farm_id] = struct{}{}
		}

		value, err := extractValuesFromInsertQuery(insertQuery(&farm))
		if err != nil {
			return err
		}
		values = append(values, value)
	}

	if len(values) != 0 {
		query := "INSERT INTO farm (id, grid_version, farm_id, name, twin_id, pricing_policy_id, certification, stellar_address, dedicated_farm) VALUES"
		query += strings.Join(values, ",") + ";"
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	fmt.Println("Farms generated")

	return nil
}

func generateContracts(db *sql.DB) error {
	var contractsValues []string
	var contractResourcesValues []string
	var billingValues []string
	startContractCnt := contractCnt

	for i := uint64(1); i <= contractCount; i++ {
		nodeID := rnd(1, nodeCount)
		state := "Deleted"
		if nodeUP[nodeID] {
			if flip(contractCreatedRatio) {
				state = "Created"
			} else if flip(0.5) {
				state = "GracePeriod"
			}
		}
		if state != "Deleted" && (minContractHRU > nodesHRU[nodeID] || minContractMRU > nodesMRU[nodeID] || minContractSRU > nodesSRU[nodeID]) {
			i--
			continue
		}
		twinID := rnd(1100, 3100)
		if renter, ok := renter[nodeID]; ok {
			twinID = renter
		}
		if _, ok := availableRentNodes[nodeID]; ok {
			i--
			continue
		}
		contract := node_contract{
			id:                    fmt.Sprintf("node-contract-%d", contractCnt),
			twin_id:               twinID,
			contract_id:           contractCnt,
			state:                 state,
			created_at:            uint64(time.Now().Unix()),
			node_id:               nodeID,
			deployment_data:       fmt.Sprintf("deployment-data-%d", contractCnt),
			deployment_hash:       fmt.Sprintf("deployment-hash-%d", contractCnt),
			number_of_public_i_ps: 0,
			grid_version:          3,
			resources_used_id:     "",
		}
		cru := rnd(minContractCRU, maxContractCRU)
		hru := rnd(minContractHRU, min(maxContractHRU, nodesHRU[nodeID]))
		sru := rnd(minContractSRU, min(maxContractSRU, nodesSRU[nodeID]))
		mru := rnd(minContractMRU, min(maxContractMRU, nodesMRU[nodeID]))
		contract_resources := contract_resources{
			id:          fmt.Sprintf("contract-resources-%d", contractCnt),
			hru:         hru,
			sru:         sru,
			cru:         cru,
			mru:         mru,
			contract_id: fmt.Sprintf("node-contract-%d", contractCnt),
		}
		if contract.state != "Deleted" {
			nodesHRU[nodeID] -= hru
			nodesSRU[nodeID] -= sru
			nodesMRU[nodeID] -= mru
			createdNodeContracts = append(createdNodeContracts, contractCnt)
		}

		value, err := extractValuesFromInsertQuery(insertQuery(&contract))
		if err != nil {
			return err
		}
		contractsValues = append(contractsValues, value)

		value, err = extractValuesFromInsertQuery(insertQuery(&contract_resources))
		if err != nil {
			return err
		}
		contractResourcesValues = append(contractResourcesValues, value)

		billings := rnd(0, 10)
		for j := uint64(0); j < billings; j++ {
			billing := contract_bill_report{
				id:                fmt.Sprintf("contract-bill-report-%d", billCnt),
				contract_id:       contractCnt,
				discount_received: "Default",
				amount_billed:     rnd(0, 100000),
				timestamp:         uint64(time.Now().UnixNano()),
			}
			billCnt++

			value, err = extractValuesFromInsertQuery(insertQuery(&billing))
			if err != nil {
				return err
			}
			billingValues = append(billingValues, value)
		}
		contractCnt++
	}

	if len(contractsValues) != 0 {
		query := "INSERT INTO node_contract (id, grid_version, contract_id, twin_id, node_id, deployment_data, deployment_hash, number_of_public_i_ps, state, created_at, resources_used_id) VALUES"
		query += strings.Join(contractsValues, ",") + ";"
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}

	if len(contractResourcesValues) != 0 {
		query := "INSERT INTO contract_resources (id, hru, sru, cru, mru, contract_id) VALUES"
		query += strings.Join(contractResourcesValues, ",") + ";"
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}

	query := fmt.Sprintf(`UPDATE node_contract SET resources_used_id = CONCAT('contract-resources-',split_part(id, '-', -1))
		WHERE CAST(split_part(id, '-', -1) AS INTEGER) BETWEEN %d AND %d;`, startContractCnt, contractCnt-1)
	if _, err := db.Exec(query); err != nil {
		return err
	}

	if len(billingValues) != 0 {
		query := "INSERT INTO contract_bill_report (id, contract_id, discount_received, amount_billed, timestamp) VALUES"
		query += strings.Join(billingValues, ",") + ";"
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}

	fmt.Println("node contracts generated")

	return nil
}

func generateNameContracts(db *sql.DB) error {
	var contractValues []string
	var billReportsValues []string
	for i := uint64(1); i <= nameContractCount; i++ {
		nodeID := rnd(1, nodeCount)
		state := "Deleted"
		if nodeUP[nodeID] {
			if flip(contractCreatedRatio) {
				state = "Created"
			} else if flip(0.5) {
				state = "GracePeriod"
			}
		}
		twinID := rnd(1100, 3100)
		if renter, ok := renter[nodeID]; ok {
			twinID = renter
		}
		if _, ok := availableRentNodes[nodeID]; ok {
			i--
			continue
		}
		contract := name_contract{
			id:           fmt.Sprintf("name-contract-%d", contractCnt),
			twin_id:      twinID,
			contract_id:  contractCnt,
			state:        state,
			created_at:   uint64(time.Now().Unix()),
			grid_version: 3,
			name:         uuid.NewString(),
		}

		value, err := extractValuesFromInsertQuery(insertQuery(&contract))
		if err != nil {
			return err
		}
		contractValues = append(contractValues, value)

		billings := rnd(0, 10)
		for j := uint64(0); j < billings; j++ {
			billing := contract_bill_report{
				id:                fmt.Sprintf("contract-bill-report-%d", billCnt),
				contract_id:       contractCnt,
				discount_received: "Default",
				amount_billed:     rnd(0, 100000),
				timestamp:         uint64(time.Now().UnixNano()),
			}
			billCnt++

			value, err = extractValuesFromInsertQuery(insertQuery(&billing))
			if err != nil {
				return err
			}
			billReportsValues = append(billReportsValues, value)
		}
		contractCnt++
	}

	if len(contractValues) != 0 {
		query := "INSERT INTO name_contract (id, grid_version, contract_id, twin_id, name, state, created_at) VALUES"
		query += strings.Join(contractValues, ",") + ";"
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}

	if len(billReportsValues) != 0 {
		query := "INSERT INTO contract_bill_report (id, contract_id, discount_received, amount_billed, timestamp) VALUES"
		query += strings.Join(billReportsValues, ",") + ";"
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}

	fmt.Println("name contracts generated")

	return nil
}
func generateRentContracts(db *sql.DB) error {
	var contractValues []string
	var billReportsValues []string
	for i := uint64(1); i <= rentContractCount; i++ {
		nl, nodeID := popRandom(availableRentNodesList)
		availableRentNodesList = nl
		delete(availableRentNodes, nodeID)
		state := "Deleted"
		if nodeUP[nodeID] {
			if flip(0.9) {
				state = "Created"
			} else if flip(0.5) {
				state = "GracePeriod"
			}
		}
		contract := rent_contract{
			id:           fmt.Sprintf("rent-contract-%d", contractCnt),
			twin_id:      rnd(1100, 3100),
			contract_id:  contractCnt,
			state:        state,
			created_at:   uint64(time.Now().Unix()),
			node_id:      nodeID,
			grid_version: 3,
		}
		if state != "Deleted" {
			renter[nodeID] = contract.twin_id
		}

		value, err := extractValuesFromInsertQuery(insertQuery(&contract))
		if err != nil {
			return err
		}
		contractValues = append(contractValues, value)

		billings := rnd(0, 10)
		for j := uint64(0); j < billings; j++ {
			billing := contract_bill_report{
				id:                fmt.Sprintf("contract-bill-report-%d", billCnt),
				contract_id:       contractCnt,
				discount_received: "Default",
				amount_billed:     rnd(0, 100000),
				timestamp:         uint64(time.Now().UnixNano()),
			}

			billCnt++

			value, err = extractValuesFromInsertQuery(insertQuery(&billing))
			if err != nil {
				return err
			}
			billReportsValues = append(billReportsValues, value)

		}
		contractCnt++
	}

	if len(contractValues) != 0 {
		query := "INSERT INTO rent_contract (id, grid_version, contract_id, twin_id, node_id, state, created_at) VALUES"
		query += strings.Join(contractValues, ",") + ";"
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	if len(billReportsValues) != 0 {
		query := "INSERT INTO contract_bill_report (id, contract_id, discount_received, amount_billed, timestamp) VALUES"
		query += strings.Join(billReportsValues, ",") + ";"
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	fmt.Println("rent contracts generated")

	return nil
}

func generateNodes(db *sql.DB) error {
	powerState := []string{"Up", "Down"}
	var locationValues []string
	var nodeValues []string
	var totalResourcesValues []string
	var publicConfigValues []string
	for i := uint64(1); i <= nodeCount; i++ {
		mru := rnd(4, 256) * 1024 * 1024 * 1024
		hru := rnd(100, 30*1024) * 1024 * 1024 * 1024 // 100GB -> 30TB
		sru := rnd(200, 30*1024) * 1024 * 1024 * 1024 // 100GB -> 30TB
		cru := rnd(4, 128)
		up := flip(nodeUpRatio)
		updatedAt := time.Now().Unix() - int64(rnd(60*40*3, 60*60*24*30*12))

		if up {
			updatedAt = time.Now().Unix() - int64(rnd(0, 60*40*1))
		}
		nodesMRU[i] = mru - max(2*uint64(gridtypes.Gigabyte), mru/10)
		nodesSRU[i] = sru - 100*uint64(gridtypes.Gigabyte)
		nodesHRU[i] = hru
		nodeUP[i] = up
		location := location{
			id:        fmt.Sprintf("location-%d", i),
			longitude: fmt.Sprintf("location--long-%d", i),
			latitude:  fmt.Sprintf("location-lat-%d", i),
		}

		countryIndex := rand.Intn(len(countries))
		cityIndex := rand.Intn(len(cities[countries[countryIndex]]))
		node := node{
			id:                fmt.Sprintf("node-%d", i),
			location_id:       fmt.Sprintf("location-%d", i),
			node_id:           i,
			farm_id:           i%100 + 1,
			twin_id:           i + 100 + 1,
			country:           countries[countryIndex],
			city:              cities[countries[countryIndex]][cityIndex],
			uptime:            1000,
			updated_at:        uint64(updatedAt),
			created:           uint64(time.Now().Unix()),
			created_at:        uint64(time.Now().Unix()),
			farming_policy_id: 1,
			grid_version:      3,
			certification:     "Diy",
			secure:            false,
			virtualized:       false,
			serial_number:     "",
			power: nodePower{
				State:  powerState[rand.Intn(len(powerState))],
				Target: powerState[rand.Intn(len(powerState))],
			},
			extra_fee: 0,
		}
		total_resources := node_resources_total{
			id:      fmt.Sprintf("total-resources-%d", i),
			hru:     hru,
			sru:     sru,
			cru:     cru,
			mru:     mru,
			node_id: fmt.Sprintf("node-%d", i),
		}
		if _, ok := dedicatedFarms[node.farm_id]; ok {
			availableRentNodes[i] = struct{}{}
			availableRentNodesList = append(availableRentNodesList, i)
		}

		value, err := extractValuesFromInsertQuery(insertQuery(&location))
		if err != nil {
			return err
		}
		locationValues = append(locationValues, value)

		value, err = extractValuesFromInsertQuery(insertQuery(&node))
		if err != nil {
			return err
		}
		nodeValues = append(nodeValues, value)

		value, err = extractValuesFromInsertQuery(insertQuery(&total_resources))
		if err != nil {
			return err
		}
		totalResourcesValues = append(totalResourcesValues, value)

		if flip(.1) {
			value, err = extractValuesFromInsertQuery(insertQuery(&public_config{
				id:      fmt.Sprintf("public-config-%d", i),
				ipv4:    "185.16.5.2/24",
				gw4:     "185.16.5.2",
				ipv6:    "::1/64",
				gw6:     "::1",
				domain:  "hamada.com",
				node_id: fmt.Sprintf("node-%d", i),
			}))
			if err != nil {
				return err
			}
			publicConfigValues = append(publicConfigValues)

		}
	}

	if len(locationValues) != 0 {
		query := "INSERT INTO location (id, longitude, latitude) VALUES"
		query += strings.Join(locationValues, ",") + ";"
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}

	if len(nodeValues) != 0 {
		query := "INSERT INTO node (id, grid_version, node_id, farm_id, twin_id, country, city, uptime, created, farming_policy_id, certification, secure, virtualized, serial_number, created_at, updated_at, location_id, power, extra_fee) VALUES"
		query += strings.Join(nodeValues, ",") + ";"
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}

	if len(totalResourcesValues) != 0 {
		query := "INSERT INTO node_resources_total (id, hru, sru, cru, mru, node_id) VALUES"
		query += strings.Join(totalResourcesValues, ",") + ";"
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}

	if len(publicConfigValues) != 0 {
		query := "INSERT INTO public_config (id, ipv4, ipv6, gw4, gw6, domain, node_id) VALUES"
		query += strings.Join(publicConfigValues, ",") + ";"
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}

	fmt.Println("nodes generated")

	return nil
}

func generateNodeGPUs(db *sql.DB) error {
	var values []string
	for i := 0; i <= 10; i++ {
		g := node_gpu{
			node_twin_id: uint64(i + 100),
			vendor:       "Advanced Micro Devices, Inc. [AMD/ATI]",
			device:       "Navi 31 [Radeon RX 7900 XT/7900 XTX",
			contract:     i % 2,
			id:           "0000:0e:00.0/1002/744c",
		}

		value, err := extractValuesFromInsertQuery(insertQuery(&g))
		if err != nil {
			return err
		}
		values = append(values, value)
	}

	if len(values) != 0 {
		query := "INSERT INTO node_gpu (node_twin_id, id, vendor, device, contract) VALUES"
		query += strings.Join(values, ",") + ";"
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}

	fmt.Println("node GPUs generated")

	return nil
}

func generateData(db *sql.DB) error {
	if err := generateTwins(db); err != nil {
		panic(err)
	}
	if err := generateFarms(db); err != nil {
		panic(err)
	}
	if err := generateNodes(db); err != nil {
		panic(err)
	}
	if err := generateRentContracts(db); err != nil {
		panic(err)
	}
	if err := generateContracts(db); err != nil {
		panic(err)
	}
	if err := generateNameContracts(db); err != nil {
		panic(err)
	}
	if err := generatePublicIPs(db); err != nil {
		panic(err)
	}
	if err := generateNodeGPUs(db); err != nil {
		panic(err)
	}
	return nil
}
