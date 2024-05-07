package deployer

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"testing"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/mocks"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/state"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	proxyTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func constructTestFQDNDeployer(t *testing.T, mock bool) (
	GatewayFQDNDeployer,
	*mocks.RMBMockClient,
	*mocks.MockSubstrateExt,
	*mocks.MockNodeClientGetter,
	*mocks.MockDeployer,
	*mocks.MockClient,
) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tfPluginClient, err := setup()
	assert.NoError(t, err)

	cl := mocks.NewRMBMockClient(ctrl)
	sub := mocks.NewMockSubstrateExt(ctrl)
	ncPool := mocks.NewMockNodeClientGetter(ctrl)
	deployer := mocks.NewMockDeployer(ctrl)
	gridProxyCl := mocks.NewMockClient(ctrl)

	if mock {
		tfPluginClient.TwinID = twinID

		tfPluginClient.SubstrateConn = sub
		tfPluginClient.NcPool = ncPool
		tfPluginClient.RMB = cl
		tfPluginClient.GridProxyClient = gridProxyCl

		tfPluginClient.State.NcPool = ncPool
		tfPluginClient.State.Substrate = sub

		tfPluginClient.GatewayFQDNDeployer.deployer = deployer
		tfPluginClient.GatewayFQDNDeployer.tfPluginClient = &tfPluginClient
	}

	return tfPluginClient.GatewayFQDNDeployer, cl, sub, ncPool, deployer, gridProxyCl
}

func constructTestFQDN() workloads.GatewayFQDNProxy {
	return workloads.GatewayFQDNProxy{
		NodeID:         nodeID,
		Name:           "name",
		TLSPassthrough: false,
		Backends:       []zos.Backend{"http://1.1.1.1", "http://2.2.2.2"},
		FQDN:           "name.com",
	}
}

func mockValidation(identity substrate.Identity, cl *mocks.RMBMockClient, sub *mocks.MockSubstrateExt, ncPool *mocks.MockNodeClientGetter, proxyCl *mocks.MockClient) {
	sub.EXPECT().
		GetBalance(identity).
		Return(substrate.Balance{
			Free: types.U128{
				Int: big.NewInt(20000000),
			},
		}, nil)

	proxyCl.EXPECT().Node(context.Background(), nodeID).
		Return(proxyTypes.NodeWithNestedCapacity{
			NodeID:       int(nodeID),
			FarmID:       1,
			PublicConfig: proxyTypes.PublicConfig{Ipv4: "1.1.1.1/16"},
		}, nil)

	proxyCl.EXPECT().Farms(context.Background(), gomock.Any(), gomock.Any()).Return([]proxyTypes.Farm{{FarmID: 1}}, 1, nil)

	ncPool.EXPECT().
		GetNodeClient(sub, nodeID).AnyTimes().
		Return(client.NewNodeClient(twinID, cl, 10), nil)

	cl.EXPECT().Call(
		gomock.Any(),
		twinID,
		"zos.network.public_config_get",
		gomock.Any(),
		gomock.Any()).
		DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
			var res *client.PublicConfig = result.(*client.PublicConfig)
			cfg := client.PublicConfig{IPv4: gridtypes.IPNet{IPNet: net.IPNet{IP: net.ParseIP("192.168.1.10")}}}
			*res = cfg
			return nil
		}).AnyTimes()

	cl.EXPECT().Call(
		gomock.Any(),
		twinID,
		"zos.system.version",
		gomock.Any(),
		gomock.Any(),
	).Return(nil)
}

func TestFQDNDeployer(t *testing.T) {
	d, cl, sub, ncPool, deployer, proxyCl := constructTestFQDNDeployer(t, true)
	gw := constructTestFQDN()

	t.Run("test validate", func(t *testing.T) {
		mockValidation(d.tfPluginClient.Identity, cl, sub, ncPool, proxyCl)
		err := d.Validate(context.Background(), &workloads.GatewayFQDNProxy{Name: "test", NodeID: nodeID})
		assert.NoError(t, err)
	})

	t.Run("test generate", func(t *testing.T) {
		dls, err := d.GenerateVersionlessDeployments(context.Background(), &gw)
		assert.NoError(t, err)

		testDl := workloads.NewGridDeployment(twinID, []gridtypes.Workload{
			{
				Version: 0,
				Type:    zos.GatewayFQDNProxyType,
				Name:    gridtypes.Name(gw.Name),
				Data: gridtypes.MustMarshal(zos.GatewayFQDNProxy{
					GatewayBase: zos.GatewayBase{
						TLSPassthrough: gw.TLSPassthrough,
						Backends:       gw.Backends,
					},
					FQDN: gw.FQDN,
				}),
			},
		})
		testDl.Metadata = "{\"version\":3,\"type\":\"Gateway Fqdn\",\"name\":\"name\",\"projectName\":\"name\"}"

		assert.Equal(t, dls, map[uint32]gridtypes.Deployment{
			nodeID: testDl,
		})
	})

	t.Run("test deploy", func(t *testing.T) {
		dls, err := d.GenerateVersionlessDeployments(context.Background(), &gw)
		assert.NoError(t, err)

		mockValidation(d.tfPluginClient.Identity, cl, sub, ncPool, proxyCl)

		deployer.EXPECT().Deploy(
			gomock.Any(),
			gw.NodeDeploymentID,
			dls,
			gomock.Any(),
		).Return(map[uint32]uint64{nodeID: contractID}, nil)

		err = d.Deploy(context.Background(), &gw)
		assert.NoError(t, err)
		assert.NotEqual(t, gw.ContractID, 0)
		assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
		assert.Equal(t, d.tfPluginClient.State.CurrentNodeDeployments, map[uint32]state.ContractIDs{nodeID: {contractID}})
	})

	t.Run("test update", func(t *testing.T) {
		d.tfPluginClient.State.CurrentNodeDeployments = map[uint32]state.ContractIDs{nodeID: {contractID}}

		gw.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}

		dls, err := d.GenerateVersionlessDeployments(context.Background(), &gw)
		assert.NoError(t, err)

		mockValidation(d.tfPluginClient.Identity, cl, sub, ncPool, proxyCl)

		deployer.EXPECT().Deploy(
			gomock.Any(),
			map[uint32]uint64{nodeID: contractID},
			dls,
			gomock.Any(),
		).Return(map[uint32]uint64{nodeID: contractID}, nil)

		err = d.Deploy(context.Background(), &gw)
		assert.NoError(t, err)
		assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
		assert.Equal(t, d.tfPluginClient.State.CurrentNodeDeployments, map[uint32]state.ContractIDs{nodeID: {contractID}})
	})

	t.Run("test update failed", func(t *testing.T) {
		d.tfPluginClient.State.CurrentNodeDeployments = map[uint32]state.ContractIDs{nodeID: {contractID}}

		gw.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}

		dls, err := d.GenerateVersionlessDeployments(context.Background(), &gw)
		assert.NoError(t, err)

		mockValidation(d.tfPluginClient.Identity, cl, sub, ncPool, proxyCl)

		deployer.EXPECT().Deploy(
			gomock.Any(),
			map[uint32]uint64{nodeID: contractID},
			dls,
			gomock.Any(),
		).Return(map[uint32]uint64{nodeID: contractID}, errors.New("error"))

		err = d.Deploy(context.Background(), &gw)
		assert.Error(t, err)
		assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
		assert.Equal(t, d.tfPluginClient.State.CurrentNodeDeployments, map[uint32]state.ContractIDs{nodeID: {contractID}})
	})

	t.Run("test cancel", func(t *testing.T) {
		d.tfPluginClient.State.CurrentNodeDeployments = map[uint32]state.ContractIDs{nodeID: {contractID}}

		gw.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}

		mockValidation(d.tfPluginClient.Identity, cl, sub, ncPool, proxyCl)

		deployer.EXPECT().Cancel(
			gomock.Any(),
			contractID,
		).Return(nil)

		err := d.Cancel(context.Background(), &gw)
		assert.NoError(t, err)
		assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{})
		assert.Empty(t, d.tfPluginClient.State.CurrentNodeDeployments[nodeID])
	})

	t.Run("test cancel failed", func(t *testing.T) {
		d.tfPluginClient.State.CurrentNodeDeployments = map[uint32]state.ContractIDs{nodeID: {contractID}}

		gw := constructTestFQDN()
		gw.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}

		mockValidation(d.tfPluginClient.Identity, cl, sub, ncPool, proxyCl)

		deployer.EXPECT().Cancel(
			gomock.Any(),
			contractID,
		).Return(errors.New("error"))

		err := d.Cancel(context.Background(), &gw)
		assert.Error(t, err)
		assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
		assert.Equal(t, d.tfPluginClient.State.CurrentNodeDeployments, map[uint32]state.ContractIDs{nodeID: {contractID}})
	})

	t.Run("test sync contracts", func(t *testing.T) {
		gw.ContractID = contractID
		gw.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}

		sub.EXPECT().DeleteInvalidContracts(
			gw.NodeDeploymentID,
		).Return(nil)

		err := d.syncContracts(context.Background(), &gw)
		assert.NoError(t, err)
		assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
		assert.Equal(t, gw.ContractID, contractID)
	})

	t.Run("test sync contracts deleted", func(t *testing.T) {
		gw.ContractID = contractID
		gw.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}

		sub.EXPECT().DeleteInvalidContracts(
			gw.NodeDeploymentID,
		).DoAndReturn(func(contracts map[uint32]uint64) error {
			delete(contracts, nodeID)
			return nil
		})

		err := d.syncContracts(context.Background(), &gw)
		assert.NoError(t, err)
		assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{})
		assert.Equal(t, gw.ContractID, uint64(0))
	})

	t.Run("test sync contracts failed", func(t *testing.T) {
		gw.ContractID = contractID
		gw.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}

		sub.EXPECT().DeleteInvalidContracts(
			gw.NodeDeploymentID,
		).Return(errors.New("error"))

		err := d.syncContracts(context.Background(), &gw)
		assert.Error(t, err)
		assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
		assert.Equal(t, gw.ContractID, contractID)
	})

	t.Run("test sync contracts failed in contract", func(t *testing.T) {
		gw.ContractID = contractID
		gw.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}

		sub.EXPECT().DeleteInvalidContracts(
			gw.NodeDeploymentID,
		).Return(errors.New("error"))

		err := d.Sync(context.Background(), &gw)
		assert.Error(t, err)
		assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
		assert.Equal(t, gw.ContractID, contractID)
	})

	t.Run("test sync", func(t *testing.T) {
		gw.ContractID = contractID
		gw.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}

		dls, err := d.GenerateVersionlessDeployments(context.Background(), &gw)
		assert.NoError(t, err)

		dl := dls[nodeID]
		dl.Workloads[0].Result.State = gridtypes.StateOk
		dl.Workloads[0].Result.Data, err = json.Marshal(zos.GatewayFQDNResult{})
		assert.NoError(t, err)

		sub.EXPECT().DeleteInvalidContracts(
			gw.NodeDeploymentID,
		).Return(nil)

		deployer.EXPECT().
			GetDeployments(gomock.Any(), map[uint32]uint64{nodeID: contractID}).
			DoAndReturn(func(ctx context.Context, _ map[uint32]uint64) (map[uint32]gridtypes.Deployment, error) {
				return map[uint32]gridtypes.Deployment{nodeID: dl}, nil
			})

		gw.FQDN = "123"
		err = d.Sync(context.Background(), &gw)
		assert.NoError(t, err)
		assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
		assert.Equal(t, gw.ContractID, contractID)
		assert.Equal(t, gw.FQDN, "name.com")
	})

	t.Run("test sync deleted workloads", func(t *testing.T) {
		gw.ContractID = contractID
		gw.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}

		dls, err := d.GenerateVersionlessDeployments(context.Background(), &gw)
		assert.NoError(t, err)

		dl := dls[nodeID]
		// state is deleted

		sub.EXPECT().DeleteInvalidContracts(
			gw.NodeDeploymentID,
		).Return(nil)

		deployer.EXPECT().
			GetDeployments(gomock.Any(), map[uint32]uint64{nodeID: contractID}).
			DoAndReturn(func(ctx context.Context, _ map[uint32]uint64) (map[uint32]gridtypes.Deployment, error) {
				return map[uint32]gridtypes.Deployment{nodeID: dl}, nil
			})

		gw.FQDN = "123"
		err = d.Sync(context.Background(), &gw)
		assert.NoError(t, err)
		assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
		assert.Equal(t, gw.ContractID, contractID)
		assert.Equal(t, gw.FQDN, "")
		assert.Equal(t, gw.Name, "")
		assert.Equal(t, gw.TLSPassthrough, false)
		assert.Equal(t, gw.Backends, []zos.Backend(nil))
	})
}

func ExampleGatewayFQDNDeployer_Deploy() {
	const mnemonic = "<mnemonics goes here>"
	const network = "<dev, test, qa, main>"
	const nodeID = 11 // use any node with status up, use ExampleFilterNodes to get valid nodeID

	tfPluginClient, err := NewTFPluginClient(mnemonic, WithNetwork(network))
	if err != nil {
		fmt.Println(err)
		return
	}
	g := workloads.GatewayFQDNProxy{
		NodeID:         nodeID,
		Name:           "test1",
		TLSPassthrough: false,
		Backends:       []zos.Backend{"http://1.1.1.1"},
		FQDN:           "name.com",
	}

	err = tfPluginClient.GatewayFQDNDeployer.Deploy(context.Background(), &g)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("deployment is done successfully")
}

func ExampleGatewayFQDNDeployer_BatchDeploy() {
	const mnemonic = "<mnemonics goes here>"
	const network = "<dev, test, qa, main>"
	const nodeID = 11 // use any node with status up, use ExampleFilterNodes to get valid nodeID

	tfPluginClient, err := NewTFPluginClient(mnemonic, WithNetwork(network))
	if err != nil {
		fmt.Println(err)
		return
	}
	g1 := workloads.GatewayFQDNProxy{
		NodeID:         nodeID,
		Name:           "test1",
		TLSPassthrough: false,
		Backends:       []zos.Backend{"http://1.1.1.1"},
		FQDN:           "test1.com",
	}
	g2 := workloads.GatewayFQDNProxy{
		NodeID:         nodeID,
		Name:           "test2",
		TLSPassthrough: false,
		Backends:       []zos.Backend{"http://2.2.2.2"},
		FQDN:           "test2.com",
	}

	err = tfPluginClient.GatewayFQDNDeployer.BatchDeploy(context.Background(), []*workloads.GatewayFQDNProxy{&g1, &g2})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("batch deployment is done successfully")
}

func ExampleGatewayFQDNDeployer_Cancel() {
	const mnemonic = "<mnemonics goes here>"
	const network = "<dev, test, qa, main>"
	const nodeID = 11 // use any node with status up, use ExampleFilterNodes to get valid nodeID

	tfPluginClient, err := NewTFPluginClient(mnemonic, WithNetwork(network))
	if err != nil {
		fmt.Println(err)
		return
	}

	// should be a valid and existing name and deploymentName
	name := "test1.com"
	deploymentName := "test1"
	g, err := tfPluginClient.State.LoadGatewayFQDNFromGrid(context.Background(), nodeID, name, deploymentName)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = tfPluginClient.GatewayFQDNDeployer.Cancel(context.Background(), &g)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("deployment is canceled successfully")
}
