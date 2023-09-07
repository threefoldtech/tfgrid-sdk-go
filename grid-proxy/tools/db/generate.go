package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"reflect"
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
)

const (
	contractCreatedRatio = .1 // from devnet
	usedPublicIPsRatio   = .9
	nodeUpRatio          = .5
	nodeCount            = 100000
	farmCount            = 10000
	normalUsers          = 50000
	publicIPCount        = 1000
	twinCount            = nodeCount + farmCount + normalUsers
	nodeContractCount    = 100000
	rentContractCount    = 1000
	nameContractCount    = 1000

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
	var twins []string

	for i := uint64(1); i <= twinCount; i++ {
		twin := twin{
			id:           fmt.Sprintf("twin-%d", i),
			account_id:   fmt.Sprintf("account-id-%d", i),
			relay:        fmt.Sprintf("relay-%d", i),
			public_key:   fmt.Sprintf("public-key-%d", i),
			twin_id:      i,
			grid_version: 3,
		}
		twins = append(twins, objectToTupleString(twin))
	}

	if err := insertTuples(db, twin{}, twins); err != nil {
		return err
	}
	fmt.Println("twins generated")

	return nil
}

func generatePublicIPs(db *sql.DB) error {
	var publicIPs []string
	var nodeContracts []uint64

	for i := uint64(1); i <= publicIPCount; i++ {
		contract_id := uint64(0)
		if flip(usedPublicIPsRatio) {
			idx, err := rnd(0, uint64(len(createdNodeContracts))-1)
			if err != nil {
				return err
			}
			contract_id = createdNodeContracts[idx]
		}
		ip := randomIPv4()
		farmID, err := rnd(1, farmCount)
		if err != nil {
			return err
		}

		public_ip := public_ip{
			id:          fmt.Sprintf("public-ip-%d", i),
			gateway:     ip.String(),
			ip:          IPv4Subnet(ip).String(),
			contract_id: contract_id,
			farm_id:     fmt.Sprintf("farm-%d", farmID),
		}
		publicIPs = append(publicIPs, objectToTupleString(public_ip))
		nodeContracts = append(nodeContracts, contract_id)
	}

	if err := insertTuples(db, public_ip{}, publicIPs); err != nil {
		return err
	}

	if err := updateNodeContractPublicIPs(db, nodeContracts); err != nil {
		return err
	}

	fmt.Println("public IPs generated")

	return nil
}

func generateFarms(db *sql.DB) error {
	var farms []string

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

		farms = append(farms, objectToTupleString(farm))
	}

	if err := insertTuples(db, farm{}, farms); err != nil {
		return err
	}
	fmt.Println("farms generated")

	return nil
}

func generateNodeContracts(db *sql.DB, billsStartID, contractsStartID int) ([]string, int, error) {
	var contracts []string
	var contractResources []string
	var billingReports []string

	for i := uint64(1); i <= nodeContractCount; i++ {
		nodeID, err := rnd(1, nodeCount)
		if err != nil {
			return nil, contractsStartID, err
		}
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

		twinID, err := rnd(1100, 3100)
		if err != nil {
			return nil, contractsStartID, err
		}

		if renter, ok := renter[nodeID]; ok {
			twinID = renter
		}

		if _, ok := availableRentNodes[nodeID]; ok {
			i--
			continue
		}

		contract := node_contract{
			id:                    fmt.Sprintf("node-contract-%d", contractsStartID),
			twin_id:               twinID,
			contract_id:           uint64(contractsStartID),
			state:                 state,
			created_at:            uint64(time.Now().Unix()),
			node_id:               nodeID,
			deployment_data:       fmt.Sprintf("deployment-data-%d", contractsStartID),
			deployment_hash:       fmt.Sprintf("deployment-hash-%d", contractsStartID),
			number_of_public_i_ps: 0,
			grid_version:          3,
			resources_used_id:     "",
		}

		cru, err := rnd(minContractCRU, maxContractCRU)
		if err != nil {
			return nil, contractsStartID, err
		}

		hru, err := rnd(minContractHRU, min(maxContractHRU, nodesHRU[nodeID]))
		if err != nil {
			return nil, contractsStartID, err
		}

		sru, err := rnd(minContractSRU, min(maxContractSRU, nodesSRU[nodeID]))
		if err != nil {
			return nil, contractsStartID, err
		}

		mru, err := rnd(minContractMRU, min(maxContractMRU, nodesMRU[nodeID]))
		if err != nil {
			return nil, contractsStartID, err
		}

		contract_resources := contract_resources{
			id:          fmt.Sprintf("contract-resources-%d", contractsStartID),
			hru:         hru,
			sru:         sru,
			cru:         cru,
			mru:         mru,
			contract_id: fmt.Sprintf("node-contract-%d", contractsStartID),
		}
		if contract.state != "Deleted" {
			nodesHRU[nodeID] -= hru
			nodesSRU[nodeID] -= sru
			nodesMRU[nodeID] -= mru
			createdNodeContracts = append(createdNodeContracts, uint64(contractsStartID))
		}

		contracts = append(contracts, objectToTupleString(contract))

		contractResources = append(contractResources, objectToTupleString(contract_resources))

		billings, err := rnd(0, 10)
		if err != nil {
			return nil, contractsStartID, err
		}

		amountBilled, err := rnd(0, 100000)
		if err != nil {
			return nil, contractsStartID, err
		}
		for j := uint64(0); j < billings; j++ {
			billing := contract_bill_report{
				id:                fmt.Sprintf("contract-bill-report-%d", billsStartID),
				contract_id:       uint64(contractsStartID),
				discount_received: "Default",
				amount_billed:     amountBilled,
				timestamp:         uint64(time.Now().UnixNano()),
			}
			billsStartID++

			billingReports = append(billingReports, objectToTupleString(billing))
		}
		contractsStartID++
	}

	if err := insertTuples(db, node_contract{}, contracts); err != nil {
		return nil, contractsStartID, err
	}

	if err := insertTuples(db, contract_resources{}, contractResources); err != nil {
		return nil, contractsStartID, err
	}

	if err := updateNodeContractResourceID(db, contractsStartID-nodeContractCount, contractsStartID); err != nil {
		return nil, contractsStartID, err
	}

	fmt.Println("node contracts generated")

	return billingReports, contractsStartID, nil
}

func generateNameContracts(db *sql.DB, billsStartID, contractsStartID int) ([]string, int, error) {
	var contracts []string
	var billReports []string
	for i := uint64(1); i <= nameContractCount; i++ {
		nodeID, err := rnd(1, nodeCount)
		if err != nil {
			return nil, contractsStartID, err
		}

		state := "Deleted"
		if nodeUP[nodeID] {
			if flip(contractCreatedRatio) {
				state = "Created"
			} else if flip(0.5) {
				state = "GracePeriod"
			}
		}

		twinID, err := rnd(1100, 3100)
		if err != nil {
			return nil, contractsStartID, err
		}

		if renter, ok := renter[nodeID]; ok {
			twinID = renter
		}

		if _, ok := availableRentNodes[nodeID]; ok {
			i--
			continue
		}

		contract := name_contract{
			id:           fmt.Sprintf("name-contract-%d", contractsStartID),
			twin_id:      twinID,
			contract_id:  uint64(contractsStartID),
			state:        state,
			created_at:   uint64(time.Now().Unix()),
			grid_version: 3,
			name:         uuid.NewString(),
		}

		contracts = append(contracts, objectToTupleString(contract))

		billings, err := rnd(0, 10)
		if err != nil {
			return nil, contractsStartID, err
		}
		amountBilled, err := rnd(0, 100000)
		for j := uint64(0); j < billings; j++ {
			billing := contract_bill_report{
				id:                fmt.Sprintf("contract-bill-report-%d", billsStartID),
				contract_id:       uint64(contractsStartID),
				discount_received: "Default",
				amount_billed:     amountBilled,
				timestamp:         uint64(time.Now().UnixNano()),
			}
			billsStartID++

			billReports = append(billReports, objectToTupleString(billing))
		}
		contractsStartID++
	}

	if err := insertTuples(db, name_contract{}, contracts); err != nil {
		return nil, contractsStartID, err
	}

	fmt.Println("name contracts generated")

	return billReports, contractsStartID, nil
}
func generateRentContracts(db *sql.DB, billsStartID, contractsStartID int) ([]string, int, error) {
	var contracts []string
	var billReports []string
	for i := uint64(1); i <= rentContractCount; i++ {
		nl, nodeID, err := popRandom(availableRentNodesList)
		if err != nil {
			return nil, contractsStartID, err
		}

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
		twinID, err := rnd(1100, 3100)

		contract := rent_contract{
			id:           fmt.Sprintf("rent-contract-%d", contractsStartID),
			twin_id:      twinID,
			contract_id:  uint64(contractsStartID),
			state:        state,
			created_at:   uint64(time.Now().Unix()),
			node_id:      nodeID,
			grid_version: 3,
		}

		if state != "Deleted" {
			renter[nodeID] = contract.twin_id
		}

		contracts = append(contracts, objectToTupleString(contract))

		billings, err := rnd(0, 10)
		if err != nil {
			return nil, contractsStartID, err
		}

		amountBilled, err := rnd(0, 100000)
		if err != nil {
			return nil, contractsStartID, err
		}

		for j := uint64(0); j < billings; j++ {
			billing := contract_bill_report{
				id:                fmt.Sprintf("contract-bill-report-%d", billsStartID),
				contract_id:       uint64(contractsStartID),
				discount_received: "Default",
				amount_billed:     amountBilled,
				timestamp:         uint64(time.Now().UnixNano()),
			}

			billsStartID++

			billReports = append(billReports, objectToTupleString(billing))

		}
		contractsStartID++
	}

	if err := insertTuples(db, rent_contract{}, contracts); err != nil {
		return nil, contractsStartID, err
	}

	fmt.Println("rent contracts generated")

	return billReports, contractsStartID, nil
}

func generateNodes(db *sql.DB) error {
	powerState := []string{"Up", "Down"}
	var locations []string
	var nodes []string
	var totalResources []string
	var publicConfigs []string
	for i := uint64(1); i <= nodeCount; i++ {
		mru, err := rnd(4, 256)
		if err != nil {
			return err
		}
		mru *= 1024 * 1024 * 1024

		hru, err := rnd(100, 30*1024)
		if err != nil {
			return err
		}
		hru *= 1024 * 1024 * 1024 // 100GB -> 30TB

		sru, err := rnd(200, 30*1024)
		if err != nil {
			return err
		}
		sru *= 1024 * 1024 * 1024 // 100GB -> 30TB

		cru, err := rnd(4, 128)
		if err != nil {
			return err
		}

		up := flip(nodeUpRatio)
		periodFromLatestUpdate, err := rnd(60*40*3, 60*60*24*30*12)
		if err != nil {
			return err
		}
		updatedAt := time.Now().Unix() - int64(periodFromLatestUpdate)

		if up {
			periodFromLatestUpdate, err = rnd(0, 60*40*1)
			if err != nil {
				return err
			}
			updatedAt = time.Now().Unix() - int64(periodFromLatestUpdate)
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

		locations = append(locations, objectToTupleString(location))

		nodes = append(nodes, objectToTupleString(node))

		totalResources = append(totalResources, objectToTupleString(total_resources))

		if flip(.1) {
			publicConfig := public_config{
				id:      fmt.Sprintf("public-config-%d", i),
				ipv4:    "185.16.5.2/24",
				gw4:     "185.16.5.2",
				ipv6:    "::1/64",
				gw6:     "::1",
				domain:  "hamada.com",
				node_id: fmt.Sprintf("node-%d", i),
			}
			publicConfigs = append(publicConfigs, objectToTupleString(publicConfig))

		}
	}

	if err := insertTuples(db, location{}, locations); err != nil {
		return err
	}

	if err := insertTuples(db, node{}, nodes); err != nil {
		return err
	}

	if err := insertTuples(db, node_resources_total{}, totalResources); err != nil {
		return err
	}

	if err := insertTuples(db, public_config{}, publicConfigs); err != nil {
		return err
	}
	fmt.Println("nodes generated")

	return nil
}

func generateNodeGPUs(db *sql.DB) error {
	var GPUs []string
	for i := 0; i <= 10; i++ {
		g := node_gpu{
			node_twin_id: uint64(i + 100),
			vendor:       "Advanced Micro Devices, Inc. [AMD/ATI]",
			device:       "Navi 31 [Radeon RX 7900 XT/7900 XTX",
			contract:     i % 2,
			id:           "0000:0e:00.0/1002/744c",
		}

		GPUs = append(GPUs, objectToTupleString(g))
	}

	if err := insertTuples(db, node_gpu{}, GPUs); err != nil {
		return err
	}

	fmt.Println("node GPUs generated")

	return nil

}

func generateContracts(db *sql.DB) error {
	contractsStartID := 1

	var billReports []string

	rentContractsBillReports, contractCount, err := generateRentContracts(db, 1, contractsStartID)
	if err != nil {
		return err
	}
	billReports = append(billReports, rentContractsBillReports...)

	nodeContractsBillReports, contractsStartID, err := generateNodeContracts(db, len(billReports)+1, contractCount)
	if err != nil {
		return err
	}
	billReports = append(billReports, nodeContractsBillReports...)

	nameContractsBillReports, contractsStartID, err := generateNameContracts(db, len(billReports)+1, contractCount)
	if err != nil {
		return err
	}
	billReports = append(billReports, nameContractsBillReports...)

	return insertTuples(db, contract_bill_report{}, billReports)
}

func insertTuples(db *sql.DB, tupleObj interface{}, tuples []string) error {

	if len(tuples) != 0 {
		query := "INSERT INTO  " + reflect.Indirect(reflect.ValueOf(tupleObj)).Type().Name() + " ("
		objType := reflect.TypeOf(tupleObj)
		for i := 0; i < objType.NumField(); i++ {
			if i != 0 {
				query += ", "
			}
			query += objType.Field(i).Name
		}

		query += ") VALUES "

		query += strings.Join(tuples, ",")
		query += ";"
		if _, err := db.Exec(query); err != nil {
			return err
		}

	}
	return nil
}

func updateNodeContractPublicIPs(db *sql.DB, nodeContracts []uint64) error {

	if len(nodeContracts) != 0 {
		var IDs []string
		for _, contractID := range nodeContracts {
			IDs = append(IDs, fmt.Sprintf("%d", contractID))

		}

		query := "UPDATE node_contract set number_of_public_i_ps = number_of_public_i_ps + 1 WHERE contract_id IN ("
		query += strings.Join(IDs, ",") + ");"
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func updateNodeContractResourceID(db *sql.DB, min, max int) error {
	query := fmt.Sprintf(`UPDATE node_contract SET resources_used_id = CONCAT('contract-resources-',split_part(id, '-', -1))
		WHERE CAST(split_part(id, '-', -1) AS INTEGER) BETWEEN %d AND %d;`, min, max)
	if _, err := db.Exec(query); err != nil {
		return err
	}

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

	if err := generateContracts(db); err != nil {
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
