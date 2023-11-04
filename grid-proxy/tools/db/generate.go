package main

import (
	"database/sql"
	"fmt"
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
	nodeCount            = 1000
	farmCount            = 100
	normalUsers          = 2000
	publicIPCount        = 1000
	twinCount            = nodeCount + farmCount + normalUsers
	nodeContractCount    = 3000
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

const deleted = "Deleted"
const created = "Created"
const gracePeriod = "GracePeriod"

func initSchema(db *sql.DB) error {
	schema, err := os.ReadFile("./schema.sql")
	if err != nil {
		return fmt.Errorf("failed to init schema: %w", err)
	}
	_, err = db.Exec(string(schema))
	if err != nil {
		return fmt.Errorf("failed to init schema: %w", err)
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
		tuple, err := objectToTupleString(twin)
		if err != nil {
			return fmt.Errorf("failed to convert twin object to tuple string: %w", err)
		}
		twins = append(twins, tuple)
	}

	if err := insertTuples(db, twin{}, twins); err != nil {
		return fmt.Errorf("failed to insert twins: %w", err)
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
				return fmt.Errorf("failed to generate random index: %w", err)
			}
			contract_id = createdNodeContracts[idx]
		}
		ip := randomIPv4()
		farmID, err := rnd(1, farmCount)
		if err != nil {
			return fmt.Errorf("failed to generate random farm id: %w", err)
		}

		public_ip := public_ip{
			id:          fmt.Sprintf("public-ip-%d", i),
			gateway:     ip.String(),
			ip:          IPv4Subnet(ip).String(),
			contract_id: contract_id,
			farm_id:     fmt.Sprintf("farm-%d", farmID),
		}
		publicIpTuple, err := objectToTupleString(public_ip)
		if err != nil {
			return fmt.Errorf("failed to convert public ip object to tuple string: %w", err)
		}
		publicIPs = append(publicIPs, publicIpTuple)
		nodeContracts = append(nodeContracts, contract_id)
	}

	if err := insertTuples(db, public_ip{}, publicIPs); err != nil {
		return fmt.Errorf("failed to insert public ips: %w", err)
	}

	if err := updateNodeContractPublicIPs(db, nodeContracts); err != nil {
		return fmt.Errorf("failed to update contract public ips: %w", err)
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

		farmTuple, err := objectToTupleString(farm)
		if err != nil {
			return fmt.Errorf("failed to convert farm object to tuple string: %w", err)
		}
		farms = append(farms, farmTuple)
	}

	if err := insertTuples(db, farm{}, farms); err != nil {
		return fmt.Errorf("failed to insert farms: %w", err)
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
			return nil, contractsStartID, fmt.Errorf("failed to generate random node id: %w", err)
		}
		state := deleted

		if nodeUP[nodeID] {
			if flip(contractCreatedRatio) {
				state = created
			} else if flip(0.5) {
				state = gracePeriod
			}
		}

		if state != deleted && (minContractHRU > nodesHRU[nodeID] || minContractMRU > nodesMRU[nodeID] || minContractSRU > nodesSRU[nodeID]) {
			i--
			continue
		}

		twinID, err := rnd(1100, 3100)
		if err != nil {
			return nil, contractsStartID, fmt.Errorf("failed to generate random twin id: %w", err)
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
			return nil, contractsStartID, fmt.Errorf("failed to generate random cru: %w", err)
		}

		hru, err := rnd(minContractHRU, min(maxContractHRU, nodesHRU[nodeID]))
		if err != nil {
			return nil, contractsStartID, fmt.Errorf("failed to generate random hru: %w", err)
		}

		sru, err := rnd(minContractSRU, min(maxContractSRU, nodesSRU[nodeID]))
		if err != nil {
			return nil, contractsStartID, fmt.Errorf("failed to generate random sru: %w", err)
		}

		mru, err := rnd(minContractMRU, min(maxContractMRU, nodesMRU[nodeID]))
		if err != nil {
			return nil, contractsStartID, fmt.Errorf("failed to generate random mru: %w", err)
		}

		contract_resources := contract_resources{
			id:          fmt.Sprintf("contract-resources-%d", contractsStartID),
			hru:         hru,
			sru:         sru,
			cru:         cru,
			mru:         mru,
			contract_id: fmt.Sprintf("node-contract-%d", contractsStartID),
		}
		if contract.state != deleted {
			nodesHRU[nodeID] -= hru
			nodesSRU[nodeID] -= sru
			nodesMRU[nodeID] -= mru
			createdNodeContracts = append(createdNodeContracts, uint64(contractsStartID))
		}

		contractTuple, err := objectToTupleString(contract)
		if err != nil {
			return nil, contractsStartID, fmt.Errorf("failed to convert contract object to tuple string: %w", err)
		}
		contracts = append(contracts, contractTuple)

		contractResourcesTuple, err := objectToTupleString(contract_resources)
		if err != nil {
			return nil, contractsStartID, fmt.Errorf("failed to convert contract resources object to tuple string: %w", err)
		}
		contractResources = append(contractResources, contractResourcesTuple)

		billings, err := rnd(0, 10)
		if err != nil {
			return nil, contractsStartID, fmt.Errorf("failed to generate random billing count: %w", err)
		}

		amountBilled, err := rnd(0, 100000)
		if err != nil {
			return nil, contractsStartID, fmt.Errorf("failed to generate random amount billed: %w", err)
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

			billTuple, err := objectToTupleString(billing)
			if err != nil {
				return nil, contractsStartID, fmt.Errorf("failed to convert contract bill report object to tuple string: %w", err)
			}
			billingReports = append(billingReports, billTuple)
		}
		contractsStartID++
	}

	if err := insertTuples(db, node_contract{}, contracts); err != nil {
		return nil, contractsStartID, fmt.Errorf("failed to insert node contracts: %w", err)
	}

	if err := insertTuples(db, contract_resources{}, contractResources); err != nil {
		return nil, contractsStartID, fmt.Errorf("failed to insert contract resources: %w", err)
	}

	if err := updateNodeContractResourceID(db, contractsStartID-nodeContractCount, contractsStartID); err != nil {
		return nil, contractsStartID, fmt.Errorf("failed to update node contract resources id: %w", err)
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
			return nil, contractsStartID, fmt.Errorf("failed to generate random node id: %w", err)
		}

		state := deleted
		if nodeUP[nodeID] {
			if flip(contractCreatedRatio) {
				state = created
			} else if flip(0.5) {
				state = gracePeriod
			}
		}

		twinID, err := rnd(1100, 3100)
		if err != nil {
			return nil, contractsStartID, fmt.Errorf("failed to generate random twin id: %w", err)
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

		contractTuple, err := objectToTupleString(contract)
		if err != nil {
			return nil, contractsStartID, fmt.Errorf("failed to convert contract object to tuple string: %w", err)
		}
		contracts = append(contracts, contractTuple)

		billings, err := rnd(0, 10)
		if err != nil {
			return nil, contractsStartID, fmt.Errorf("failed to generate random billings count: %w", err)
		}
		amountBilled, err := rnd(0, 100000)
		if err != nil {
			return nil, contractsStartID, fmt.Errorf("failed to generate random amount billed: %w", err)
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

			billTuple, err := objectToTupleString(billing)
			if err != nil {
				return nil, contractsStartID, fmt.Errorf("failed to convert contract bill report object to tuple string: %w", err)
			}
			billReports = append(billReports, billTuple)
		}
		contractsStartID++
	}

	if err := insertTuples(db, name_contract{}, contracts); err != nil {
		return nil, contractsStartID, fmt.Errorf("failed to insert name contracts: %w", err)
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
			return nil, contractsStartID, fmt.Errorf("failed to select random element from the gives slice: %w", err)
		}

		availableRentNodesList = nl
		delete(availableRentNodes, nodeID)
		state := deleted
		if nodeUP[nodeID] {
			if flip(0.9) {
				state = created
			} else if flip(0.5) {
				state = gracePeriod
			}
		}

		twinID, err := rnd(1100, 3100)
		if err != nil {
			return nil, contractsStartID, fmt.Errorf("failed to generate a random twin id: %w", err)
		}

		contract := rent_contract{
			id:           fmt.Sprintf("rent-contract-%d", contractsStartID),
			twin_id:      twinID,
			contract_id:  uint64(contractsStartID),
			state:        state,
			created_at:   uint64(time.Now().Unix()),
			node_id:      nodeID,
			grid_version: 3,
		}

		if state != deleted {
			renter[nodeID] = contract.twin_id
		}

		contractTuple, err := objectToTupleString(contract)
		if err != nil {
			return nil, contractsStartID, fmt.Errorf("failed to convert contract object to tuple string: %w", err)
		}
		contracts = append(contracts, contractTuple)

		billings, err := rnd(0, 10)
		if err != nil {
			return nil, contractsStartID, fmt.Errorf("failed to generate billings count: %w", err)
		}

		amountBilled, err := rnd(0, 100000)
		if err != nil {
			return nil, contractsStartID, fmt.Errorf("failed to generate amount billed: %w", err)
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

			billTuple, err := objectToTupleString(billing)
			if err != nil {
				return nil, contractsStartID, fmt.Errorf("failed to convert contract bill report object to tuple string: %w", err)
			}
			billReports = append(billReports, billTuple)

		}
		contractsStartID++
	}

	if err := insertTuples(db, rent_contract{}, contracts); err != nil {
		return nil, contractsStartID, fmt.Errorf("failed to insert rent contracts: %w", err)
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
			return fmt.Errorf("failed to generate random mru: %w", err)
		}
		mru *= 1024 * 1024 * 1024

		hru, err := rnd(100, 30*1024)
		if err != nil {
			return fmt.Errorf("failed to generate random hru: %w", err)
		}
		hru *= 1024 * 1024 * 1024 // 100GB -> 30TB

		sru, err := rnd(200, 30*1024)
		if err != nil {
			return fmt.Errorf("failed to generate random sru: %w", err)
		}
		sru *= 1024 * 1024 * 1024 // 100GB -> 30TB

		cru, err := rnd(4, 128)
		if err != nil {
			return fmt.Errorf("failed to generate random cru: %w", err)
		}

		up := flip(nodeUpRatio)
		periodFromLatestUpdate, err := rnd(60*40*3, 60*60*24*30*12)
		if err != nil {
			return fmt.Errorf("failed to generate random period from latest update: %w", err)
		}
		updatedAt := time.Now().Unix() - int64(periodFromLatestUpdate)

		if up {
			periodFromLatestUpdate, err = rnd(0, 60*40*1)
			if err != nil {
				return fmt.Errorf("failed to generate period from latest update: %w", err)
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

		countryIndex := r.Intn(len(countries))
		cityIndex := r.Intn(len(cities[countries[countryIndex]]))
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
				State:  powerState[r.Intn(len(powerState))],
				Target: powerState[r.Intn(len(powerState))],
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

		locationTuple, err := objectToTupleString(location)
		if err != nil {
			return fmt.Errorf("failed to convert location object to tuple string: %w", err)
		}
		locations = append(locations, locationTuple)

		nodeTuple, err := objectToTupleString(node)
		if err != nil {
			return fmt.Errorf("failed to convert node object to tuple string: %w", err)
		}
		nodes = append(nodes, nodeTuple)

		totalResourcesTuple, err := objectToTupleString(total_resources)
		if err != nil {
			return fmt.Errorf("failed to convert total resources object to tuple string: %w", err)
		}
		totalResources = append(totalResources, totalResourcesTuple)

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
			publicConfigTuple, err := objectToTupleString(publicConfig)
			if err != nil {
				return fmt.Errorf("failed to convert public config object to tuple string: %w", err)
			}
			publicConfigs = append(publicConfigs, publicConfigTuple)

		}
	}

	if err := insertTuples(db, location{}, locations); err != nil {
		return fmt.Errorf("failed to insert locations: %w", err)
	}

	if err := insertTuples(db, node{}, nodes); err != nil {
		return fmt.Errorf("failed to isnert nodes: %w", err)
	}

	if err := insertTuples(db, node_resources_total{}, totalResources); err != nil {
		return fmt.Errorf("failed to insert node resources total: %w", err)
	}

	if err := insertTuples(db, public_config{}, publicConfigs); err != nil {
		return fmt.Errorf("failed to insert public configs: %w", err)
	}
	fmt.Println("nodes generated")

	return nil
}

func generateNodeGPUs(db *sql.DB) error {
	var GPUs []string
	vendors := []string{"NVIDIA Corporation", "AMD", "Intel Corporation"}
	devices := []string{"GeForce RTX 3080", "Radeon RX 6800 XT", "Intel Iris Xe MAX"}

	for i := 0; i <= 10; i++ {
		gpuNum := len(vendors)
		for j := 0; j <= gpuNum; j++ {
			g := node_gpu{
				node_twin_id: uint64(i + 100),
				vendor:       vendors[j],
				device:       devices[j],
				contract:     i % 2,
				id:           fmt.Sprintf("0000:0e:00.0/1002/744c/%d", j),
			}
			gpuTuple, err := objectToTupleString(g)
			if err != nil {
				return fmt.Errorf("failed to convert gpu object to tuple string: %w", err)
			}
			GPUs = append(GPUs, gpuTuple)
		}
	}

	if err := insertTuples(db, node_gpu{}, GPUs); err != nil {
		return fmt.Errorf("failed to insert node gpu: %w", err)
	}

	fmt.Println("node GPUs generated")

	return nil
}

func generateContracts(db *sql.DB) error {
	rentContractIDStart := 1

	var billReports []string

	rentContractsBillReports, nodeContractIDStart, err := generateRentContracts(db, 1, rentContractIDStart)
	if err != nil {
		return fmt.Errorf("failed to generate rent contracts: %w", err)
	}
	billReports = append(billReports, rentContractsBillReports...)

	nodeContractsBillReports, nameContractIDStart, err := generateNodeContracts(db, len(billReports)+1, nodeContractIDStart)
	if err != nil {
		return fmt.Errorf("failed to generate node contracts: %w", err)
	}
	billReports = append(billReports, nodeContractsBillReports...)

	nameContractsBillReports, _, err := generateNameContracts(db, len(billReports)+1, nameContractIDStart)
	if err != nil {
		return fmt.Errorf("failed to generate name contracts: %w", err)
	}
	billReports = append(billReports, nameContractsBillReports...)

	if err := insertTuples(db, contract_bill_report{}, billReports); err != nil {
		return fmt.Errorf("failed to generate contract bill reports: %w", err)
	}
	return nil
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
			return fmt.Errorf("failed to insert tuples: %w", err)
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
			return fmt.Errorf("failed to update node contracts public ips: %w", err)
		}
	}
	return nil
}

func updateNodeContractResourceID(db *sql.DB, min, max int) error {
	query := fmt.Sprintf(`UPDATE node_contract SET resources_used_id = CONCAT('contract-resources-',split_part(id, '-', -1))
		WHERE CAST(split_part(id, '-', -1) AS INTEGER) BETWEEN %d AND %d;`, min, max)
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to update node contract resource id: %w", err)
	}
	return nil
}
func generateData(db *sql.DB) error {
	if err := generateTwins(db); err != nil {
		return fmt.Errorf("failed to genrate twins: %w", err)
	}

	if err := generateFarms(db); err != nil {
		return fmt.Errorf("failed to generate farms: %w", err)
	}

	if err := generateNodes(db); err != nil {
		return fmt.Errorf("failed to generate nodes: %w", err)
	}

	if err := generateContracts(db); err != nil {
		return fmt.Errorf("failed to generate contracts: %w", err)
	}

	if err := generatePublicIPs(db); err != nil {
		return fmt.Errorf("failed to generate public ips: %w", err)
	}

	if err := generateNodeGPUs(db); err != nil {
		return fmt.Errorf("failed to generate node gpus: %w", err)
	}
	return nil
}
