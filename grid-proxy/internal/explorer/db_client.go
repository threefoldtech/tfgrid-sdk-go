package explorer

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// DBClient is an implementation for the db client interface [github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client.DBClient]
//
// It fetches the desired data from the database, does the appropriate type conversions, and returns the result.
type DBClient struct {
	DB db.Database
}

var _ client.DBClient = (*DBClient)(nil)

func (c *DBClient) Nodes(ctx context.Context, filter types.NodeFilter, pagination types.Limit) ([]types.Node, int, error) {
	dbNodes, cnt, err := c.DB.GetNodes(ctx, filter, pagination)
	if err != nil {
		return nil, 0, err
	}

	nodes := make([]types.Node, len(dbNodes))
	for idx, node := range dbNodes {
		nodes[idx] = nodeFromDBNode(node)
	}

	return nodes, int(cnt), nil
}

func (c *DBClient) Farms(ctx context.Context, filter types.FarmFilter, pagination types.Limit) ([]types.Farm, int, error) {
	dbFarms, cnt, err := c.DB.GetFarms(ctx, filter, pagination)
	if err != nil {
		return nil, 0, err
	}

	farms := make([]types.Farm, 0, len(dbFarms))
	for _, farm := range dbFarms {
		f, err := farmFromDBFarm(farm)
		if err != nil {
			return nil, 0, err
		}
		farms = append(farms, f)
	}

	return farms, int(cnt), nil
}

func (c *DBClient) Contracts(ctx context.Context, filter types.ContractFilter, pagination types.Limit) ([]types.Contract, int, error) {
	dbContracts, cnt, err := c.DB.GetContracts(ctx, filter, pagination)
	if err != nil {
		return nil, 0, err
	}

	contracts := make([]types.Contract, len(dbContracts))
	for idx, contract := range dbContracts {
		contracts[idx], err = contractFromDBContract(contract)
		if err != nil {
			return nil, 0, err
		}
	}

	return contracts, int(cnt), nil
}

func (c *DBClient) Contract(ctx context.Context, contractID uint32) (types.Contract, error) {
	dbContract, err := c.DB.GetContract(ctx, contractID)
	if err != nil {
		return types.Contract{}, err
	}

	contract, err := contractFromDBContract(dbContract)
	if err != nil {
		log.Err(err).Msg("failed to convert db contract to api contract")
	}

	return contract, nil
}

func (c *DBClient) ContractBills(ctx context.Context, contractID uint32, limit types.Limit) ([]types.ContractBilling, uint, error) {
	dbBills, cnt, err := c.DB.GetContractBills(ctx, contractID, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get contract %d bills: %w", contractID, err)
	}

	bills := []types.ContractBilling{}
	for _, report := range dbBills {
		bills = append(bills, types.ContractBilling(report))
	}

	return bills, cnt, nil
}

func (c *DBClient) Twins(ctx context.Context, filter types.TwinFilter, pagination types.Limit) ([]types.Twin, int, error) {
	dbTwins, cnt, err := c.DB.GetTwins(ctx, filter, pagination)
	if err != nil {
		return nil, 0, err
	}

	return dbTwins, int(cnt), nil
}

func (c *DBClient) Node(ctx context.Context, nodeID uint32) (types.NodeWithNestedCapacity, error) {
	dbNode, err := c.DB.GetNode(ctx, nodeID)
	if err != nil {
		return types.NodeWithNestedCapacity{}, err
	}

	node := nodeWithNestedCapacityFromDBNode(dbNode)
	return node, nil
}

func (c *DBClient) NodeStatus(ctx context.Context, nodeID uint32) (types.NodeStatus, error) {
	dbNode, err := c.DB.GetNode(ctx, nodeID)
	if err != nil {
		return types.NodeStatus{}, err
	}

	node := nodeWithNestedCapacityFromDBNode(dbNode)
	status := types.NodeStatus{Status: node.Status}
	return status, nil
}

func (c *DBClient) Counters(ctx context.Context, filter types.StatsFilter) (types.Counters, error) {
	return c.DB.GetCounters(ctx, filter)
}
