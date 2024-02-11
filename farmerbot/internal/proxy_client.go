package internal

import (
	"context"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// Substrate is substrate client interface
type ProxyClient interface {
	Node(ctx context.Context, nodeID uint32) (res types.NodeWithNestedCapacity, err error)
}
