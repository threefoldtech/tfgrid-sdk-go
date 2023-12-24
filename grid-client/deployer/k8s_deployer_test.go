package deployer

import (
	"context"
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
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func constructTestK8s(t *testing.T, mock bool) (
	K8sDeployer,
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

		tfPluginClient.K8sDeployer.deployer = deployer
		tfPluginClient.K8sDeployer.tfPluginClient = &tfPluginClient
	}
	net := constructTestNetwork()
	tfPluginClient.State.Networks = state.NetworkState{net.Name: state.Network{
		Subnets:               map[uint32]string{nodeID: net.IPRange.String()},
		NodeDeploymentHostIDs: map[uint32]state.DeploymentHostIDs{nodeID: map[uint64][]byte{contractID: {}}},
	}}

	return tfPluginClient.K8sDeployer, cl, sub, ncPool, deployer, gridProxyCl
}

func k8sMockValidation(identity substrate.Identity, cl *mocks.RMBMockClient, sub *mocks.MockSubstrateExt, ncPool *mocks.MockNodeClientGetter, proxyCl *mocks.MockClient, d K8sDeployer) {
	sub.EXPECT().
		GetBalance(d.tfPluginClient.Identity).
		Return(substrate.Balance{
			Free: types.U128{
				Int: big.NewInt(100000),
			},
		}, nil)

	cl.EXPECT().
		Call(
			gomock.Any(),
			nodeID,
			"zos.system.version",
			nil,
			gomock.Any(),
		).Return(nil).AnyTimes()

	ncPool.EXPECT().
		GetNodeClient(
			gomock.Any(),
			nodeID,
		).Return(client.NewNodeClient(nodeID, cl, d.tfPluginClient.RMBTimeout), nil)
}

func constructK8sCluster() (workloads.K8sCluster, error) {
	flist := "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist"
	flistCheckSum, err := workloads.GetFlistChecksum(flist)
	if err != nil {
		return workloads.K8sCluster{}, err
	}

	master := workloads.K8sNode{
		Name:          "K8sForTesting",
		Node:          nodeID,
		DiskSize:      5,
		PublicIP:      true,
		PublicIP6:     true,
		Planetary:     true,
		Flist:         flist,
		FlistChecksum: flistCheckSum,
		ComputedIP:    "5.5.5.5/24",
		ComputedIP6:   "::7/64",
		YggIP:         "::8/64",
		IP:            "10.1.0.2",
		CPU:           2,
		Memory:        1024,
	}

	worker := workloads.K8sNode{
		Name:          "worker1",
		Node:          nodeID,
		DiskSize:      5,
		PublicIP:      true,
		PublicIP6:     true,
		Planetary:     true,
		Flist:         flist,
		FlistChecksum: flistCheckSum,
		ComputedIP:    "",
		ComputedIP6:   "",
		YggIP:         "",
		IP:            "",
		CPU:           2,
		Memory:        1024,
	}
	workers := []workloads.K8sNode{worker}
	Cluster := workloads.K8sCluster{
		Master:       &master,
		Workers:      workers[:],
		Token:        "tokens",
		SSHKey:       "",
		NetworkName:  "network",
		NodesIPRange: make(map[uint32]gridtypes.IPNet),
	}
	return Cluster, nil
}

func TestK8sDeployer(t *testing.T) {
	d, cl, sub, ncPool, deployer, proxyCl := constructTestK8s(t, true)
	k8sCluster, err := constructK8sCluster()
	assert.NoError(t, err)

	t.Run("test validate master reachable", func(t *testing.T) {
		k8sMockValidation(d.tfPluginClient.Identity, cl, sub, ncPool, proxyCl, d)

		err = d.tfPluginClient.State.AssignNodesIPRange(&k8sCluster)
		assert.NoError(t, err)

		err = d.Validate(context.Background(), &k8sCluster)
		assert.NoError(t, err)
	})

	t.Run("test generate", func(t *testing.T) {
		err = d.tfPluginClient.State.AssignNodesIPRange(&k8sCluster)
		assert.NoError(t, err)

		dls, err := d.GenerateVersionlessDeployments(context.Background(), &k8sCluster)
		assert.NoError(t, err)

		nodeWorkloads := make(map[uint32][]gridtypes.Workload)
		masterWorkloads := k8sCluster.Master.MasterZosWorkload(&k8sCluster)
		nodeWorkloads[k8sCluster.Master.Node] = append(nodeWorkloads[k8sCluster.Master.Node], masterWorkloads...)
		for _, w := range k8sCluster.Workers {
			workerWorkloads := w.WorkerZosWorkload(&k8sCluster)
			nodeWorkloads[w.Node] = append(nodeWorkloads[w.Node], workerWorkloads...)
		}

		wl := nodeWorkloads[nodeID]
		testDl := workloads.NewGridDeployment(d.tfPluginClient.TwinID, wl)
		testDl.Metadata = "{\"type\":\"kubernetes\",\"name\":\"K8sForTesting\",\"projectName\":\"Kubernetes\"}"

		assert.Equal(t, dls, map[uint32]gridtypes.Deployment{
			nodeID: testDl,
		})
	})

	t.Run("test deploy", func(t *testing.T) {
		err = d.tfPluginClient.State.AssignNodesIPRange(&k8sCluster)
		assert.NoError(t, err)

		dls, err := d.GenerateVersionlessDeployments(context.Background(), &k8sCluster)
		assert.NoError(t, err)

		k8sMockValidation(d.tfPluginClient.Identity, cl, sub, ncPool, proxyCl, d)

		newDeploymentsSolutionProvider := make(map[uint32]*uint64)
		newDeploymentsSolutionProvider[k8sCluster.Master.Node] = nil

		deployer.EXPECT().Deploy(
			gomock.Any(),
			k8sCluster.NodeDeploymentID,
			dls,
			newDeploymentsSolutionProvider,
		).Return(map[uint32]uint64{nodeID: contractID}, nil)

		err = d.Deploy(context.Background(), &k8sCluster)
		assert.NoError(t, err)

		assert.Equal(t, k8sCluster.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
	})

	t.Run("test update", func(t *testing.T) {
		k8sCluster.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}

		err = d.tfPluginClient.State.AssignNodesIPRange(&k8sCluster)
		assert.NoError(t, err)

		dls, err := d.GenerateVersionlessDeployments(context.Background(), &k8sCluster)
		assert.NoError(t, err)

		k8sMockValidation(d.tfPluginClient.Identity, cl, sub, ncPool, proxyCl, d)

		sub.EXPECT().GetContract(uint64(100)).Return(subi.Contract{
			Contract: &substrate.Contract{
				State: substrate.ContractState{IsCreated: true},
				ContractType: substrate.ContractType{
					NodeContract: substrate.NodeContract{
						Node:           types.U32(nodeID),
						PublicIPsCount: 0,
					},
				},
			},
		}, nil)

		deployer.EXPECT().Deploy(
			gomock.Any(),
			map[uint32]uint64{nodeID: contractID},
			dls,
			gomock.Any(),
		).Return(map[uint32]uint64{nodeID: contractID}, nil)

		err = d.Deploy(context.Background(), &k8sCluster)
		assert.NoError(t, err)
		assert.Equal(t, k8sCluster.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
		assert.Equal(t, d.tfPluginClient.State.CurrentNodeDeployments, map[uint32]state.ContractIDs{nodeID: {contractID}})
	})

	t.Run("test update failed", func(t *testing.T) {
		d.tfPluginClient.State.CurrentNodeDeployments = map[uint32]state.ContractIDs{nodeID: {contractID}}
		k8sCluster.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}

		err = d.tfPluginClient.State.AssignNodesIPRange(&k8sCluster)
		assert.NoError(t, err)

		dls, err := d.GenerateVersionlessDeployments(context.Background(), &k8sCluster)
		assert.NoError(t, err)

		k8sMockValidation(d.tfPluginClient.Identity, cl, sub, ncPool, proxyCl, d)

		sub.EXPECT().GetContract(uint64(100)).Return(subi.Contract{
			Contract: &substrate.Contract{
				State: substrate.ContractState{IsCreated: true},
				ContractType: substrate.ContractType{
					NodeContract: substrate.NodeContract{
						Node:           types.U32(nodeID),
						PublicIPsCount: 0,
					},
				},
			},
		}, nil)

		deployer.EXPECT().Deploy(
			gomock.Any(),
			map[uint32]uint64{nodeID: contractID},
			dls,
			gomock.Any(),
		).Return(map[uint32]uint64{nodeID: contractID}, errors.New("error"))

		err = d.Deploy(context.Background(), &k8sCluster)
		assert.Error(t, err)
		assert.Equal(t, k8sCluster.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
		assert.Equal(t, d.tfPluginClient.State.CurrentNodeDeployments, map[uint32]state.ContractIDs{nodeID: {contractID}})
	})

	t.Run("test cancel", func(t *testing.T) {
		d.tfPluginClient.State.CurrentNodeDeployments = map[uint32]state.ContractIDs{nodeID: {contractID}}
		k8sCluster.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}
		k8sCluster.NodesIPRange = map[uint32]gridtypes.IPNet{uint32(10): {}}

		k8sMockValidation(d.tfPluginClient.Identity, cl, sub, ncPool, proxyCl, d)

		deployer.EXPECT().Cancel(
			gomock.Any(), contractID,
		).Return(nil)

		err = d.Cancel(context.Background(), &k8sCluster)
		assert.NoError(t, err)
		assert.Empty(t, k8sCluster.NodeDeploymentID)
		assert.Empty(t, d.tfPluginClient.State.CurrentNodeDeployments[nodeID])
	})

	t.Run("test cancel failed", func(t *testing.T) {
		d.tfPluginClient.State.CurrentNodeDeployments = map[uint32]state.ContractIDs{nodeID: {contractID}}
		k8sCluster.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}
		k8sCluster.NodesIPRange = map[uint32]gridtypes.IPNet{uint32(10): {}}

		k8sMockValidation(d.tfPluginClient.Identity, cl, sub, ncPool, proxyCl, d)

		deployer.EXPECT().Cancel(
			gomock.Any(), contractID,
		).Return(errors.New("error"))

		err = d.Cancel(context.Background(), &k8sCluster)
		assert.Error(t, err)
		assert.Equal(t, k8sCluster.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
		assert.Equal(t, d.tfPluginClient.State.CurrentNodeDeployments, map[uint32]state.ContractIDs{nodeID: {contractID}})
	})
}

func ExampleK8sDeployer_Deploy() {
	const mnemonic = "<mnemonics goes here>"
	const network = "<dev, test, qa, main>"
	const nodeID = 11 // use any node with status up, use ExampleFilterNodes to get valid nodeID

	const flist = "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist"
	flistCheckSum, err := workloads.GetFlistChecksum(flist)
	if err != nil {
		fmt.Println(err)
		return
	}

	tfPluginClient, err := NewTFPluginClient(mnemonic, "sr25519", network, "", "", "", 0, false)
	if err != nil {
		fmt.Println(err)
		return
	}

	n := workloads.ZNet{
		Name:        "network",
		Description: "network for testing",
		Nodes:       []uint32{nodeID},
		IPRange: gridtypes.NewIPNet(net.IPNet{
			IP:   net.IPv4(10, 1, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}),
		AddWGAccess: false,
	}

	master := workloads.K8sNode{
		Name:          "K8sForTesting",
		Node:          nodeID,
		Flist:         flist,
		FlistChecksum: flistCheckSum,
		CPU:           2,
		DiskSize:      5,
		Memory:        1024,
	}

	worker := workloads.K8sNode{
		Name:          "worker1",
		Node:          nodeID,
		Flist:         flist,
		FlistChecksum: flistCheckSum,
		DiskSize:      5,
		CPU:           2,
		Memory:        1024,
	}

	cluster := workloads.K8sCluster{
		Master:       &master,
		Workers:      []workloads.K8sNode{worker},
		Token:        "tokens",
		SSHKey:       "<ssh key goes here>",
		NetworkName:  n.Name,
		NodesIPRange: make(map[uint32]gridtypes.IPNet),
	}

	err = tfPluginClient.NetworkDeployer.Deploy(context.Background(), &n)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = tfPluginClient.K8sDeployer.Deploy(context.Background(), &cluster)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("deployment done successfully")
}

func ExampleK8sDeployer_BatchDeploy() {
	const mnemonic = "<mnemonics goes here>"
	const network = "<dev, test, qa, main>"
	const nodeID = 11 // use any node with status up, use ExampleFilterNodes to get valid nodeID

	const flist = "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist"
	flistCheckSum, err := workloads.GetFlistChecksum(flist)
	if err != nil {
		fmt.Println(err)
		return
	}

	tfPluginClient, err := NewTFPluginClient(mnemonic, "sr25519", network, "", "", "", 0, false)
	if err != nil {
		fmt.Println(err)
		return
	}

	n := workloads.ZNet{
		Name:        "network",
		Description: "network for testing",
		Nodes:       []uint32{nodeID},
		IPRange: gridtypes.NewIPNet(net.IPNet{
			IP:   net.IPv4(10, 1, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}),
		AddWGAccess: false,
	}

	master := workloads.K8sNode{
		Name:          "mr1",
		Node:          nodeID,
		Flist:         flist,
		FlistChecksum: flistCheckSum,
		CPU:           2,
		DiskSize:      5,
		Memory:        1024,
	}

	worker := workloads.K8sNode{
		Name:          "worker1",
		Node:          nodeID,
		Flist:         flist,
		FlistChecksum: flistCheckSum,
		DiskSize:      5,
		CPU:           2,
		Memory:        1024,
	}

	cluster1 := workloads.K8sCluster{
		Master:       &master,
		Workers:      []workloads.K8sNode{worker},
		Token:        "tokens",
		SSHKey:       "<ssh key goes here>",
		NetworkName:  n.Name,
		NodesIPRange: make(map[uint32]gridtypes.IPNet),
	}

	master.Name = "mr2"
	worker.Name = "worker2"
	cluster2 := workloads.K8sCluster{
		Master:       &master,
		Workers:      []workloads.K8sNode{worker},
		Token:        "tokens",
		SSHKey:       "<ssh key goes here>",
		NetworkName:  n.Name,
		NodesIPRange: make(map[uint32]gridtypes.IPNet),
	}

	err = tfPluginClient.NetworkDeployer.Deploy(context.Background(), &n)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = tfPluginClient.K8sDeployer.BatchDeploy(context.Background(), []*workloads.K8sCluster{&cluster1, &cluster2})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("batch deployment is done successfully")
}

func ExampleK8sDeployer_Cancel() {
	const mnemonic = "<mnemonics goes here>"
	const network = "<dev, test, qa, main>"

	tfPluginClient, err := NewTFPluginClient(mnemonic, "sr25519", network, "", "", "", 0, false)
	if err != nil {
		fmt.Println(err)
		return
	}

	// should be a valid and existing k8s cluster
	cluster := workloads.K8sCluster{}

	err = tfPluginClient.K8sDeployer.Cancel(context.Background(), &cluster)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("deployment is canceled successfully")
}
