package modifiers

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

const deleted = "Deleted"
const created = "Created"
const gracePeriod = "GracePeriod"

func (g *Generator) GenerateTwins(start, size int) error {
	var twins []string

	for i := uint64(start); i < uint64(start+size); i++ {
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

	if err := g.insertTuples(twin{}, twins); err != nil {
		return fmt.Errorf("failed to insert twins: %w", err)
	}
	fmt.Println("twins generated")

	return nil
}

func (g *Generator) GenerateFarms(start, size, twinStart int) error {
	var farms []string

	for i := uint64(start); i < uint64(start+size); i++ {
		farm := farm{
			id:                fmt.Sprintf("farm-%d", i),
			farm_id:           i,
			name:              fmt.Sprintf("farm-name-%d", i),
			certification:     "Diy",
			dedicated_farm:    flip(.1),
			twin_id:           uint64(twinStart) + (i - uint64(start)),
			pricing_policy_id: 1,
			grid_version:      3,
			stellar_address:   "",
		}

		if farm.dedicated_farm {
			g.dedicatedFarms[farm.farm_id] = struct{}{}
		}

		farmTuple, err := objectToTupleString(farm)
		if err != nil {
			return fmt.Errorf("failed to convert farm object to tuple string: %w", err)
		}
		farms = append(farms, farmTuple)
	}

	if err := g.insertTuples(farm{}, farms); err != nil {
		return fmt.Errorf("failed to insert farms: %w", err)
	}
	fmt.Println("farms generated")

	return nil
}

func (g *Generator) GenerateNodes(start, size, farmStart, farmSize, twinStart int) error {
	powerState := []string{"Up", "Down"}
	var locations []string
	var nodes []string
	var totalResources []string
	var publicConfigs []string
	for i := uint64(start); i < uint64(start+size); i++ {
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

		g.nodesMRU[i] = mru - max(2*uint64(gridtypes.Gigabyte), mru/10)
		g.nodesSRU[i] = sru - 100*uint64(gridtypes.Gigabyte)
		g.nodesHRU[i] = hru
		g.nodeUP[i] = up

		// location latitude and longitue needs to be castable to decimal
		// if not, the convert_to_decimal function will raise a notice
		// reporting the incident, which downgrades performance
		location := location{
			id:        fmt.Sprintf("location-%d", i),
			longitude: fmt.Sprintf("%d", i),
			latitude:  fmt.Sprintf("%d", i),
		}

		countryIndex := r.Intn(len(countries))
		cityIndex := r.Intn(len(cities[countries[countryIndex]]))
		farmId := r.Intn((farmStart+farmSize)-farmStart) + farmStart
		twinId := twinStart + farmSize + (int(i) - start)
		node := node{
			id:                fmt.Sprintf("node-%d", i),
			location_id:       fmt.Sprintf("location-%d", i),
			node_id:           i,
			farm_id:           uint64(farmId),
			twin_id:           uint64(twinId),
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

		if _, ok := g.dedicatedFarms[node.farm_id]; ok {
			g.availableRentNodes[i] = struct{}{}
			g.availableRentNodesList = append(g.availableRentNodesList, i)
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

	if err := g.insertTuples(location{}, locations); err != nil {
		return fmt.Errorf("failed to insert locations: %w", err)
	}

	if err := g.insertTuples(node{}, nodes); err != nil {
		return fmt.Errorf("failed to isnert nodes: %w", err)
	}

	if err := g.insertTuples(node_resources_total{}, totalResources); err != nil {
		return fmt.Errorf("failed to insert node resources total: %w", err)
	}

	if err := g.insertTuples(public_config{}, publicConfigs); err != nil {
		return fmt.Errorf("failed to insert public configs: %w", err)
	}
	fmt.Println("nodes generated")

	return nil
}

func (g *Generator) GenerateContracts(billStart, contractStart, nodeConCount, nameConCount, rentConCount, nodeStart, nodeSize int) error {
	var billReports []string

	rentContractsBillReports, nodeContractIDStart, err := g.GenerateRentContracts(billStart, contractStart, rentConCount)
	if err != nil {
		return fmt.Errorf("failed to generate rent contracts: %w", err)
	}
	billReports = append(billReports, rentContractsBillReports...)

	nodeContractsBillReports, nameContractIDStart, err := g.generateNodeContracts(len(billReports)+billStart, nodeContractIDStart, nodeConCount, nodeStart, nodeSize)
	if err != nil {
		return fmt.Errorf("failed to generate node contracts: %w", err)
	}
	billReports = append(billReports, nodeContractsBillReports...)

	nameContractsBillReports, _, err := g.GenerateNameContracts(len(billReports)+billStart, nameContractIDStart, nameConCount)
	if err != nil {
		return fmt.Errorf("failed to generate name contracts: %w", err)
	}
	billReports = append(billReports, nameContractsBillReports...)

	if err := g.insertTuples(contract_bill_report{}, billReports); err != nil {
		return fmt.Errorf("failed to generate contract bill reports: %w", err)
	}
	return nil
}

func (g *Generator) generateNodeContracts(billsStartID, contractsStartID, contractCount, nodeStart, nodeSize int) ([]string, int, error) {
	end := contractsStartID + contractCount

	var contracts []string
	var contractResources []string
	var billingReports []string

	for i := contractsStartID; i < end; i++ {
		nodeID := uint64(r.Intn((nodeStart+nodeSize)-nodeStart) + nodeStart)
		state := deleted

		if g.nodeUP[nodeID] {
			if flip(contractCreatedRatio) {
				state = created
			} else if flip(0.5) {
				state = gracePeriod
			}
		}

		if state != deleted && (minContractHRU > g.nodesHRU[nodeID] || minContractMRU > g.nodesMRU[nodeID] || minContractSRU > g.nodesSRU[nodeID]) {
			i--
			continue
		}

		twinID, err := rnd(1100, 3100)
		if err != nil {
			return nil, i, fmt.Errorf("failed to generate random twin id: %w", err)
		}

		if renter, ok := g.renter[nodeID]; ok {
			twinID = renter
		}

		if _, ok := g.availableRentNodes[nodeID]; ok {
			i--
			continue
		}

		contract := node_contract{
			id:                    fmt.Sprintf("node-contract-%d", i),
			twin_id:               twinID,
			contract_id:           uint64(i),
			state:                 state,
			created_at:            uint64(time.Now().Unix()),
			node_id:               nodeID,
			deployment_data:       fmt.Sprintf("deployment-data-%d", i),
			deployment_hash:       fmt.Sprintf("deployment-hash-%d", i),
			number_of_public_i_ps: 0,
			grid_version:          3,
			resources_used_id:     "",
		}

		cru, err := rnd(minContractCRU, maxContractCRU)
		if err != nil {
			return nil, i, fmt.Errorf("failed to generate random cru: %w", err)
		}

		hru, err := rnd(minContractHRU, min(maxContractHRU, g.nodesHRU[nodeID]))
		if err != nil {
			return nil, i, fmt.Errorf("failed to generate random hru: %w", err)
		}

		sru, err := rnd(minContractSRU, min(maxContractSRU, g.nodesSRU[nodeID]))
		if err != nil {
			return nil, i, fmt.Errorf("failed to generate random sru: %w", err)
		}

		mru, err := rnd(minContractMRU, min(maxContractMRU, g.nodesMRU[nodeID]))
		if err != nil {
			return nil, i, fmt.Errorf("failed to generate random mru: %w", err)
		}

		contract_resources := contract_resources{
			id:          fmt.Sprintf("contract-resources-%d", i),
			hru:         hru,
			sru:         sru,
			cru:         cru,
			mru:         mru,
			contract_id: fmt.Sprintf("node-contract-%d", i),
		}
		if contract.state != deleted {
			g.nodesHRU[nodeID] -= hru
			g.nodesSRU[nodeID] -= sru
			g.nodesMRU[nodeID] -= mru
			g.createdNodeContracts = append(g.createdNodeContracts, uint64(i))
		}

		contractTuple, err := objectToTupleString(contract)
		if err != nil {
			return nil, i, fmt.Errorf("failed to convert contract object to tuple string: %w", err)
		}
		contracts = append(contracts, contractTuple)

		contractResourcesTuple, err := objectToTupleString(contract_resources)
		if err != nil {
			return nil, i, fmt.Errorf("failed to convert contract resources object to tuple string: %w", err)
		}
		contractResources = append(contractResources, contractResourcesTuple)

		billings, err := rnd(0, 10)
		if err != nil {
			return nil, i, fmt.Errorf("failed to generate random billing count: %w", err)
		}

		amountBilled, err := rnd(0, 100000)
		if err != nil {
			return nil, i, fmt.Errorf("failed to generate random amount billed: %w", err)
		}
		for j := uint64(0); j < billings; j++ {
			billing := contract_bill_report{
				id:                fmt.Sprintf("contract-bill-report-%d", billsStartID),
				contract_id:       uint64(i),
				discount_received: "Default",
				amount_billed:     amountBilled,
				timestamp:         uint64(time.Now().UnixNano()),
			}
			billsStartID++

			billTuple, err := objectToTupleString(billing)
			if err != nil {
				return nil, i, fmt.Errorf("failed to convert contract bill report object to tuple string: %w", err)
			}
			billingReports = append(billingReports, billTuple)
		}
	}

	if err := g.insertTuples(node_contract{}, contracts); err != nil {
		return nil, end, fmt.Errorf("failed to insert node contracts: %w", err)
	}

	if err := g.insertTuples(contract_resources{}, contractResources); err != nil {
		return nil, end, fmt.Errorf("failed to insert contract resources: %w", err)
	}

	if err := g.updateNodeContractResourceID(contractsStartID, end); err != nil {
		return nil, end, fmt.Errorf("failed to update node contract resources id: %w", err)
	}

	fmt.Println("node contracts generated")

	return billingReports, end, nil
}

func (g *Generator) updateNodeContractResourceID(min, max int) error {
	query := fmt.Sprintf(`UPDATE node_contract SET resources_used_id = CONCAT('contract-resources-',split_part(id, '-', -1))
		WHERE CAST(split_part(id, '-', -1) AS INTEGER) BETWEEN %d AND %d;`, min, max)
	if _, err := g.db.Exec(query); err != nil {
		return fmt.Errorf("failed to update node contract resource id: %w", err)
	}
	return nil
}

func (g *Generator) GenerateNameContracts(billsStartID, contractsStartID, contractCount int) ([]string, int, error) {
	end := contractsStartID + contractCount
	var contracts []string
	var billReports []string
	for i := contractsStartID; i < end; i++ {
		nodeID, err := rnd(1, uint64(NodeCount))
		if err != nil {
			return nil, i, fmt.Errorf("failed to generate random node id: %w", err)
		}

		state := deleted
		if g.nodeUP[nodeID] {
			if flip(contractCreatedRatio) {
				state = created
			} else if flip(0.5) {
				state = gracePeriod
			}
		}

		twinID, err := rnd(1100, 3100)
		if err != nil {
			return nil, i, fmt.Errorf("failed to generate random twin id: %w", err)
		}

		if renter, ok := g.renter[nodeID]; ok {
			twinID = renter
		}

		if _, ok := g.availableRentNodes[nodeID]; ok {
			i--
			continue
		}

		contract := name_contract{
			id:           fmt.Sprintf("name-contract-%d", i),
			twin_id:      twinID,
			contract_id:  uint64(i),
			state:        state,
			created_at:   uint64(time.Now().Unix()),
			grid_version: 3,
			name:         uuid.NewString(),
		}

		contractTuple, err := objectToTupleString(contract)
		if err != nil {
			return nil, i, fmt.Errorf("failed to convert contract object to tuple string: %w", err)
		}
		contracts = append(contracts, contractTuple)

		billings, err := rnd(0, 10)
		if err != nil {
			return nil, i, fmt.Errorf("failed to generate random billings count: %w", err)
		}
		amountBilled, err := rnd(0, 100000)
		if err != nil {
			return nil, i, fmt.Errorf("failed to generate random amount billed: %w", err)
		}

		for j := uint64(0); j < billings; j++ {
			billing := contract_bill_report{
				id:                fmt.Sprintf("contract-bill-report-%d", billsStartID),
				contract_id:       uint64(i),
				discount_received: "Default",
				amount_billed:     amountBilled,
				timestamp:         uint64(time.Now().UnixNano()),
			}
			billsStartID++

			billTuple, err := objectToTupleString(billing)
			if err != nil {
				return nil, i, fmt.Errorf("failed to convert contract bill report object to tuple string: %w", err)
			}
			billReports = append(billReports, billTuple)
		}
	}

	if err := g.insertTuples(name_contract{}, contracts); err != nil {
		return nil, end, fmt.Errorf("failed to insert name contracts: %w", err)
	}

	fmt.Println("name contracts generated")

	return billReports, end, nil
}

func (g *Generator) GenerateRentContracts(billsStart, contractStart, rentConCount int) ([]string, int, error) {
	end := contractStart + rentConCount

	var contracts []string
	var billReports []string
	for i := contractStart; i < end; i++ {
		nl, nodeID, err := popRandom(g.availableRentNodesList)
		if err != nil {
			return nil, i, fmt.Errorf("failed to select random element from the given slice: %w", err)
		}

		g.availableRentNodesList = nl
		delete(g.availableRentNodes, nodeID)
		state := deleted
		if g.nodeUP[nodeID] {
			if flip(0.9) {
				state = created
			} else if flip(0.5) {
				state = gracePeriod
			}
		}

		twinID, err := rnd(1100, 3100)
		if err != nil {
			return nil, i, fmt.Errorf("failed to generate a random twin id: %w", err)
		}

		contract := rent_contract{
			id:           fmt.Sprintf("rent-contract-%d", i),
			twin_id:      twinID,
			contract_id:  uint64(i),
			state:        state,
			created_at:   uint64(time.Now().Unix()),
			node_id:      nodeID,
			grid_version: 3,
		}

		if state != deleted {
			g.renter[nodeID] = contract.twin_id
		}

		contractTuple, err := objectToTupleString(contract)
		if err != nil {
			return nil, i, fmt.Errorf("failed to convert contract object to tuple string: %w", err)
		}
		contracts = append(contracts, contractTuple)

		billings, err := rnd(0, 10)
		if err != nil {
			return nil, i, fmt.Errorf("failed to generate billings count: %w", err)
		}

		amountBilled, err := rnd(0, 100000)
		if err != nil {
			return nil, i, fmt.Errorf("failed to generate amount billed: %w", err)
		}

		for j := uint64(0); j < billings; j++ {
			billing := contract_bill_report{
				id:                fmt.Sprintf("contract-bill-report-%d", billsStart),
				contract_id:       uint64(i),
				discount_received: "Default",
				amount_billed:     amountBilled,
				timestamp:         uint64(time.Now().UnixNano()),
			}

			billsStart++

			billTuple, err := objectToTupleString(billing)
			if err != nil {
				return nil, i, fmt.Errorf("failed to convert contract bill report object to tuple string: %w", err)
			}
			billReports = append(billReports, billTuple)

		}
	}

	if err := g.insertTuples(rent_contract{}, contracts); err != nil {
		return nil, end, fmt.Errorf("failed to insert rent contracts: %w", err)
	}

	fmt.Println("rent contracts generated")

	return billReports, end, nil
}

func (g *Generator) GeneratePublicIPs(start, size int) error {
	var publicIPs []string
	var nodeContracts []uint64

	for i := uint64(start); i < uint64(start+size); i++ {
		contract_id := uint64(0)
		if flip(usedPublicIPsRatio) {
			idx, err := rnd(0, uint64(len(g.createdNodeContracts))-1)
			if err != nil {
				return fmt.Errorf("failed to generate random index: %w", err)
			}
			contract_id = g.createdNodeContracts[idx]
		}
		ip := randomIPv4()
		farmID, err := rnd(1, uint64(FarmCount))
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

	if err := g.insertTuples(public_ip{}, publicIPs); err != nil {
		return fmt.Errorf("failed to insert public ips: %w", err)
	}

	if err := g.updateNodeContractPublicIPs(nodeContracts); err != nil {
		return fmt.Errorf("failed to update contract public ips: %w", err)
	}

	fmt.Println("public IPs generated")

	return nil
}

func (g *Generator) updateNodeContractPublicIPs(nodeContracts []uint64) error {

	if len(nodeContracts) != 0 {
		var IDs []string
		for _, contractID := range nodeContracts {
			IDs = append(IDs, fmt.Sprintf("%d", contractID))

		}

		query := "UPDATE node_contract set number_of_public_i_ps = number_of_public_i_ps + 1 WHERE contract_id IN ("
		query += strings.Join(IDs, ",") + ");"
		if _, err := g.db.Exec(query); err != nil {
			return fmt.Errorf("failed to update node contracts public ips: %w", err)
		}
	}
	return nil
}

func (g *Generator) GenerateNodeGPUs() error {
	var GPUs []string
	vendors := []string{"NVIDIA Corporation", "AMD", "Intel Corporation"}
	devices := []string{"GeForce RTX 3080", "Radeon RX 6800 XT", "Intel Iris Xe MAX"}

	for i := 0; i <= 10; i++ {
		gpuNum := len(vendors) - 1
		for j := 0; j <= gpuNum; j++ {
			g := node_gpu{
				node_twin_id: uint64(i + FarmCount + 2), // node twin ids start from 102
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

	if err := g.insertTuples(node_gpu{}, GPUs); err != nil {
		return fmt.Errorf("failed to insert node gpu: %w", err)
	}

	fmt.Println("node GPUs generated")

	return nil
}

func (g *Generator) GenerateCountries() error {
	var countriesValues []string
	index := 0
	for countryName, region := range regions {
		index++
		country := country{
			id:         fmt.Sprintf("country-%d", index),
			country_id: uint64(index),
			name:       countryName,
			code:       countriesCodes[countryName],
			region:     "unknown",
			subregion:  region,
			lat:        fmt.Sprintf("%d", 0),
			long:       fmt.Sprintf("%d", 0),
		}

		countryTuple, err := objectToTupleString(country)
		if err != nil {
			return fmt.Errorf("failed to convert country object to tuple string: %w", err)
		}
		countriesValues = append(countriesValues, countryTuple)
	}

	if err := g.insertTuples(country{}, countriesValues); err != nil {
		return fmt.Errorf("failed to insert country: %w", err)
	}
	fmt.Println("countries generated")

	return nil
}
