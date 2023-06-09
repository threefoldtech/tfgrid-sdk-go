// Package state for grid state
package state

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

var contractID uint64 = 100
var nodeID uint32 = 10

func constructTestNetwork() workloads.ZNet {
	return workloads.ZNet{
		Name:        "network",
		Description: "network for testing",
		Nodes:       []uint32{nodeID},
		IPRange: gridtypes.NewIPNet(net.IPNet{
			IP:   net.IPv4(10, 1, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}),
		AddWGAccess: false,
	}
}

func TestNetworkState(t *testing.T) {
	net := constructTestNetwork()
	nodeID = net.Nodes[0]

	networkState := NetworkState{net.Name: Network{
		Subnets:               map[uint32]string{nodeID: net.IPRange.String()},
		NodeDeploymentHostIDs: map[uint32]DeploymentHostIDs{nodeID: map[uint64][]byte{contractID: {}}},
	}}
	network := networkState.GetNetwork(net.Name)

	assert.Equal(t, network.GetNodeSubnet(nodeID), net.IPRange.String())
	assert.Empty(t, network.GetDeploymentHostIDs(nodeID, contractID))

	network.SetNodeSubnet(nodeID, "10.1.1.0/24")
	assert.Equal(t, network.GetNodeSubnet(nodeID), "10.1.1.0/24")

	network.SetDeploymentHostIDs(nodeID, contractID, []byte{1, 2, 3})
	assert.Equal(t, network.GetDeploymentHostIDs(nodeID, contractID), []byte{1, 2, 3})

	network.deleteNodeSubnet(nodeID)
	assert.Empty(t, network.GetNodeSubnet(nodeID))

	network.DeleteDeploymentHostIDs(nodeID, contractID)
	assert.Empty(t, network.GetDeploymentHostIDs(nodeID, contractID))
}
