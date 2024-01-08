package crafter

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

func (c *Crafter) GenerateTwins() error {
	start := c.TwinStart
	end := c.TwinCount + c.TwinStart

	var twins []string

	for i := uint64(start); i < uint64(end); i++ {
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

	if err := c.insertTuples(twin{}, twins); err != nil {
		return fmt.Errorf("failed to insert twins: %w", err)
	}
	fmt.Printf("twins generated [%d : %d[\n", start, end)

	return nil
}

func (c *Crafter) GenerateFarms() error {
	start := c.FarmStart
	end := c.FarmCount + c.FarmStart
	farmTwinsStart := c.TwinStart + c.FarmStart

	var farms []string
	for i := uint64(start); i < uint64(end); i++ {
		farm := farm{
			id:                fmt.Sprintf("farm-%d", i),
			farm_id:           i,
			name:              fmt.Sprintf("farm-name-%d", i),
			certification:     "Diy",
			dedicated_farm:    flip(.1),
			twin_id:           uint64(farmTwinsStart) + i,
			pricing_policy_id: 1,
			grid_version:      3,
			stellar_address:   "",
		}

		if farm.dedicated_farm {
			c.dedicatedFarms[farm.farm_id] = struct{}{}
		}

		farmTuple, err := objectToTupleString(farm)
		if err != nil {
			return fmt.Errorf("failed to convert farm object to tuple string: %w", err)
		}
		farms = append(farms, farmTuple)
	}

	if err := c.insertTuples(farm{}, farms); err != nil {
		return fmt.Errorf("failed to insert farms: %w", err)
	}
	fmt.Printf("farms generated [%d : %d[\n", start, end)

	return nil
}

func (c *Crafter) GenerateNodes() error {
	start := c.NodeStart
	end := c.NodeStart + c.NodeCount
	nodeTwinsStart := c.TwinStart + (c.FarmStart + c.FarmCount)

	powerState := []string{"Up", "Down"}
	var locations []string
	var healthReports []string
	var nodes []string
	var totalResources []string
	var publicConfigs []string
	for i := uint64(start); i < uint64(end); i++ {
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

		c.nodesMRU[i] = mru - max(2*uint64(gridtypes.Gigabyte), mru/10)
		c.nodesSRU[i] = sru - 100*uint64(gridtypes.Gigabyte)
		c.nodesHRU[i] = hru
		c.nodeUP[i] = up

		// location latitude and longitue needs to be castable to decimal
		// if not, the convert_to_decimal function will raise a notice
		// reporting the incident, which downgrades performance
		locationId := fmt.Sprintf("location-%d", uint64(start)+i)
		location := location{
			id:        locationId,
			longitude: fmt.Sprintf("%d", i),
			latitude:  fmt.Sprintf("%d", i),
		}

		countryIndex := r.Intn(len(countries))
		cityIndex := r.Intn(len(cities[countries[countryIndex]]))
		farmId := r.Intn(int(c.FarmCount)) + int(c.FarmStart)
		node := node{
			id:                fmt.Sprintf("node-%d", i),
			location_id:       locationId,
			node_id:           i,
			farm_id:           uint64(farmId),
			twin_id:           uint64(nodeTwinsStart) + i,
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
			extra_fee:         0,
			dedicated:         false,
		}

		if flip(.3) {
			node.power = &nodePower{
				State:  powerState[r.Intn(len(powerState))],
				Target: powerState[r.Intn(len(powerState))],
			}
		}

		total_resources := node_resources_total{
			id:      fmt.Sprintf("total-resources-%d", i),
			hru:     hru,
			sru:     sru,
			cru:     cru,
			mru:     mru,
			node_id: fmt.Sprintf("node-%d", i),
		}

		health := true
		if flip(.5) {
			health = false
		}
		healthReport := health_report{
			node_twin_id: uint64(nodeTwinsStart) + i,
			healthy:      health,
		}

		if _, ok := c.dedicatedFarms[node.farm_id]; ok {
			c.availableRentNodes[i] = struct{}{}
			c.availableRentNodesList = append(c.availableRentNodesList, i)
		}

		locationTuple, err := objectToTupleString(location)
		if err != nil {
			return fmt.Errorf("failed to convert location object to tuple string: %w", err)
		}
		locations = append(locations, locationTuple)

		healthTuple, err := objectToTupleString(healthReport)
		if err != nil {
			return fmt.Errorf("failed to convert health report to tuple string: %w", err)
		}
		healthReports = append(healthReports, healthTuple)

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

	if err := c.insertTuples(location{}, locations); err != nil {
		return fmt.Errorf("failed to insert locations: %w", err)
	}

	if err := c.insertTuples(health_report{}, healthReports); err != nil {
		return fmt.Errorf("failed to insert health reports: %w", err)
	}

	if err := c.insertTuples(node{}, nodes); err != nil {
		return fmt.Errorf("failed to insert nodes: %w", err)
	}

	if err := c.insertTuples(node_resources_total{}, totalResources); err != nil {
		return fmt.Errorf("failed to insert node resources total: %w", err)
	}

	if err := c.insertTuples(public_config{}, publicConfigs); err != nil {
		return fmt.Errorf("failed to insert public configs: %w", err)
	}
	fmt.Printf("nodes generated [%d : %d[\n", start, end)

	return nil
}

func (c *Crafter) GenerateContracts() error {
	var billReports []string

	rentContractsBillReports, nodeContractIDStart, err := c.GenerateRentContracts(int(c.BillStart), int(c.ContractStart), int(c.RentContractCount))
	if err != nil {
		return fmt.Errorf("failed to generate rent contracts: %w", err)
	}
	billReports = append(billReports, rentContractsBillReports...)

	nodeContractsBillReports, nameContractIDStart, err := c.generateNodeContracts(len(billReports)+int(c.BillStart), nodeContractIDStart, int(c.NodeContractCount), int(c.NodeStart), int(c.NodeCount))
	if err != nil {
		return fmt.Errorf("failed to generate node contracts: %w", err)
	}
	billReports = append(billReports, nodeContractsBillReports...)

	nameContractsBillReports, _, err := c.GenerateNameContracts(len(billReports)+int(c.BillStart), nameContractIDStart, int(c.NameContractCount))
	if err != nil {
		return fmt.Errorf("failed to generate name contracts: %w", err)
	}
	billReports = append(billReports, nameContractsBillReports...)

	if err := c.insertTuples(contract_bill_report{}, billReports); err != nil {
		return fmt.Errorf("failed to generate contract bill reports: %w", err)
	}
	return nil
}

func (c *Crafter) generateNodeContracts(billsStartID, contractsStartID, contractCount, nodeStart, nodeSize int) ([]string, int, error) {
	end := contractsStartID + contractCount
	start := contractsStartID

	var contracts []string
	var contractResources []string
	var billingReports []string
	toDelete := ""
	toGracePeriod := ""
	contractToResource := map[uint64]string{}
	for i := start; i < end; i++ {
		nodeID := uint64(r.Intn(nodeSize) + nodeStart)
		state := deleted
		if c.nodeUP[nodeID] {
			if flip(0.9) {
				state = created
			} else if flip(0.4) {
				state = gracePeriod
			}
		}

		if state == deleted {
			if len(toDelete) != 0 {
				toDelete += ", "
			}
			toDelete += fmt.Sprint(i)
		}

		if state == gracePeriod {
			if len(toGracePeriod) != 0 {
				toGracePeriod += ", "
			}
			toGracePeriod += fmt.Sprint(i)
		}

		if state != deleted && (minContractHRU > c.nodesHRU[nodeID] || minContractMRU > c.nodesMRU[nodeID] || minContractSRU > c.nodesSRU[nodeID]) {
			i--
			continue
		}

		twinID, err := rnd(1100, 3100)
		if err != nil {
			return nil, i, fmt.Errorf("failed to generate random twin id: %w", err)
		}

		if renter, ok := c.renter[nodeID]; ok {
			twinID = renter
		}

		if _, ok := c.availableRentNodes[nodeID]; ok {
			i--
			continue
		}

		contract := node_contract{
			id:                    fmt.Sprintf("node-contract-%d", i),
			twin_id:               twinID,
			contract_id:           uint64(i),
			state:                 created,
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

		hru, err := rnd(minContractHRU, min(maxContractHRU, c.nodesHRU[nodeID]))
		if err != nil {
			return nil, i, fmt.Errorf("failed to generate random hru: %w", err)
		}

		sru, err := rnd(minContractSRU, min(maxContractSRU, c.nodesSRU[nodeID]))
		if err != nil {
			return nil, i, fmt.Errorf("failed to generate random sru: %w", err)
		}

		mru, err := rnd(minContractMRU, min(maxContractMRU, c.nodesMRU[nodeID]))
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
		contractToResource[contract.contract_id] = contract_resources.id

		if state != deleted {
			c.nodesHRU[nodeID] -= hru
			c.nodesSRU[nodeID] -= sru
			c.nodesMRU[nodeID] -= mru
			c.createdNodeContracts = append(c.createdNodeContracts, uint64(i))
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

	if err := c.insertTuples(node_contract{}, contracts); err != nil {
		return nil, end, fmt.Errorf("failed to insert node contracts: %w", err)
	}

	if err := c.insertTuples(contract_resources{}, contractResources); err != nil {
		return nil, end, fmt.Errorf("failed to insert contract resources: %w", err)
	}

	if err := c.updateNodeContractResourceID(contractToResource); err != nil {
		return nil, end, fmt.Errorf("failed to update node contract resources id: %w", err)
	}

	if len(toDelete) > 0 {
		if _, err := c.db.Exec(fmt.Sprintf("UPDATE node_contract SET state = '%s' WHERE contract_id IN (%s)", deleted, toDelete)); err != nil {
			return nil, 0, fmt.Errorf("failed to update node_contract state to deleted: %w", err)
		}
	}

	if len(toGracePeriod) > 0 {
		if _, err := c.db.Exec(fmt.Sprintf("UPDATE node_contract SET state = '%s' WHERE contract_id IN (%s)", gracePeriod, toGracePeriod)); err != nil {
			return nil, 0, fmt.Errorf("failed to update node_contract state to grace period: %w", err)
		}
	}

	fmt.Printf("node contracts generated [%d : %d[\n", start, end)

	return billingReports, end, nil
}

func (c *Crafter) updateNodeContractResourceID(contractToResource map[uint64]string) error {
	query := ""
	for contractID, ResourceID := range contractToResource {
		query += fmt.Sprintf("UPDATE node_contract SET resources_used_id = '%s' WHERE contract_id = %d;", ResourceID, contractID)
	}

	if _, err := c.db.Exec(query); err != nil {
		return fmt.Errorf("failed to update node contract resource id: %w", err)
	}
	return nil
}

func (c *Crafter) GenerateNameContracts(billsStartID, contractsStartID, contractCount int) ([]string, int, error) {
	end := contractsStartID + contractCount
	start := contractsStartID
	var contracts []string
	var billReports []string
	toDelete := ""
	toGracePeriod := ""

	for i := start; i < end; i++ {
		// WATCH:
		nodeID, err := rnd(1, uint64(c.NodeCount))
		if err != nil {
			return nil, i, fmt.Errorf("failed to generate random node id: %w", err)
		}

		state := deleted
		if c.nodeUP[nodeID] {
			if flip(0.9) {
				state = created
			} else if flip(0.5) {
				state = gracePeriod
			}
		}

		if state == deleted {
			if len(toDelete) != 0 {
				toDelete += ", "
			}
			toDelete += fmt.Sprint(i)
		}

		if state == gracePeriod {
			if len(toGracePeriod) != 0 {
				toGracePeriod += ", "
			}
			toGracePeriod += fmt.Sprint(i)
		}

		twinID, err := rnd(1100, 3100)
		if err != nil {
			return nil, i, fmt.Errorf("failed to generate random twin id: %w", err)
		}

		if renter, ok := c.renter[nodeID]; ok {
			twinID = renter
		}

		if _, ok := c.availableRentNodes[nodeID]; ok {
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

	if err := c.insertTuples(name_contract{}, contracts); err != nil {
		return nil, end, fmt.Errorf("failed to insert name contracts: %w", err)
	}

	if len(toDelete) > 0 {
		if _, err := c.db.Exec(fmt.Sprintf("UPDATE rent_contract SET state = '%s' WHERE contract_id IN (%s)", deleted, toDelete)); err != nil {
			return nil, 0, fmt.Errorf("failed to update rent_contract state to deleted: %w", err)
		}
	}

	if len(toGracePeriod) > 0 {
		if _, err := c.db.Exec(fmt.Sprintf("UPDATE rent_contract SET state = '%s' WHERE contract_id IN (%s)", gracePeriod, toGracePeriod)); err != nil {
			return nil, 0, fmt.Errorf("failed to update rent_contract state to grace period: %w", err)
		}
	}

	fmt.Printf("name contracts generated  [%d : %d[\n", start, end)

	return billReports, end, nil
}

func (c *Crafter) GenerateRentContracts(billsStart, contractStart, rentConCount int) ([]string, int, error) {
	end := contractStart + rentConCount
	start := contractStart

	var contracts []string
	var billReports []string
	toDelete := ""
	toGracePeriod := ""
	for i := start; i < end; i++ {
		nl, nodeID, err := popRandom(c.availableRentNodesList)
		if err != nil {
			return nil, i, fmt.Errorf("failed to select random element from the given slice: %w", err)
		}

		c.availableRentNodesList = nl
		delete(c.availableRentNodes, nodeID)
		state := deleted
		if c.nodeUP[nodeID] {
			if flip(0.9) {
				state = created
			} else if flip(0.5) {
				state = gracePeriod
			}
		}

		if state == deleted {
			if len(toDelete) != 0 {
				toDelete += ", "
			}
			toDelete += fmt.Sprint(i)
		}

		if state == gracePeriod {
			if len(toGracePeriod) != 0 {
				toGracePeriod += ", "
			}
			toGracePeriod += fmt.Sprint(i)
		}

		twinID, err := rnd(1100, 3100)
		if err != nil {
			return nil, i, fmt.Errorf("failed to generate a random twin id: %w", err)
		}

		contract := rent_contract{
			id:           fmt.Sprintf("rent-contract-%d", i),
			twin_id:      twinID,
			contract_id:  uint64(i),
			state:        created,
			created_at:   uint64(time.Now().Unix()),
			node_id:      nodeID,
			grid_version: 3,
		}

		if state != deleted {
			c.renter[nodeID] = contract.twin_id
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

	if err := c.insertTuples(rent_contract{}, contracts); err != nil {
		return nil, end, fmt.Errorf("failed to insert rent contracts: %w", err)
	}

	if len(toDelete) > 0 {
		if _, err := c.db.Exec(fmt.Sprintf("UPDATE rent_contract SET state = '%s' WHERE contract_id IN (%s)", deleted, toDelete)); err != nil {
			return nil, 0, fmt.Errorf("failed to update rent_contract state to deleted: %w", err)
		}
	}

	if len(toGracePeriod) > 0 {
		if _, err := c.db.Exec(fmt.Sprintf("UPDATE rent_contract SET state = '%s' WHERE contract_id IN (%s)", gracePeriod, toGracePeriod)); err != nil {
			return nil, 0, fmt.Errorf("failed to update rent_contract state to grace period: %w", err)
		}
	}

	fmt.Printf("rent contracts generated [%d : %d[\n", start, end)

	return billReports, end, nil
}

func (c *Crafter) GeneratePublicIPs() error {
	start := c.PublicIPStart
	end := c.PublicIPCount + c.PublicIPStart

	var publicIPs []string
	var nodeContracts []uint64
	reservedIPs := map[string]uint64{}
	for i := uint64(start); i < uint64(end); i++ {
		contract_id := uint64(0)
		if flip(usedPublicIPsRatio) {
			idx, err := rnd(0, uint64(len(c.createdNodeContracts))-1)
			if err != nil {
				return fmt.Errorf("failed to generate random index: %w", err)
			}
			contract_id = c.createdNodeContracts[idx]
		}

		ip := randomIPv4()

		farmID := r.Int63n(int64(c.FarmCount)) + int64(c.FarmStart)

		public_ip := public_ip{
			id:          fmt.Sprintf("public-ip-%d", i),
			gateway:     ip.String(),
			ip:          IPv4Subnet(ip).String(),
			contract_id: 0,
			farm_id:     fmt.Sprintf("farm-%d", farmID),
		}

		if contract_id != 0 {
			reservedIPs[public_ip.id] = contract_id
		}

		publicIpTuple, err := objectToTupleString(public_ip)
		if err != nil {
			return fmt.Errorf("failed to convert public ip object to tuple string: %w", err)
		}
		publicIPs = append(publicIPs, publicIpTuple)
		nodeContracts = append(nodeContracts, contract_id)
	}

	if err := c.insertTuples(public_ip{}, publicIPs); err != nil {
		return fmt.Errorf("failed to insert public ips: %w", err)
	}

	if err := c.updateNodeContractPublicIPs(nodeContracts); err != nil {
		return fmt.Errorf("failed to update contract public ips: %w", err)
	}

	for id, contractID := range reservedIPs {
		if _, err := c.db.Exec(fmt.Sprintf("UPDATE public_ip SET contract_id = %d WHERE id = '%s'", contractID, id)); err != nil {
			return fmt.Errorf("failed to reserve ip %s: %w", id, err)
		}
	}

	fmt.Printf("public IPs generated  [%d : %d[\n", start, end)

	return nil
}

func (c *Crafter) updateNodeContractPublicIPs(nodeContracts []uint64) error {

	if len(nodeContracts) != 0 {
		var IDs []string
		for _, contractID := range nodeContracts {
			IDs = append(IDs, fmt.Sprintf("%d", contractID))

		}

		query := "UPDATE node_contract set number_of_public_i_ps = number_of_public_i_ps + 1 WHERE contract_id IN ("
		query += strings.Join(IDs, ",") + ");"
		if _, err := c.db.Exec(query); err != nil {
			return fmt.Errorf("failed to update node contracts public ips: %w", err)
		}
	}
	return nil
}

func (c *Crafter) GenerateNodeGPUs() error {
	var GPUs []string
	vendors := []string{"NVIDIA Corporation", "AMD", "Intel Corporation"}
	devices := []string{"GeForce RTX 3080", "Radeon RX 6800 XT", "Intel Iris Xe MAX"}

	nodeTwinsStart := c.TwinStart + (c.FarmStart + c.FarmCount)
	nodeWithGpuNum := 10

	for i := 1; i <= nodeWithGpuNum; i++ {
		gpuNum := len(vendors) - 1
		for j := 0; j <= gpuNum; j++ {
			g := node_gpu{
				// WATCH
				node_twin_id: uint64(nodeTwinsStart + uint(i)),
				vendor:       vendors[j],
				device:       devices[j],
				contract:     i % 2,
				id:           fmt.Sprintf("node-gpu-%d-%d", nodeTwinsStart+uint(i), j),
			}
			gpuTuple, err := objectToTupleString(g)
			if err != nil {
				return fmt.Errorf("failed to convert gpu object to tuple string: %w", err)
			}
			GPUs = append(GPUs, gpuTuple)
		}
	}

	if err := c.insertTuples(node_gpu{}, GPUs); err != nil {
		return fmt.Errorf("failed to insert node gpu: %w", err)
	}

	fmt.Println("node GPUs generated")

	return nil
}

func (c *Crafter) GenerateCountries() error {
	var countriesValues []string

	// depends on nodeStart to not duplicate the value of country.id
	start := c.NodeStart

	index := start
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

	if err := c.insertTuples(country{}, countriesValues); err != nil {
		return fmt.Errorf("failed to insert country: %w", err)
	}
	fmt.Println("countries generated")

	return nil
}
