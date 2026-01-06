package grid

import (
	"slices"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// IsZoslightNode checks if a node is zoslight based on proxy node data
// This is a faster alternative that doesn't require RMB calls
func IsZoslightNode(node types.Node) bool {
	// Check if node has both network-light and zmachine-light features
	hasNetworkLight := slices.Contains(node.Features, zos.NetworkLightType)
	hasZMachineLight := slices.Contains(node.Features, zos.ZMachineLightType)

	return hasNetworkLight && hasZMachineLight
}
