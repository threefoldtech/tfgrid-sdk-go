package explorer

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/mw"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

func errorReply(err error) mw.Response {
	if errors.Is(err, ErrNodeNotFound) {
		return mw.NotFound(err)
	} else if errors.Is(err, ErrGatewayNotFound) {
		return mw.NotFound(err)
	} else if errors.Is(err, ErrBadGateway) {
		return mw.BadGateway(err)
	} else {
		return mw.Error(err)
	}
}

func createResponse(totalCount uint, limit types.Limit) mw.Response {
	r := mw.Ok()

	if limit.RetCount {
		pages := math.Ceil(float64(totalCount) / float64(limit.Size))

		r = r.
			WithHeader("count", fmt.Sprintf("%d", totalCount)).
			WithHeader("size", fmt.Sprintf("%d", limit.Size)).
			WithHeader("pages", fmt.Sprintf("%d", int(pages)))
	}
	return r
}

// getNodeData is a helper function that wraps fetch node data
// it caches the results in redis to save time
func (a *App) getNodeData(ctx context.Context, nodeIDStr string) (types.NodeWithNestedCapacity, error) {
	nodeID, err := strconv.Atoi(nodeIDStr)
	if err != nil {
		return types.NodeWithNestedCapacity{}, errors.Wrap(ErrBadGateway, fmt.Sprintf("invalid node id %d: %s", nodeID, err.Error()))
	}
	info, err := a.cl.Node(ctx, uint32(nodeID))
	if errors.Is(err, db.ErrNodeNotFound) {
		return types.NodeWithNestedCapacity{}, ErrNodeNotFound
	} else if err != nil {
		// TODO: wrapping
		return types.NodeWithNestedCapacity{}, err
	}

	return info, nil
}

// getContractData is a helper function that wraps fetch contract data
func (a *App) getContractData(ctx context.Context, contractIDStr string) (types.Contract, error) {
	contractID, err := strconv.Atoi(contractIDStr)
	if err != nil {
		return types.Contract{}, errors.Wrapf(err, "invalid contract id: %s", contractIDStr)
	}

	info, err := a.cl.Contract(ctx, uint32(contractID))

	if errors.Is(err, db.ErrContractNotFound) {
		return types.Contract{}, ErrContractNotFound
	} else if err != nil {
		return types.Contract{}, err
	}

	return info, nil
}

// getContractBillsData is a helper function that gets bills reports for a single contract data
func (a *App) getContractBillsData(ctx context.Context, contractIDStr string, limit types.Limit) ([]types.ContractBilling, uint, error) {
	contractID, err := strconv.Atoi(contractIDStr)
	if err != nil {
		return []types.ContractBilling{}, 0, errors.Wrapf(err, "invalid contract id: %s", contractIDStr)
	}

	bills, billsCount, err := a.cl.ContractBills(ctx, uint32(contractID), limit)
	if err != nil {
		return []types.ContractBilling{}, 0, errors.Wrapf(err, "contract not found with id: %d", contractID)
	}

	return bills, billsCount, nil
}

// calcContractConsumption for the last hour
func calcContractConsumption(c db.DBContract, latestBills []db.ContractBilling) float64 {
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
