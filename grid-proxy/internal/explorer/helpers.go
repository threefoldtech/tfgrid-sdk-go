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
	info, err := a.db.GetNode(ctx, uint32(nodeID))
	if errors.Is(err, db.ErrNodeNotFound) {
		return types.NodeWithNestedCapacity{}, ErrNodeNotFound
	} else if err != nil {
		// TODO: wrapping
		return types.NodeWithNestedCapacity{}, err
	}
	apiNode := nodeWithNestedCapacityFromDBNode(info)
	return apiNode, nil
}

// getContractData is a helper function that wraps fetch contract data
func (a *App) getContractData(ctx context.Context, contractIDStr string) (types.Contract, error) {
	contractID, err := strconv.Atoi(contractIDStr)
	if err != nil {
		return types.Contract{}, errors.Wrapf(err, "invalid contract id: %s", contractIDStr)
	}

	info, err := a.db.GetContract(ctx, uint32(contractID))

	if errors.Is(err, db.ErrContractNotFound) {
		return types.Contract{}, ErrContractNotFound
	} else if err != nil {
		return types.Contract{}, err
	}

	res, err := contractFromDBContract(info)
	if err != nil {
		return types.Contract{}, err
	}

	return res, nil
}

// getContractBillsData is a helper function that gets bills reports for a single contract data
func (a *App) getContractBillsData(ctx context.Context, contractIDStr string, limit types.Limit) ([]types.ContractBilling, uint, error) {
	contractID, err := strconv.Atoi(contractIDStr)
	if err != nil {
		return []types.ContractBilling{}, 0, errors.Wrapf(err, "invalid contract id: %s", contractIDStr)
	}

	info, billsCount, err := a.db.GetContractBills(ctx, uint32(contractID), limit)
	if err != nil {
		return []types.ContractBilling{}, 0, errors.Wrapf(err, "contract not found with id: %d", contractID)
	}

	bills := []types.ContractBilling{}
	for _, report := range info {
		bills = append(bills, types.ContractBilling(report))
	}

	return bills, billsCount, nil
}
