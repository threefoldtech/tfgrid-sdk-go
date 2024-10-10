package deployer

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
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
	zosTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

var nameContractID uint64 = 200

func constructTestNameDeployer(t *testing.T, mock bool) (
	GatewayNameDeployer,
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

		tfPluginClient.GatewayNameDeployer.deployer = deployer
		tfPluginClient.GatewayNameDeployer.tfPluginClient = &tfPluginClient
	}

	return tfPluginClient.GatewayNameDeployer, cl, sub, ncPool, deployer, gridProxyCl
}

func constructTestName() workloads.GatewayNameProxy {
	return workloads.GatewayNameProxy{
		NodeID:         nodeID,
		Name:           "name",
		TLSPassthrough: false,
		Backends:       []zos.Backend{"http://1.1.1.1"},
		FQDN:           "name.com",
	}
}

func TestNameDeployer(t *testing.T) {
	d, cl, sub, ncPool, deployer, proxyCl := constructTestNameDeployer(t, true)
	gw := constructTestName()

	t.Run("test validate node not reachable", func(t *testing.T) {
		sub.EXPECT().
			GetBalance(d.tfPluginClient.Identity).
			Return(substrate.Balance{
				Free: types.U128{
					Int: big.NewInt(20000000),
				},
			}, nil)
		cl.
			EXPECT().
			Call(
				gomock.Any(),
				nodeID,
				"zos.system.version",
				nil,
				gomock.Any(),
			).
			Return(errors.New("could not reach node"))
		ncPool.
			EXPECT().
			GetNodeClient(
				gomock.Any(),
				nodeID,
			).
			Return(client.NewNodeClient(nodeID, cl, d.tfPluginClient.RMBTimeout), nil)

		err := d.Validate(context.TODO(), &gw)
		assert.Error(t, err)
	})

	t.Run("test generate", func(t *testing.T) {
		dls, err := d.GenerateVersionlessDeployments(context.Background(), &gw)
		assert.NoError(t, err)

		testDl := workloads.NewGridDeployment(twinID, 0, []zosTypes.Workload{
			{
				Version: 0,
				Type:    zosTypes.GatewayNameProxyType,
				Name:    gw.Name,
				Data: gridtypes.MustMarshal(zos.GatewayNameProxy{
					GatewayBase: zos.GatewayBase{
						TLSPassthrough: gw.TLSPassthrough,
						Backends:       gw.Backends,
					},
					Name: gw.Name,
				}),
			},
		})
		testDl.Metadata = "{\"version\":3,\"type\":\"Gateway Name\",\"name\":\"name\",\"projectName\":\"name\"}"

		assert.Equal(t, dls, map[uint32]zosTypes.Deployment{
			nodeID: testDl,
		})
	})

	t.Run("test deploy", func(t *testing.T) {
		dls, err := d.GenerateVersionlessDeployments(context.Background(), &gw)
		assert.NoError(t, err)

		mockValidation(d.tfPluginClient.Identity, cl, sub, ncPool, proxyCl)

		newDeploymentsSolutionProvider := map[uint32]*uint64{nodeID: nil}

		deployer.EXPECT().Deploy(
			gomock.Any(),
			gw.NodeDeploymentID,
			dls,
			newDeploymentsSolutionProvider,
		).Return(map[uint32]uint64{nodeID: contractID}, nil)

		sub.EXPECT().
			CreateNameContract(d.tfPluginClient.Identity, gw.Name).
			Return(contractID, nil)

		err = d.Deploy(context.Background(), &gw)
		assert.NoError(t, err)
		assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
		assert.Equal(t, d.tfPluginClient.State.CurrentNodeDeployments, map[uint32]state.ContractIDs{nodeID: {contractID}})
	})

	t.Run("test update", func(t *testing.T) {
		d.tfPluginClient.State.CurrentNodeDeployments = map[uint32]state.ContractIDs{nodeID: {contractID}}

		gw.NameContractID = nameContractID
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

		sub.EXPECT().
			InvalidateNameContract(gomock.Any(), d.tfPluginClient.Identity, nameContractID, gw.Name).
			Return(nameContractID, nil)

		err = d.Deploy(context.Background(), &gw)
		assert.NoError(t, err)
		assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
		assert.Equal(t, d.tfPluginClient.State.CurrentNodeDeployments, map[uint32]state.ContractIDs{nodeID: {contractID}})
	})

	t.Run("test update failed", func(t *testing.T) {
		d.tfPluginClient.State.CurrentNodeDeployments = map[uint32]state.ContractIDs{nodeID: {contractID}}

		gw.NameContractID = nameContractID
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

		sub.EXPECT().
			InvalidateNameContract(gomock.Any(), d.tfPluginClient.Identity, nameContractID, gw.Name).
			Return(nameContractID, nil)
		sub.EXPECT().
			CancelContract(d.tfPluginClient.Identity, nameContractID).
			Return(nil)
		err = d.Deploy(context.Background(), &gw)
		assert.Error(t, err)
		assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
		assert.Equal(t, gw.NameContractID, nameContractID)
		assert.Equal(t, d.tfPluginClient.State.CurrentNodeDeployments, map[uint32]state.ContractIDs{nodeID: {contractID}})
	})

	t.Run("test cancel", func(t *testing.T) {
		d.tfPluginClient.State.CurrentNodeDeployments = map[uint32]state.ContractIDs{nodeID: {contractID}}

		gw.NameContractID = nameContractID
		gw.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}

		mockValidation(d.tfPluginClient.Identity, cl, sub, ncPool, proxyCl)

		deployer.EXPECT().Cancel(
			gomock.Any(),
			contractID,
		).Return(nil)

		sub.EXPECT().
			EnsureContractCanceled(d.tfPluginClient.Identity, nameContractID).
			Return(nil)

		err := d.Cancel(context.Background(), &gw)
		assert.NoError(t, err)
		assert.Empty(t, gw.NodeDeploymentID)
		assert.Empty(t, d.tfPluginClient.State.CurrentNodeDeployments[nodeID])
		assert.Equal(t, gw.NameContractID, uint64(0))
	})

	t.Run("test cancel failed", func(t *testing.T) {
		d.tfPluginClient.State.CurrentNodeDeployments = map[uint32]state.ContractIDs{nodeID: {contractID}}
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

	t.Run("test cancel contracts failed", func(t *testing.T) {
		d.tfPluginClient.State.CurrentNodeDeployments = map[uint32]state.ContractIDs{nodeID: {contractID}}

		gw := constructTestName()
		gw.NameContractID = nameContractID
		gw.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}

		mockValidation(d.tfPluginClient.Identity, cl, sub, ncPool, proxyCl)

		deployer.EXPECT().Cancel(
			gomock.Any(),
			contractID,
		).Return(nil)

		sub.EXPECT().
			EnsureContractCanceled(d.tfPluginClient.Identity, nameContractID).
			Return(errors.New("error"))

		err := d.Cancel(context.Background(), &gw)
		assert.Error(t, err)
		assert.Empty(t, gw.NodeDeploymentID)
		assert.Empty(t, d.tfPluginClient.State.CurrentNodeDeployments[nodeID])
		assert.Equal(t, gw.NameContractID, nameContractID)
	})

	t.Run("test sync contracts", func(t *testing.T) {
		gw.ContractID = contractID
		gw.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}

		sub.EXPECT().DeleteInvalidContracts(
			gw.NodeDeploymentID,
		).Return(nil)

		sub.EXPECT().IsValidContract(
			gw.NameContractID,
		).Return(true, nil)

		err := d.syncContracts(context.Background(), &gw)
		assert.NoError(t, err)
		assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
		assert.Equal(t, gw.ContractID, contractID)
	})

	t.Run("test sync deleted contracts", func(t *testing.T) {
		gw.ContractID = contractID
		gw.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}

		sub.EXPECT().DeleteInvalidContracts(
			gw.NodeDeploymentID,
		).DoAndReturn(func(contracts map[uint32]uint64) error {
			delete(contracts, nodeID)
			return nil
		})

		sub.EXPECT().IsValidContract(
			gw.NameContractID,
		).Return(false, nil)

		err := d.syncContracts(context.Background(), &gw)
		assert.NoError(t, err)
		assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{})
		assert.Equal(t, gw.NameContractID, uint64(0))
		assert.Equal(t, gw.ContractID, uint64(0))
	})

	t.Run("test sync contracts failed", func(t *testing.T) {
		gw.ContractID = contractID
		gw.NameContractID = nameContractID
		gw.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}

		sub.EXPECT().DeleteInvalidContracts(
			gw.NodeDeploymentID,
		).Return(errors.New("error"))

		err := d.syncContracts(context.Background(), &gw)
		assert.Error(t, err)
		assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
		assert.Equal(t, gw.NameContractID, nameContractID)
		assert.Equal(t, gw.ContractID, contractID)
	})

	t.Run("test sync contracts", func(t *testing.T) {
		gw.ContractID = contractID
		gw.NameContractID = nameContractID
		gw.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}

		dls, err := d.GenerateVersionlessDeployments(context.Background(), &gw)
		assert.NoError(t, err)

		dl := dls[nodeID]

		dl.Workloads[0].Result.State = zosTypes.StateOk
		dl.Workloads[0].Result.Data, err = json.Marshal(zos.GatewayProxyResult{FQDN: "name.com"})
		assert.NoError(t, err)

		sub.EXPECT().DeleteInvalidContracts(
			gw.NodeDeploymentID,
		).Return(nil)

		sub.EXPECT().IsValidContract(
			gw.NameContractID,
		).Return(true, nil)

		deployer.EXPECT().
			GetDeployments(gomock.Any(), map[uint32]uint64{nodeID: contractID}).
			Return(map[uint32]zosTypes.Deployment{nodeID: dl}, nil)

		gw.FQDN = "123"
		err = d.Sync(context.Background(), &gw)
		assert.Equal(t, gw.FQDN, "name.com")
		assert.NoError(t, err)
		assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
		assert.Equal(t, gw.NameContractID, nameContractID)
		assert.Equal(t, gw.ContractID, contractID)
	})

	t.Run("test sync contracts", func(t *testing.T) {
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

		sub.EXPECT().IsValidContract(
			gw.NameContractID,
		).Return(true, nil)

		deployer.EXPECT().
			GetDeployments(gomock.Any(), map[uint32]uint64{nodeID: contractID}).
			Return(map[uint32]zosTypes.Deployment{nodeID: dl}, nil)

		gw.FQDN = "123"
		err = d.Sync(context.Background(), &gw)
		assert.NoError(t, err)
		assert.Empty(t, gw.Backends)
		assert.Empty(t, gw.TLSPassthrough)
		assert.Empty(t, gw.Name)
		assert.Empty(t, gw.FQDN)
		assert.Equal(t, gw.ContractID, contractID)
		assert.Equal(t, gw.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
	})
}

func ExampleGatewayNameDeployer_Deploy() {
	const mnemonic = "<mnemonics goes here>"
	const network = "<dev, test, qa, main>"
	const nodeID = 11 // use any node with status up, use ExampleFilterNodes to get valid nodeID

	tfPluginClient, err := NewTFPluginClient(mnemonic, WithNetwork(network))
	if err != nil {
		fmt.Println(err)
		return
	}
	g := workloads.GatewayNameProxy{
		NodeID:         nodeID,
		Name:           "test",
		TLSPassthrough: false,
		Backends:       []zos.Backend{"http://1.1.1.1"},
		FQDN:           "test.com",
	}

	err = tfPluginClient.GatewayNameDeployer.Deploy(context.Background(), &g)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("deployment is done successfully")
}

func ExampleGatewayNameDeployer_BatchDeploy() {
	const mnemonic = "<mnemonics goes here>"
	const network = "<dev, test, qa, main>"
	const nodeID = 11 // use any node with status up, use ExampleFilterNodes to get valid nodeID

	tfPluginClient, err := NewTFPluginClient(mnemonic, WithNetwork(network))
	if err != nil {
		fmt.Println(err)
		return
	}
	g1 := workloads.GatewayNameProxy{
		NodeID:         nodeID,
		Name:           "test1",
		TLSPassthrough: false,
		Backends:       []zos.Backend{"http://1.1.1.1"},
		FQDN:           "test1.com",
	}
	g2 := workloads.GatewayNameProxy{
		NodeID:         nodeID,
		Name:           "test2",
		TLSPassthrough: false,
		Backends:       []zos.Backend{"http://2.2.2.2"},
		FQDN:           "test2.com",
	}

	err = tfPluginClient.GatewayNameDeployer.BatchDeploy(context.Background(), []*workloads.GatewayNameProxy{&g1, &g2})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("batch deployment is done successfully")
}

func ExampleGatewayNameDeployer_Cancel() {
	const mnemonic = "<mnemonics goes here>"
	const network = "<dev, test, qa, main>"
	const nodeID = 11 // use any node with status up, use ExampleFilterNodes to get valid nodeID

	tfPluginClient, err := NewTFPluginClient(mnemonic, WithNetwork(network))
	if err != nil {
		fmt.Println(err)
		return
	}

	// should be a valid and existing name and deploymentName
	name := "test.com"
	deploymentName := "test"
	g, err := tfPluginClient.State.LoadGatewayNameFromGrid(context.Background(), nodeID, name, deploymentName)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = tfPluginClient.GatewayNameDeployer.Cancel(context.Background(), &g)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("deployment is canceled successfully")
}
