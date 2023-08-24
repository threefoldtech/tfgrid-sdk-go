package mock

import (
	"sort"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// Contracts returns contracts with the given filters and pagination parameters
func (g *GridProxyMockClient) Contracts(filter types.ContractFilter, limit types.Limit) (res []types.Contract, totalCount int, err error) {
	res = []types.Contract{}

	if limit.Page == 0 {
		limit.Page = 1
	}
	if limit.Size == 0 {
		limit.Size = 50
	}
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

	for _, contract := range g.data.NodeContracts {
		if contract.satisfies(filter) {
			contract := types.Contract{
				ContractID: uint(contract.ContractID),
				TwinID:     uint(contract.TwinID),
				State:      contract.State,
				CreatedAt:  uint(contract.CreatedAt),
				Type:       "node",
				Details: types.NodeContractDetails{
					NodeID:            uint(contract.NodeID),
					DeploymentData:    contract.DeploymentData,
					DeploymentHash:    contract.DeploymentHash,
					NumberOfPublicIps: uint(contract.NumberOfPublicIPs),
				},
			}
			res = append(res, contract)
		}
	}

	for _, contract := range g.data.RentContracts {
		if contract.satisfies(filter) {
			contract := types.Contract{
				ContractID: uint(contract.ContractID),
				TwinID:     uint(contract.TwinID),
				State:      contract.State,
				CreatedAt:  uint(contract.CreatedAt),
				Type:       "rent",
				Details: types.RentContractDetails{
					NodeID: uint(contract.NodeID),
				},
			}
			res = append(res, contract)
		}
	}

	for _, contract := range g.data.NameContracts {
		if contract.satisfies(filter) {
			contract := types.Contract{
				ContractID: uint(contract.ContractID),
				TwinID:     uint(contract.TwinID),
				State:      contract.State,
				CreatedAt:  uint(contract.CreatedAt),
				Type:       "name",
				Details: types.NameContractDetails{
					Name: contract.Name,
				},
			}
			res = append(res, contract)
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].ContractID < res[j].ContractID
	})

	res, totalCount = getPage(res, limit)

	return
}

func (c *RentContract) satisfies(f types.ContractFilter) bool {
	if f.ContractID != nil && *f.ContractID != c.ContractID {
		return false
	}

	if f.TwinID != nil && *f.TwinID != c.TwinID {
		return false
	}

	if f.NodeID != nil && *f.NodeID != c.NodeID {
		return false
	}

	if f.Type != nil && *f.Type != "rent" {
		return false
	}

	if f.State != nil && *f.State != c.State {
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

func (c *NodeContract) satisfies(f types.ContractFilter) bool {
	if f.ContractID != nil && *f.ContractID != c.ContractID {
		return false
	}

	if f.TwinID != nil && *f.TwinID != c.TwinID {
		return false
	}

	if f.NodeID != nil && *f.NodeID != c.NodeID {
		return false
	}

	if f.Type != nil && *f.Type != "node" {
		return false
	}

	if f.State != nil && *f.State != c.State {
		return false
	}

	if f.Name != nil && *f.Name != "" {
		return false
	}

	if f.NumberOfPublicIps != nil && *f.NumberOfPublicIps > c.NumberOfPublicIPs {
		return false
	}

	if f.DeploymentData != nil && *f.DeploymentData != c.DeploymentData {
		return false
	}

	if f.DeploymentHash != nil && *f.DeploymentHash != c.DeploymentHash {
		return false
	}

	return true
}

func (c *NameContract) satisfies(f types.ContractFilter) bool {
	if f.ContractID != nil && *f.ContractID != c.ContractID {
		return false
	}

	if f.TwinID != nil && *f.TwinID != c.TwinID {
		return false
	}

	if f.NodeID != nil && *f.NodeID != 0 {
		return false
	}

	if f.Type != nil && *f.Type != "name" {
		return false
	}

	if f.State != nil && *f.State != c.State {
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
