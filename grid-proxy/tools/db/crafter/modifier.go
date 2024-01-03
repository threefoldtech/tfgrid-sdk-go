package crafter

import (
	"fmt"
)

func (g *Crafter) UpdateNodeCountry() error {
	updatesCount := 10
	query := ""

	for i := 0; i < updatesCount; i++ {
		// WATCH
		nodeId := r.Intn(int(g.NodeCount)) + 1
		country := countries[r.Intn(len(countries))]
		query += fmt.Sprintf("UPDATE node SET country = '%s' WHERE node_id = %d;", country, nodeId)
	}

	_, err := g.db.Exec(query)
	fmt.Println("node country updated")
	return err
}

func (g *Crafter) UpdateNodeTotalResources() error {
	updatesCount := 10
	scaling := 1 * 1024 * 1024 * 1024
	query := ""
	for i := 0; i < updatesCount; i++ {
		// WATCH
		nodeId := r.Intn(int(g.NodeCount)) + 1

		cru := 10
		hru := g.nodesHRU[uint64(nodeId)] + uint64(scaling)
		mru := g.nodesMRU[uint64(nodeId)] + uint64(scaling)
		sru := g.nodesSRU[uint64(nodeId)] + uint64(scaling)

		query += fmt.Sprintf("UPDATE node_resources_total SET cru = %d, hru = %d, mru = %d, sru = %d WHERE node_id = 'node-%d';", cru, hru, mru, sru, nodeId)
	}

	_, err := g.db.Exec(query)
	fmt.Println("node total resources updated")
	return err
}

func (g *Crafter) UpdateContractResources() error {
	updatesCount := 10
	query := ""
	for i := 0; i < updatesCount; i++ {
		// WATCH
		contractId := r.Intn(int(g.NodeContractCount)) + 1

		cru := minContractCRU
		hru := minContractHRU
		sru := minContractSRU
		mru := minContractMRU

		query += fmt.Sprintf("UPDATE contract_resources SET cru = %d, hru = %d, mru = %d, sru = %d WHERE contract_id = 'node-contract-%d';", cru, hru, mru, sru, contractId)
	}

	_, err := g.db.Exec(query)
	fmt.Println("contract resources updated")
	return err
}

func (g *Crafter) UpdateNodeContractState() error {
	updatesCount := 10
	query := ""
	states := []string{"Deleted", "GracePeriod"}

	for i := 0; i < updatesCount; i++ {
		contractId := g.createdNodeContracts[r.Intn(len(g.createdNodeContracts))]
		state := states[r.Intn(2)]
		query += fmt.Sprintf("UPDATE node_contract SET state = '%s' WHERE contract_id = %d AND state != 'Deleted';", state, contractId)
	}

	_, err := g.db.Exec(query)
	fmt.Println("node contract state updated")
	return err
}

func (g *Crafter) UpdateRentContract() error {
	updatesCount := 10
	query := ""
	states := []string{"Deleted", "GracePeriod"}

	for i := 0; i < updatesCount; i++ {
		// WATCH
		contractId := r.Intn(int(g.RentContractCount)) + 1
		state := states[r.Intn(2)]
		query += fmt.Sprintf("UPDATE rent_contract SET state = '%s' WHERE contract_id = %d;", state, contractId)
	}

	_, err := g.db.Exec(query)
	fmt.Println("rent contracts updated")
	return err
}

func (g *Crafter) UpdatePublicIps() error {
	updatesCount := 10
	query := ""

	for i := 0; i < updatesCount; i++ {
		idx := r.Intn(len(g.createdNodeContracts))
		contractID := g.createdNodeContracts[idx]
		// WATCH
		publicIPID := r.Intn(int(g.PublicIPCount))

		query += fmt.Sprintf("UPDATE public_ip SET contract_id = (CASE WHEN contract_id = 0 THEN %d ELSE 0 END) WHERE id = 'public-ip-%d';", contractID, publicIPID)
		query += fmt.Sprintf("UPDATE node_contract SET number_of_public_i_ps = (SELECT COUNT(id) FROM public_ip WHERE contract_id = %d) WHERE contract_id = %d;", contractID, contractID)

	}

	_, err := g.db.Exec(query)
	fmt.Println("public ip contract_id update")
	return err
}
