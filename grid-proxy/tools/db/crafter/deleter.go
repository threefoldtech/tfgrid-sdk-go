package crafter

import (
	"fmt"
)

// deletions
func (g *Crafter) DeleteNodes() error {
	// delete node contracts on this node
	// free public ips that are assigned to the deleted contracts
	// delete rent contracts on this node
	// delete node
	deleteCount := r.Intn(10) + 1
	query := ""

	for i := 0; i < deleteCount; i++ {
		nodeID := int(g.NodeCount) - i

		query += fmt.Sprintf("UPDATE public_ip SET contract_id = 0 WHERE contract_id IN (SELECT contract_id FROM node_contract WHERE node_id = %d);", nodeID)
		query += fmt.Sprintf("UPDATE node_contract SET state = 'Deleted' WHERE node_id = %d;", nodeID)
		query += fmt.Sprintf("UPDATE rent_contract set state = 'Deleted' WHERE node_id = %d;", nodeID)
		query += fmt.Sprintf("DELETE FROM node_resources_total WHERE node_id = (SELECT id FROM node WHERE node_id = %d);", nodeID)
		query += fmt.Sprintf("DELETE FROM public_config WHERE node_id = (SELECT id FROM node WHERE node_id = %d);", nodeID)
		query += fmt.Sprintf("DELETE FROM node WHERE node_id = %d;", nodeID)
		query += fmt.Sprintf("DELETE FROM health_report WHERE node_twin_id = (SELECT twin_id FROM node WHERE node_id = %d);", nodeID)
	}

	fmt.Println("nodes deleted")

	_, err := g.db.Exec(query)
	return err
}

func (g *Crafter) DeletePublicIps() error {
	maxDeleteCount := r.Intn(10) + 1
	query := fmt.Sprintf("DELETE FROM public_ip WHERE id in (SELECT id FROM public_ip WHERE contract_id = 0 LIMIT %d);", maxDeleteCount)

	_, err := g.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to delete public ips: %w", err)
	}

	fmt.Println("public ips deleted")

	return nil
}
