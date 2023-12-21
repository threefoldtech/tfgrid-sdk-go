package modifiers

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

func (g *Generator) UpdateNodeCountry() error {
	updatesCount := 10
	query := ""

	for i := 0; i < updatesCount; i++ {
		nodeId := r.Intn(int(g.NodeCount)) + 1
		country := g.countries[r.Intn(len(g.countries))]
		query += fmt.Sprintf("UPDATE node SET country = '%s' WHERE node_id = %d;", country, nodeId)
	}

	log.Debug().Str("query", query).Msg("update node country")

	_, err := g.db.Exec(query)
	return err
}

func (g *Generator) UpdateNodeTotalResources() error {
	updatesCount := 10
	padding := 1 * 1024 * 1024 * 1024
	query := ""
	for i := 0; i < updatesCount; i++ {
		nodeId := r.Intn(int(g.NodeCount)) + 1

		cru := 10
		hru := g.nodesHRU[uint64(nodeId)] + uint64(padding)
		mru := g.nodesMRU[uint64(nodeId)] + uint64(padding)
		sru := g.nodesSRU[uint64(nodeId)] + uint64(padding)

		query += fmt.Sprintf("UPDATE node_resources_total SET cru = %d, hru = %d, mru = %d, sru = %d WHERE node_id = 'node-%d';", cru, hru, mru, sru, nodeId)
	}

	log.Debug().Str("query", query).Msg("update node country")

	_, err := g.db.Exec(query)
	return err
}

func (g *Generator) UpdateContractResources() error {
	updatesCount := 10
	query := ""
	for i := 0; i < updatesCount; i++ {
		contractId := r.Intn(int(g.NodeContractCount)) + 1

		cru := g.minContractCRU
		hru := g.minContractHRU
		sru := g.minContractSRU
		mru := g.minContractMRU

		query += fmt.Sprintf("UPDATE contract_resources SET cru = %d, hru = %d, mru = %d, sru = %d WHERE contract_id = 'node-contract-%d';", cru, hru, mru, sru, contractId)
	}

	log.Debug().Str("query", query).Msg("update node country")

	_, err := g.db.Exec(query)
	return err
}

func (g *Generator) UpdateNodeContractState() error {
	updatesCount := 10
	query := ""
	states := []string{"Deleted", "GracePeriod"}

	for i := 0; i < updatesCount; i++ {
		contractId := g.createdNodeContracts[r.Intn(len(g.createdNodeContracts))]
		state := states[r.Intn(2)]
		query += fmt.Sprintf("UPDATE node_contract SET state = '%s' WHERE contract_id = %d;", state, contractId)
	}

	log.Debug().Str("query", query).Msg("update node country")

	_, err := g.db.Exec(query)
	return err
}

func (g *Generator) UpdateRentContract() error {
	updatesCount := 10
	query := ""
	states := []string{"Deleted", "GracePeriod"}

	for i := 0; i < updatesCount; i++ {
		contractId := r.Intn(int(g.RentContractCount)) + 1
		state := states[r.Intn(2)]
		query += fmt.Sprintf("UPDATE rent_contract SET state = '%s' WHERE contract_id = %d;", state, contractId)
	}

	log.Debug().Str("query", query).Msg("update node country")

	_, err := g.db.Exec(query)
	return err
}

func (g *Generator) UpdatePublicIps() error {

	return nil
}

// deletions
func (g *Generator) DeleteNode() error {

	return nil
}

func (g *Generator) DeletePublicIps() error {

	return nil
}
