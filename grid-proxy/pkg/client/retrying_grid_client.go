package client

import (
	"context"
	"log"
	"time"

	backoff "github.com/cenkalti/backoff/v3"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// RetryingClient wraps the given client and does the actions with retrying
type RetryingClient struct {
	cl      Client
	timeout time.Duration
}

// NewRetryingClient retrying grid proxy client constructor
func NewRetryingClient(cl Client) Client {
	return NewRetryingClientWithTimeout(cl, 2*time.Minute)
}

// NewRetryingClient retrying grid proxy client constructor with a timeout as a parameter
func NewRetryingClientWithTimeout(cl Client, timeout time.Duration) Client {
	proxy := RetryingClient{cl, timeout}
	return &proxy
}

func bf(timeout time.Duration) *backoff.ExponentialBackOff {
	res := backoff.NewExponentialBackOff()
	res.MaxElapsedTime = timeout
	return res
}

func notify(cmd string) func(error, time.Duration) {
	return func(err error, duration time.Duration) {
		log.Printf("failure: %s, command: %s, duration: %s", err.Error(), cmd, duration)
	}
}

// Ping makes sure the server is up
func (g *RetryingClient) Ping() error {
	f := func() error {
		return g.cl.Ping()
	}
	return backoff.RetryNotify(f, bf(g.timeout), notify("ping"))

}

// Nodes returns nodes with the given filters and pagination parameters
func (g *RetryingClient) Nodes(ctx context.Context, filter types.NodeFilter, pagination types.Limit) (res []types.Node, totalCount int, err error) {
	f := func() error {
		res, totalCount, err = g.cl.Nodes(ctx, filter, pagination)
		return err
	}
	err = backoff.RetryNotify(f, bf(g.timeout), notify("nodes"))
	return
}

// Twins returns twins with the given filters and pagination parameters
func (g *RetryingClient) Twins(ctx context.Context, filter types.TwinFilter, pagination types.Limit) (res []types.Twin, totalCount int, err error) {
	f := func() error {
		res, totalCount, err = g.cl.Twins(ctx, filter, pagination)
		return err
	}
	err = backoff.RetryNotify(f, bf(g.timeout), notify("twins"))
	return
}

// Farms returns farms with the given filters and pagination parameters
func (g *RetryingClient) Farms(ctx context.Context, filter types.FarmFilter, pagination types.Limit) (res []types.Farm, totalCount int, err error) {
	f := func() error {
		res, totalCount, err = g.cl.Farms(ctx, filter, pagination)
		return err
	}
	err = backoff.RetryNotify(f, bf(g.timeout), notify("farms"))
	return
}

// Contracts returns contracts with the given filters and pagination parameters
func (g *RetryingClient) Contracts(ctx context.Context, filter types.ContractFilter, pagination types.Limit) (res []types.Contract, totalCount int, err error) {
	f := func() error {
		res, totalCount, err = g.cl.Contracts(ctx, filter, pagination)
		return err
	}
	err = backoff.RetryNotify(f, bf(g.timeout), notify("contracts"))
	return
}

// Node returns the node with the give id
func (g *RetryingClient) Node(ctx context.Context, nodeID uint32) (res types.NodeWithNestedCapacity, err error) {
	f := func() error {
		res, err = g.cl.Node(ctx, nodeID)
		return err
	}
	err = backoff.RetryNotify(f, bf(g.timeout), notify("node"))
	return
}

// Counters returns statistics about the grid
func (g *RetryingClient) Counters(ctx context.Context, filter types.StatsFilter) (res types.Counters, err error) {
	f := func() error {
		res, err = g.cl.Counters(ctx, filter)
		return err
	}
	err = backoff.RetryNotify(f, bf(g.timeout), notify("counters"))
	return
}

// Node returns the node with the give id
func (g *RetryingClient) NodeStatus(ctx context.Context, nodeID uint32) (res types.NodeStatus, err error) {
	f := func() error {
		res, err = g.cl.NodeStatus(ctx, nodeID)
		return err
	}
	err = backoff.RetryNotify(f, bf(g.timeout), notify("node_status"))
	return
}

// Contract returns the contract with the give id
func (g *RetryingClient) Contract(ctx context.Context, contractID uint32) (res types.Contract, err error) {
	f := func() error {
		res, err = g.cl.Contract(ctx, contractID)
		return err
	}
	err = backoff.RetryNotify(f, bf(g.timeout), notify("contract"))
	return
}

// ContractBills returns the contract bills with the give id
func (g *RetryingClient) ContractBills(ctx context.Context, contractID uint32, limit types.Limit) (res []types.ContractBilling, totalCount uint, err error) {
	f := func() error {
		res, totalCount, err = g.cl.ContractBills(ctx, contractID, limit)
		return err
	}
	err = backoff.RetryNotify(f, bf(g.timeout), notify("contract_bills"))
	return
}
