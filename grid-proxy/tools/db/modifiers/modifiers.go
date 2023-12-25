package modifiers

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

func (g *Generator) UpdateNodeCountry() error {
	updatesCount := 10
	query := ""

	for i := 0; i < updatesCount; i++ {
		nodeId := r.Intn(int(NodeCount)) + 1
		country := countries[r.Intn(len(countries))]
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
		nodeId := r.Intn(int(NodeCount)) + 1

		cru := 10
		hru := g.nodesHRU[uint64(nodeId)] + uint64(padding)
		mru := g.nodesMRU[uint64(nodeId)] + uint64(padding)
		sru := g.nodesSRU[uint64(nodeId)] + uint64(padding)

		query += fmt.Sprintf("UPDATE node_resources_total SET cru = %d, hru = %d, mru = %d, sru = %d WHERE node_id = 'node-%d';", cru, hru, mru, sru, nodeId)
	}

	log.Debug().Str("query", query).Msg("update node total resources")

	_, err := g.db.Exec(query)
	return err
}

func (g *Generator) UpdateContractResources() error {
	updatesCount := 10
	query := ""
	for i := 0; i < updatesCount; i++ {
		contractId := r.Intn(int(NodeContractCount)) + 1

		cru := minContractCRU
		hru := minContractHRU
		sru := minContractSRU
		mru := minContractMRU

		query += fmt.Sprintf("UPDATE contract_resources SET cru = %d, hru = %d, mru = %d, sru = %d WHERE contract_id = 'node-contract-%d';", cru, hru, mru, sru, contractId)
	}

	log.Debug().Str("query", query).Msg("update contract resources")

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
		query += fmt.Sprintf("UPDATE node_contract SET state = '%s' WHERE contract_id = %d AND state != 'Deleted';", state, contractId)
	}

	log.Debug().Str("query", query).Msg("update node contract state")

	_, err := g.db.Exec(query)
	return err
}

func (g *Generator) UpdateRentContract() error {
	updatesCount := 10
	query := ""
	states := []string{"Deleted", "GracePeriod"}

	for i := 0; i < updatesCount; i++ {
		contractId := r.Intn(int(RentContractCount)) + 1
		state := states[r.Intn(2)]
		query += fmt.Sprintf("UPDATE rent_contract SET state = '%s' WHERE contract_id = %d;", state, contractId)
	}

	log.Debug().Str("query", query).Msg("update rent contracts")

	_, err := g.db.Exec(query)
	return err
}

func (g *Generator) UpdatePublicIps() error {
	updatesCount := 10
	query := ""

	for i := 0; i < updatesCount; i++ {
		idx := r.Intn(len(g.createdNodeContracts))
		contractID := g.createdNodeContracts[idx]
		publicIPID := r.Intn(int(PublicIPCount))

		query += fmt.Sprintf("UPDATE public_ip SET contract_id = (CASE WHEN contract_id = 0 THEN %d ELSE 0 END) WHERE id = 'public-ip-%d';", contractID, publicIPID)
		query += fmt.Sprintf("UPDATE node_contract SET number_of_public_i_ps = (SELECT COUNT(id) FROM public_ip WHERE contract_id = %d) WHERE contract_id = %d;", contractID, contractID)

	}

	log.Debug().Str("query", query).Msg("update public ip contract_id")

	_, err := g.db.Exec(query)
	return err
}

// deletions
func (g *Generator) DeleteNodes(nodesCount int) error {
	// delete node contracts on this node
	// free public ips that are assigned to the deleted contracts
	// delete rent contracts on this node
	// delete node
	deleteCount := r.Intn(10) + 1
	query := ""

	for i := 0; i < deleteCount; i++ {
		nodeID := nodesCount - i

		query += fmt.Sprintf("UPDATE public_ip SET contract_id = 0 WHERE contract_id IN (SELECT contract_id FROM node_contract WHERE node_id = %d);", nodeID)
		query += fmt.Sprintf("UPDATE node_contract SET state = 'Deleted' WHERE node_id = %d;", nodeID)
		query += fmt.Sprintf("UPDATE rent_contract set state = 'Deleted' WHERE node_id = %d;", nodeID)
		query += fmt.Sprintf("DELETE FROM node_resources_total WHERE node_id = (SELECT id FROM node WHERE node_id = %d);", nodeID)
		query += fmt.Sprintf("DELETE FROM public_config WHERE node_id = (SELECT id FROM node WHERE node_id = %d);", nodeID)
		query += fmt.Sprintf("DELETE FROM node WHERE node_id = %d;", nodeID)
	}

	log.Debug().Str("query", query).Msg("delete nodes")

	_, err := g.db.Exec(query)
	return err
}

func (g *Generator) DeletePublicIps() error {
	maxDeleteCount := r.Intn(10) + 1
	query := fmt.Sprintf("DELETE FROM public_ip WHERE id in (SELECT id FROM public_ip WHERE contract_id = 0 LIMIT %d);", maxDeleteCount)

	_, err := g.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to delete public ips: %w", err)
	}

	log.Debug().Str("query", query).Msg("delete public ips")

	return nil
}