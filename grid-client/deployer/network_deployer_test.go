package deployer

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/mocks"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/state"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	zosTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
)

func constructTestNetwork() workloads.ZNet {
	return workloads.ZNet{
		Name:        "network",
		Description: "network for testing",
		Nodes:       []uint32{nodeID},
		IPRange: zosTypes.IPNet{IPNet: net.IPNet{
			IP:   net.IPv4(10, 1, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}},
		AddWGAccess: false,
	}
}

func constructTestNetworkDeployer(t *testing.T, tfPluginClient TFPluginClient, mock bool) (NetworkDeployer, *mocks.RMBMockClient, *mocks.MockSubstrateExt, *mocks.MockNodeClientGetter) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cl := mocks.NewRMBMockClient(ctrl)
	sub := mocks.NewMockSubstrateExt(ctrl)
	ncPool := mocks.NewMockNodeClientGetter(ctrl)

	if mock {
		tfPluginClient.SubstrateConn = sub
		tfPluginClient.NcPool = ncPool
		tfPluginClient.RMB = cl

		tfPluginClient.State.NcPool = ncPool
		tfPluginClient.State.Substrate = sub

		tfPluginClient.TwinID = twinID

		tfPluginClient.NetworkDeployer.tfPluginClient = &tfPluginClient
	}

	return tfPluginClient.NetworkDeployer, cl, sub, ncPool
}

func TestNetworkDeployer(t *testing.T) {
	tfPluginClient, err := setup()
	assert.NoError(t, err)

	t.Run("test validate", func(t *testing.T) {
		znet := constructTestNetwork()
		d, _, _, _ := constructTestNetworkDeployer(t, tfPluginClient, false)

		znet.IPRange.Mask = net.CIDRMask(20, 32)
		_, err := d.Validate(context.Background(), []workloads.Network{&znet})
		assert.Error(t, err)

		znet.IPRange.Mask = net.CIDRMask(16, 32)
		_, err = d.Validate(context.Background(), []workloads.Network{&znet})
		assert.NoError(t, err)
	})

	t.Run("test invalid network", func(t *testing.T) {
		znet := constructTestNetwork()
		znet.Nodes = []uint32{}
		d, _, _, _ := constructTestNetworkDeployer(t, tfPluginClient, false)

		znet.IPRange.Mask = net.CIDRMask(20, 32)
		_, err := d.Validate(context.Background(), []workloads.Network{&znet})
		assert.Error(t, err)

		znet.IPRange.Mask = net.CIDRMask(16, 32)
		_, err = d.Validate(context.Background(), []workloads.Network{&znet})
		assert.Error(t, err)
	})

	d, cl, sub, ncPool := constructTestNetworkDeployer(t, tfPluginClient, true)
	znet := constructTestNetwork()

	t.Run("test generate", func(t *testing.T) {
		znet.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}

		cl.EXPECT().
			Call(gomock.Any(), twinID, "zos.network.public_config_get", gomock.Any(), gomock.Any()).
			Return(nil).
			AnyTimes()

		cl.EXPECT().
			Call(gomock.Any(), twinID, "zos.network.interfaces", gomock.Any(), gomock.Any()).
			Return(nil).
			AnyTimes()

		cl.EXPECT().
			Call(gomock.Any(), twinID, "zos.network.list_wg_ports", gomock.Any(), gomock.Any()).
			Return(nil).
			AnyTimes()

		ncPool.EXPECT().
			GetNodeClient(sub, nodeID).
			Return(client.NewNodeClient(twinID, cl, d.tfPluginClient.RMBTimeout), nil).
			AnyTimes()

		dls, err := d.GenerateVersionlessDeployments(context.Background(), []workloads.Network{&znet})
		assert.NoError(t, err)

		externalIP := ""
		if znet.ExternalIP != nil {
			externalIP = znet.ExternalIP.String()
		}

		metadata, err := json.Marshal(workloads.NetworkMetaData{
			Version: int(workloads.Version3),
			UserAccesses: []workloads.UserAccess{
				{
					Subnet:     externalIP,
					PrivateKey: znet.ExternalSK.String(),
					NodeID:     znet.PublicNodeID,
				},
			},
		})
		assert.NoError(t, err)

		workload := znet.ZosWorkload(znet.NodesIPRange[nodeID], znet.Keys[nodeID].String(), uint16(znet.WGPort[nodeID]), []zosTypes.Peer{}, string(metadata), nil)
		networkDl := workloads.NewGridDeployment(twinID, 0, []zosTypes.Workload{workload})

		networkDl.Metadata = "{\"version\":3,\"type\":\"network\",\"name\":\"network\",\"projectName\":\"Network\"}"

		assert.Equal(t, len(networkDl.Workloads), len(dls[znet.Nodes[0]][0].Workloads))
		assert.Equal(t, networkDl.Workloads, dls[znet.Nodes[0]][0].Workloads)
		assert.Equal(t, dls, map[uint32][]zosTypes.Deployment{
			nodeID: {networkDl},
		})
	})
}

func TestNetworkBatchCancel(t *testing.T) {
	tfPluginClient, err := setup()
	assert.NoError(t, err)
	networks := []workloads.Network{
		&workloads.ZNet{
			NodeDeploymentID: map[uint32]uint64{
				1: 100,
				2: 200,
			},
			Nodes: []uint32{1},
		},
		&workloads.ZNet{
			NodeDeploymentID: map[uint32]uint64{
				1: 101,
				2: 201,
			},
			Nodes: []uint32{2},
		},
	}
	tfPluginClient.State.CurrentNodeDeployments = map[uint32]state.ContractIDs{
		1: []uint64{100, 101, 12345},
		2: []uint64{200, 201},
	}
	net, _, sub, _ := constructTestNetworkDeployer(t, tfPluginClient, true)
	sub.EXPECT().BatchCancelContract(
		tfPluginClient.Identity,
		gomock.InAnyOrder([]uint64{100, 101, 200, 201}),
	).Return(nil)
	err = net.BatchCancel(context.Background(), networks)
	assert.NoError(t, err)
	assert.Len(t, networks[0].GetNodeDeploymentID(), 0)
	assert.Len(t, networks[1].GetNodeDeploymentID(), 0)
	assert.Len(t, tfPluginClient.State.CurrentNodeDeployments[1], 1)
	assert.Len(t, tfPluginClient.State.CurrentNodeDeployments[2], 0)
}

func ExampleNetworkDeployer_Deploy() {
	const mnemonic = "<mnemonics goes here>"
	const network = "<dev, test, qa, main>"
	const nodeID = 11 // use any node with status up, use ExampleFilterNodes to get valid nodeID

	tfPluginClient, err := NewTFPluginClient(mnemonic, WithNetwork(network), WithRMBTimeout(10))
	if err != nil {
		fmt.Println(err)
		return
	}

	n := workloads.ZNet{
		Name:        "network",
		Description: "network for testing",
		Nodes:       []uint32{nodeID},
		IPRange: zosTypes.IPNet{IPNet: net.IPNet{
			IP:   net.IPv4(10, 1, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}},
		AddWGAccess: false,
	}

	err = tfPluginClient.NetworkDeployer.Deploy(context.Background(), &n)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("deployment is done successfully")
}

func ExampleNetworkDeployer_BatchDeploy() {
	const mnemonic = "<mnemonics goes here>"
	const network = "<dev, test, qa, main>"
	const nodeID = 11 // use any node with status up, use ExampleFilterNodes to get valid nodeID

	tfPluginClient, err := NewTFPluginClient(mnemonic, WithNetwork(network))
	if err != nil {
		fmt.Println(err)
		return
	}

	n1 := workloads.ZNet{
		Name:        "network1",
		Description: "network for testing",
		Nodes:       []uint32{nodeID},
		IPRange: zosTypes.IPNet{IPNet: net.IPNet{
			IP:   net.IPv4(10, 1, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}},
		AddWGAccess: false,
	}
	n2 := workloads.ZNet{
		Name:        "network2",
		Description: "network for testing",
		Nodes:       []uint32{nodeID},
		IPRange: zosTypes.IPNet{IPNet: net.IPNet{
			IP:   net.IPv4(10, 1, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}},
		AddWGAccess: false,
	}

	err = tfPluginClient.NetworkDeployer.BatchDeploy(context.Background(), []workloads.Network{&n1, &n2})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("batch deployment is done successfully")
}

func ExampleNetworkDeployer_Cancel() {
	const mnemonic = "<mnemonics goes here>"
	const network = "<dev, test, qa, main>"

	tfPluginClient, err := NewTFPluginClient(mnemonic, WithNetwork(network))
	if err != nil {
		fmt.Println(err)
		return
	}

	// should be a valid and existing network name
	networkName := "network"
	n, err := tfPluginClient.State.LoadNetworkFromGrid(context.Background(), networkName)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = tfPluginClient.NetworkDeployer.Cancel(context.Background(), &n)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("deployment is canceled successfully")
}
