package db

import (
	"math/rand"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// GetContracts returns contracts filtered and paginated
func (d *PostgresDatabase) GetContracts(filter types.ContractFilter, limit types.Limit) ([]DBContract, uint, error) {
	q := d.gormDB.
		Table(`(SELECT contract_id, twin_id, state, created_at, ''AS name, node_id, deployment_data, deployment_hash, number_of_public_i_ps, 'node' AS type
	FROM node_contract 
	UNION 
	SELECT contract_id, twin_id, state, created_at, '' AS name, node_id, '', '', 0, 'rent' AS type
	FROM rent_contract 
	UNION 
	SELECT contract_id, twin_id, state, created_at, name, 0, '', '', 0, 'name' AS type
	FROM name_contract) contracts`).
		Select(
			"contracts.contract_id",
			"twin_id",
			"state",
			"created_at",
			"name",
			"node_id",
			"deployment_data",
			"deployment_hash",
			"number_of_public_i_ps as number_of_public_ips",
			"type",
			"COALESCE(contract_billing.billings, '[]') as contract_billings",
		).
		Joins(
			`LEFT JOIN (
				SELECT 
					contract_bill_report.contract_id,
					COALESCE(json_agg(json_build_object('amount_billed', amount_billed, 'discount_received', discount_received, 'timestamp', timestamp)), '[]') as billings
				FROM
					contract_bill_report
				GROUP BY contract_id
			) contract_billing
			ON contracts.contract_id = contract_billing.contract_id`,
		)
	if filter.Type != nil {
		q = q.Where("type = ?", *filter.Type)
	}
	if filter.State != nil {
		q = q.Where("state ILIKE ?", *filter.State)
	}
	if filter.TwinID != nil {
		q = q.Where("twin_id = ?", *filter.TwinID)
	}
	if filter.ContractID != nil {
		q = q.Where("contracts.contract_id = ?", *filter.ContractID)
	}
	if filter.NodeID != nil {
		q = q.Where("node_id = ?", *filter.NodeID)
	}
	if filter.NumberOfPublicIps != nil {
		q = q.Where("number_of_public_i_ps >= ?", *filter.NumberOfPublicIps)
	}
	if filter.Name != nil {
		q = q.Where("name = ?", *filter.Name)
	}
	if filter.DeploymentData != nil {
		q = q.Where("deployment_data = ?", *filter.DeploymentData)
	}
	if filter.DeploymentHash != nil {
		q = q.Where("deployment_hash = ?", *filter.DeploymentHash)
	}
	var count int64
	if limit.Randomize || limit.RetCount {
		if res := q.Count(&count); res.Error != nil {
			return nil, 0, errors.Wrap(res.Error, "couldn't get contract count")
		}
	}
	if limit.Randomize {
		q = q.Limit(int(limit.Size)).
			Offset(int(rand.Intn(int(count)) - int(limit.Size)))
	} else {
		q = q.Limit(int(limit.Size)).
			Offset(int(limit.Page-1) * int(limit.Size)).
			Order("contract_id")
	}
	var contracts []DBContract
	if res := q.Scan(&contracts); res.Error != nil {
		return contracts, uint(count), errors.Wrap(res.Error, "failed to scan returned contracts from database")
	}
	return contracts, uint(count), nil
}
