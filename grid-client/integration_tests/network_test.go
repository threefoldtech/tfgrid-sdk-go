// Package integration for integration tests
package integration

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
)

func TestNetworkDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	if err != nil {
		t.Skipf("plugin creation failed: %v", err)
	}

	nodes, err := deployer.FilterNodes(context.Background(), tfPluginClient, generateNodeFilter(), nil, nil, nil, 2)
	if err != nil {
		t.Skipf("no available nodes found: %v", err)
	}

	nodeID1 := uint32(nodes[0].NodeID)
	nodeID2 := uint32(nodes[1].NodeID)

	network := workloads.ZNet{
		Name:        fmt.Sprintf("net_%s", generateRandString(10)),
		Description: "not skynet",
		Nodes:       []uint32{nodeID1},
		IPRange: zos.IPNet{IPNet: net.IPNet{
			IP:   net.IPv4(10, 1, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}},
		AddWGAccess: true,
	}

	networkCp := network

	t.Run("deploy network with wireguard access", func(t *testing.T) {
		err = tfPluginClient.NetworkDeployer.Deploy(context.Background(), &network)
		require.NoError(t, err)

		t.Cleanup(func() {
			err = tfPluginClient.NetworkDeployer.Cancel(context.Background(), &network)
			require.NoError(t, err)
		})

		net, err := tfPluginClient.State.LoadNetworkFromGrid(context.Background(), network.Name)
		require.NoError(t, err)

		require.NotEmpty(t, net.AccessWGConfig)
	})

	t.Run("deploy network with wireguard access on different nodes", func(t *testing.T) {
		networkCp.Nodes = []uint32{nodeID2}

		err = tfPluginClient.NetworkDeployer.Deploy(context.Background(), &networkCp)
		require.NoError(t, err)

		_, err := tfPluginClient.State.LoadNetworkFromGrid(context.Background(), networkCp.Name)
		require.NoError(t, err)

		net, err := tfPluginClient.State.LoadNetworkFromGrid(context.Background(), network.Name)
		require.NoError(t, err)

		require.NotEmpty(t, net.AccessWGConfig)
	})

	t.Run("update network remove wireguard access", func(t *testing.T) {
		networkCp.AddWGAccess = false
		networkCp.Nodes = []uint32{nodeID2}

		err = tfPluginClient.NetworkDeployer.Deploy(context.Background(), &networkCp)
		require.NoError(t, err)

		t.Cleanup(func() {
			err = tfPluginClient.NetworkDeployer.Cancel(context.Background(), &networkCp)
			require.NoError(t, err)
		})

		net, err := tfPluginClient.State.LoadNetworkFromGrid(context.Background(), network.Name)
		require.NoError(t, err)

		require.Empty(t, net.AccessWGConfig)
	})

	t.Run("test get public node", func(t *testing.T) {
		publicNodeID, err := deployer.GetPublicNode(
			context.Background(),
			tfPluginClient,
			[]uint32{},
		)
		require.NoError(t, err)
		require.NotEqual(t, 0, publicNodeID)
	})
}
