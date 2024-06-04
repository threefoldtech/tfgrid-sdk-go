package explorer

import (
	"context"
	"fmt"
	"math"

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

func (c *DBClient) Stats(ctx context.Context, filter types.StatsFilter) (types.Stats, error) {
	return c.DB.GetStats(ctx, filter)
}

func (c *DBClient) GetTwinFees(ctx context.Context, twinId uint64) (types.TwinFee, error) {
	// get all contracts for a twin id
	filter := types.ContractFilter{
		TwinID: &twinId,
		State:  []string{"Created", "GracePeriod", "Deleted"},
	}
	limit := types.Limit{
		Size: 99999,
	}
	twinContracts, _, err := c.DB.GetContracts(ctx, filter, limit)
	if err != nil {
		return types.TwinFee{}, err
	}

	nonDeletedCIds := []uint32{}
	allCIds := []uint32{}
	contracts := make(map[uint32]db.DBContract)
	for _, contract := range twinContracts {
		if contract.State != "Deleted" {
			nonDeletedCIds = append(nonDeletedCIds, uint32(contract.ContractID))
		}
		allCIds = append(allCIds, uint32(contract.ContractID))
		contracts[uint32(contract.ContractID)] = contract
	}

	// get the latest two reports for each contract
	reports, err := c.DB.GetContractReports(ctx, nonDeletedCIds, 2)
	if err != nil {
		return types.TwinFee{}, err
	}

	contractReports := make(map[uint32][]db.ContractBilling)
	for _, report := range reports {
		contractReports[uint32(report.ContractId)] = append(contractReports[uint32(report.ContractId)], report)
	}

	// calc bills
	var fee types.TwinFee
	for _, id := range nonDeletedCIds {
		contractFee := calcContractFee(contracts[id], contractReports[id])
		fee.LastHourSpent += contractFee
	}

	// calc all spent
	spent, err := c.DB.GetContractsTotalBilledAmount(ctx, allCIds)
	if err != nil {
		return types.TwinFee{}, err
	}

	fee.TotalSpend = float64(spent) / math.Pow(10, 7)

	return fee, err
}

func calcContractFee(c db.DBContract, latestBills []db.ContractBilling) float64 {
	var duration float64
	switch len(latestBills) {
	case 0:
		return 0
	case 1:
		duration = float64(latestBills[0].Timestamp-uint64(c.CreatedAt)) / float64(3600)
	case 2:
		duration = float64(latestBills[0].Timestamp-latestBills[1].Timestamp) / float64(3600)
	default:
		duration = 1
	}

	return float64(latestBills[0].AmountBilled) / duration / math.Pow(10, 7)
}
