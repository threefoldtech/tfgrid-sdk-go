package mock

import (
	"sort"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// Contracts returns contracts with the given filters and pagination parameters
func (g *GridProxyMockClient) Contracts(filter types.ContractFilter, limit types.Limit) ([]types.Contract, int, error) {
	res := []types.Contract{}

	billings := g.getContractBillings()

	nodeContracts := g.filterNodeContracts(filter, billings)
	res = append(res, nodeContracts...)

	rentContracts := g.filterRentContracts(filter, billings)
	res = append(res, rentContracts...)

	nameContracts := g.filterNameContracts(filter, billings)
	res = append(res, nameContracts...)

	sort.Slice(res, func(i, j int) bool {
		return res[i].ContractID < res[j].ContractID
	})

	res, count := getPage(res, limit)

	return res, count, nil
}

func (g *GridProxyMockClient) getContractBillings() map[uint64][]types.ContractBilling {
	billings := make(map[uint64][]types.ContractBilling)
	for contractID, contractBillings := range g.data.Billings {
		for _, billing := range contractBillings {
			billings[contractID] = append(billings[contractID], types.ContractBilling{
				AmountBilled:     billing.AmountBilled,
				DiscountReceived: billing.DiscountReceived,
				Timestamp:        billing.Timestamp,
			})
		}

		sort.Slice(billings[contractID], func(i, j int) bool {
			return billings[contractID][i].Timestamp < billings[contractID][j].Timestamp
		})
	}

	return billings
}

func (g *GridProxyMockClient) filterNodeContracts(filter types.ContractFilter, billings map[uint64][]types.ContractBilling) []types.Contract {
	contracts := []types.Contract{}
	for _, contract := range g.data.NodeContracts {
		if contract.satisfies(filter) {
			contract := types.Contract{
				ContractID: contract.ContractID,
				TwinID:     contract.TwinID,
				State:      contract.State,
				CreatedAt:  contract.CreatedAt,
				Type:       "node",
				Details: types.NodeContractDetails{
					NodeID:            contract.NodeID,
					DeploymentData:    contract.DeploymentData,
					DeploymentHash:    contract.DeploymentHash,
					NumberOfPublicIps: contract.NumberOfPublicIPs,
				},
				Billing: append([]types.ContractBilling{}, billings[contract.ContractID]...),
			}
			contracts = append(contracts, contract)
		}
	}

	return contracts
}

func (g *GridProxyMockClient) filterRentContracts(filter types.ContractFilter, billings map[uint64][]types.ContractBilling) []types.Contract {
	contracts := []types.Contract{}
	for _, contract := range g.data.RentContracts {
		if contract.satisfies(filter) {
			contract := types.Contract{
				ContractID: contract.ContractID,
				TwinID:     contract.TwinID,
				State:      contract.State,
				CreatedAt:  contract.CreatedAt,
				Type:       "rent",
				Details: types.RentContractDetails{
					NodeID: contract.NodeID,
				},
				Billing: append([]types.ContractBilling{}, billings[contract.ContractID]...),
			}
			contracts = append(contracts, contract)
		}
	}

	return contracts
}

func (g *GridProxyMockClient) filterNameContracts(filter types.ContractFilter, billings map[uint64][]types.ContractBilling) []types.Contract {
	contracts := []types.Contract{}
	for _, contract := range g.data.NameContracts {
		if contract.satisfies(filter) {
			contract := types.Contract{
				ContractID: contract.ContractID,
				TwinID:     contract.TwinID,
				State:      contract.State,
				CreatedAt:  contract.CreatedAt,
				Type:       "name",
				Details: types.NameContractDetails{
					Name: contract.Name,
				},
				Billing: append([]types.ContractBilling{}, billings[contract.ContractID]...),
			}
			contracts = append(contracts, contract)
		}
	}

	return contracts
}

func (c *DBRentContract) satisfies(f types.ContractFilter) bool {
	if f.ContractID != nil && c.ContractID != *f.ContractID {
		return false
	}

	if f.TwinID != nil && c.TwinID != *f.TwinID {
		return false
	}

	if f.NodeID != nil && c.NodeID != *f.NodeID {
		return false
	}

	if f.Type != nil && *f.Type != "rent" {
		return false
	}

	if f.State != nil && c.State != *f.State {
		return false
	}

	if f.Name != nil && *f.Name != "" {
		return false
	}

	if f.NumberOfPublicIps != nil && *f.NumberOfPublicIps != 0 {
		return false
	}

	if f.DeploymentData != nil && *f.DeploymentData != "" {
		return false
	}

	if f.DeploymentHash != nil && *f.DeploymentHash != "" {
		return false
	}

	return true
}

func (c *DBNameContract) satisfies(f types.ContractFilter) bool {
	if f.ContractID != nil && c.ContractID != *f.ContractID {
		return false
	}

	if f.TwinID != nil && c.TwinID != *f.TwinID {
		return false
	}

	if f.NodeID != nil {
		return false
	}

	if f.Type != nil && *f.Type != "name" {
		return false
	}

	if f.State != nil && c.State != *f.State {
		return false
	}

	if f.Name != nil && *f.Name != c.Name {
		return false
	}

	if f.NumberOfPublicIps != nil && *f.NumberOfPublicIps != 0 {
		return false
	}

	if f.DeploymentData != nil && *f.DeploymentData != "" {
		return false
	}

	if f.DeploymentHash != nil && *f.DeploymentHash != "" {
		return false
	}

	return true
}

func (c *DBNodeContract) satisfies(f types.ContractFilter) bool {
	if f.ContractID != nil && c.ContractID != *f.ContractID {
		return false
	}

	if f.TwinID != nil && c.TwinID != *f.TwinID {
		return false
	}

	if f.NodeID != nil && c.NodeID != *f.NodeID {
		return false
	}

	if f.Type != nil && *f.Type != "node" {
		return false
	}

	if f.State != nil && c.State != *f.State {
		return false
	}

	if f.Name != nil && *f.Name != "" {
		return false
	}

	if f.NumberOfPublicIps != nil && c.NumberOfPublicIPs < *f.NumberOfPublicIps { // TODO: fix
		return false
	}

	if f.DeploymentData != nil && c.DeploymentData != *f.DeploymentData {
		return false
	}

	if f.DeploymentHash != nil && c.DeploymentHash != *f.DeploymentHash {
		return false
	}

	return true
}
