package mock

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// Contracts returns contracts with the given filters and pagination parameters
func (g *GridProxyMockClient) Contracts(ctx context.Context, filter types.ContractFilter, limit types.Limit) (res []types.Contract, totalCount int, err error) {
	res = []types.Contract{}

	if limit.Page == 0 {
		limit.Page = 1
	}
	if limit.Size == 0 {
		limit.Size = 50
	}

	for _, contract := range g.data.NodeContracts {
		if contract.satisfies(filter, g.data.Nodes, g.data.Farms) {
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
					FarmName:          g.data.Farms[g.data.Nodes[contract.NodeID].FarmID].Name,
					FarmId:            g.data.Nodes[contract.NodeID].FarmID,
				},
			}
			res = append(res, contract)
		}
	}

	for _, contract := range g.data.RentContracts {
		if contract.satisfies(filter, g.data.Nodes, g.data.Farms) {
			contract := types.Contract{
				ContractID: uint(contract.ContractID),
				TwinID:     uint(contract.TwinID),
				State:      contract.State,
				CreatedAt:  uint(contract.CreatedAt),
				Type:       "rent",
				Details: types.RentContractDetails{
					NodeID:   uint(contract.NodeID),
					FarmId:   g.data.Nodes[contract.NodeID].FarmID,
					FarmName: g.data.Farms[g.data.Nodes[contract.NodeID].FarmID].Name,
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

func (c *RentContract) satisfies(f types.ContractFilter, nodes map[uint64]Node, farms map[uint64]Farm) bool {
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

	if len(f.State) != 0 && !containsState(f.State, c.State) {
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

	if f.FarmId != nil && *f.FarmId != nodes[c.NodeID].FarmID {
		return false
	}

	if f.FarmName != nil && !strings.EqualFold(*f.FarmName, farms[nodes[c.NodeID].FarmID].Name) {
		return false
	}

	return true
}

func (c *NodeContract) satisfies(f types.ContractFilter, nodes map[uint64]Node, farms map[uint64]Farm) bool {
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

	if f.State != nil && len(f.State) != 0 && !containsState(f.State, c.State) {
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

	if f.FarmId != nil && *f.FarmId != nodes[c.NodeID].FarmID {
		return false
	}

	if f.FarmName != nil && !strings.EqualFold(*f.FarmName, farms[nodes[c.NodeID].FarmID].Name) {
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

	if f.State != nil && len(f.State) != 0 && !containsState(f.State, c.State) {
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

	if f.FarmId != nil && *f.FarmId != 0 {
		return false
	}

	if f.FarmName != nil && *f.FarmName != "" {
		return false
	}

	return true
}

// Contract returns a single contract with the given contractID
func (g *GridProxyMockClient) Contract(ctx context.Context, contractID uint32) (res types.Contract, err error) {
	nodeContract, ok := g.data.NodeContracts[uint64(contractID)]
	if ok {
		return types.Contract{
			ContractID: uint(nodeContract.ContractID),
			TwinID:     uint(nodeContract.TwinID),
			State:      nodeContract.State,
			CreatedAt:  uint(nodeContract.CreatedAt),
			Type:       "node",
			Details: types.NodeContractDetails{
				NodeID:            uint(nodeContract.NodeID),
				DeploymentData:    nodeContract.DeploymentData,
				DeploymentHash:    nodeContract.DeploymentHash,
				NumberOfPublicIps: uint(nodeContract.NumberOfPublicIPs),
				FarmId:            g.data.Nodes[nodeContract.NodeID].FarmID,
				FarmName:          g.data.Farms[g.data.Nodes[nodeContract.NodeID].FarmID].Name,
			},
		}, err
	}

	nameContract, ok := g.data.NameContracts[uint64(contractID)]
	if ok {
		return types.Contract{
			ContractID: uint(nameContract.ContractID),
			TwinID:     uint(nameContract.TwinID),
			State:      nameContract.State,
			CreatedAt:  uint(nameContract.CreatedAt),
			Type:       "name",
			Details: types.NameContractDetails{
				Name: nameContract.Name,
			},
		}, err
	}

	rentContract, ok := g.data.RentContracts[uint64(contractID)]
	if ok {
		return types.Contract{
			ContractID: uint(rentContract.ContractID),
			TwinID:     uint(rentContract.TwinID),
			State:      rentContract.State,
			CreatedAt:  uint(rentContract.CreatedAt),
			Type:       "rent",
			Details: types.RentContractDetails{
				NodeID:   uint(rentContract.NodeID),
				FarmId:   g.data.Nodes[nodeContract.NodeID].FarmID,
				FarmName: g.data.Farms[g.data.Nodes[nodeContract.NodeID].FarmID].Name,
			},
		}, err
	}

	if !ok {
		return res, fmt.Errorf("contract not found")
	}

	return res, err
}

// ContractBills returns all bills reports for a contract with the given contract id and pagination parameters
func (g *GridProxyMockClient) ContractBills(ctx context.Context, contractID uint32, limit types.Limit) (res []types.ContractBilling, totalCount uint, err error) {
	res = []types.ContractBilling{}
	bills := g.data.Billings[uint64(contractID)]

	for _, bill := range bills {
		res = append(res, types.ContractBilling{
			AmountBilled:     bill.AmountBilled,
			DiscountReceived: bill.DiscountReceived,
			Timestamp:        bill.Timestamp,
		})
	}

	totalCount = uint(len(bills))
	return res, totalCount, err
}

func containsState(states []string, state string) bool {
	for _, _state := range states {
		if strings.EqualFold(_state, state) {
			return true
		}
	}

	return false
}
