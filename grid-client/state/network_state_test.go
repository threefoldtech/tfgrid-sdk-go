// Package state for grid state
package state

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
)

var nodeID uint32 = 10

func constructTestNetwork() workloads.ZNet {
	return workloads.ZNet{
		Name:        "network",
		Description: "network for testing",
		Nodes:       []uint32{nodeID},
		IPRange: zos.IPNet{IPNet: net.IPNet{
			IP:   net.IPv4(10, 1, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}},
		AddWGAccess: false,
	}
}

func TestNetworkState(t *testing.T) {
	net := constructTestNetwork()
	nodeID = net.Nodes[0]

	networkState := NetworkState{
		State: map[string]Network{
			net.Name: {Subnets: map[uint32]string{nodeID: net.IPRange.String()}},
		},
	}

	network := networkState.GetNetwork(net.Name)

	assert.Equal(t, network.GetNodeSubnet(nodeID), net.IPRange.String())

	network.SetNodeSubnet(nodeID, "10.1.1.0/24")
	assert.Equal(t, network.GetNodeSubnet(nodeID), "10.1.1.0/24")

	network.deleteNodeSubnet(nodeID)
	assert.Empty(t, network.GetNodeSubnet(nodeID))
}
