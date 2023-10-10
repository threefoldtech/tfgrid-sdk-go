package explorer

import (
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// GridProxyClient is an implementation for the gridproxy client interface [github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client.Client]
//
// It fetches the desired data from the database, does the appropriate type conversions, and returns the result.
type GridProxyClient struct {
	DB db.Database
}

var _ client.Client = (*GridProxyClient)(nil)

func (c *GridProxyClient) Ping() error {
	return nil
}

func (c *GridProxyClient) Nodes(filter types.NodeFilter, pagination types.Limit) ([]types.Node, int, error) {
	dbNodes, cnt, err := c.DB.GetNodes(filter, pagination)
	if err != nil {
		return nil, 0, err
	}

	nodes := make([]types.Node, len(dbNodes))
	for idx, node := range dbNodes {
		nodes[idx] = nodeFromDBNode(node)
	}

	return nodes, int(cnt), nil
}

func (c *GridProxyClient) Farms(filter types.FarmFilter, pagination types.Limit) ([]types.Farm, int, error) {
	dbFarms, cnt, err := c.DB.GetFarms(filter, pagination)
	if err != nil {
		return nil, 0, err
	}

	farms := make([]types.Farm, 0, len(dbFarms))
	for _, farm := range dbFarms {
		f, err := farmFromDBFarm(farm)
		if err != nil {
			log.Err(err).Msg("couldn't convert db farm to api farm")
		}
		farms = append(farms, f)
	}

	return farms, int(cnt), nil
}

func (c *GridProxyClient) Contracts(filter types.ContractFilter, pagination types.Limit) ([]types.Contract, int, error) {
	dbContracts, cnt, err := c.DB.GetContracts(filter, pagination)
	if err != nil {
		return nil, 0, err
	}

	contracts := make([]types.Contract, len(dbContracts))
	for idx, contract := range dbContracts {
		contracts[idx], err = contractFromDBContract(contract)
		if err != nil {
			log.Err(err).Msg("failed to convert db contract to api contract")
		}
	}

	return contracts, int(cnt), nil
}

func (c *GridProxyClient) Twins(filter types.TwinFilter, pagination types.Limit) ([]types.Twin, int, error) {
	dbTwins, cnt, err := c.DB.GetTwins(filter, pagination)
	if err != nil {
		return nil, 0, err
	}

	return dbTwins, int(cnt), nil
}

func (c *GridProxyClient) Node(nodeID uint32) (types.NodeWithNestedCapacity, error) {
	dbNode, err := c.DB.GetNode(nodeID)
	if err != nil {
		return types.NodeWithNestedCapacity{}, err
	}

	node := nodeWithNestedCapacityFromDBNode(dbNode)
	return node, nil
}

func (c *GridProxyClient) NodeStatus(nodeID uint32) (types.NodeStatus, error) {
	dbNode, err := c.DB.GetNode(nodeID)
	if err != nil {
		return types.NodeStatus{}, err
	}

	node := nodeWithNestedCapacityFromDBNode(dbNode)
	status := types.NodeStatus{Status: node.Status}
	return status, nil
}

func (c *GridProxyClient) Counters(filter types.StatsFilter) (types.Counters, error) {
	return c.DB.GetCounters(filter)
}
