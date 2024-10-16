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
	zosTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
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
	tfPluginClient.State.Networks = state.NetworkState{
		State: map[string]state.Network{net.Name: {
			Subnets: map[uint32]string{nodeID: net.IPRange.String()},
		}},
	}

	return tfPluginClient.K8sDeployer, cl, sub, ncPool, deployer, gridProxyCl
}

func k8sMockValidation(identity substrate.Identity, cl *mocks.RMBMockClient, sub *mocks.MockSubstrateExt, ncPool *mocks.MockNodeClientGetter, proxyCl *mocks.MockClient, d K8sDeployer) {
	sub.EXPECT().
		GetBalance(d.tfPluginClient.Identity).
		Return(substrate.Balance{
			Free: types.U128{
				Int: big.NewInt(20000000),
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
	flistCheckSum, err := workloads.GetFlistChecksum(workloads.K8sFlist)
	if err != nil {
		return workloads.K8sCluster{}, err
	}

	master := workloads.K8sNode{
		VM: &workloads.VM{
			Name:        "K8sForTesting",
			NetworkName: "network",
			NodeID:      nodeID,
			PublicIP:    true,
			PublicIP6:   true,
			Planetary:   true,
			ComputedIP:  "5.5.5.5/24",
			ComputedIP6: "::7/64",
			PlanetaryIP: "::8/64",
			IP:          "10.1.0.2",
			CPU:         2,
			MemoryMB:    1024,
		},
		DiskSizeGB: 5,
	}

	worker := workloads.K8sNode{
		VM: &workloads.VM{
			Name:        "worker1",
			NetworkName: "network",
			NodeID:      nodeID,
			PublicIP:    true,
			PublicIP6:   true,
			Planetary:   true,
			ComputedIP:  "",
			ComputedIP6: "",
			PlanetaryIP: "",
			IP:          "",
			CPU:         2,
			MemoryMB:    1024,
		},
		DiskSizeGB: 5,
	}
	workers := []workloads.K8sNode{worker}
	Cluster := workloads.K8sCluster{
		Master:        &master,
		Workers:       workers[:],
		Token:         "tokens",
		SSHKey:        "",
		NetworkName:   "network",
		Flist:         workloads.K8sFlist,
		FlistChecksum: flistCheckSum,
		NodesIPRange:  make(map[uint32]gridtypes.IPNet),
	}
	return Cluster, nil
}

func TestK8sDeployer(t *testing.T) {
	d, cl, sub, ncPool, deployer, proxyCl := constructTestK8s(t, true)
	k8sCluster, err := constructK8sCluster()
	assert.NoError(t, err)

	t.Run("test validate master reachable", func(t *testing.T) {
		k8sMockValidation(d.tfPluginClient.Identity, cl, sub, ncPool, proxyCl, d)

		assignNodesFlistsAndEntryPoints(&k8sCluster)
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
		nodeWorkloads[k8sCluster.Master.NodeID] = append(nodeWorkloads[k8sCluster.Master.NodeID], masterWorkloads...)
		for _, w := range k8sCluster.Workers {
			workerWorkloads := w.WorkerZosWorkload(&k8sCluster)
			nodeWorkloads[w.NodeID] = append(nodeWorkloads[w.NodeID], workerWorkloads...)
		}

		wl := nodeWorkloads[nodeID]
		var zosWls []zosTypes.Workload
		for _, w := range wl {
			zosWls = append(zosWls, zosTypes.NewWorkloadFromZosWorkload(w))
		}
		testDl := workloads.NewGridDeployment(d.tfPluginClient.TwinID, 0, zosWls)
		testDl.Metadata = "{\"version\":3,\"type\":\"kubernetes\",\"name\":\"K8sForTesting\",\"projectName\":\"kubernetes/K8sForTesting\"}"

		assert.Equal(t, dls, map[uint32]zosTypes.Deployment{
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
		newDeploymentsSolutionProvider[k8sCluster.Master.NodeID] = nil

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

	flistCheckSum, err := workloads.GetFlistChecksum(workloads.K8sFlist)
	if err != nil {
		fmt.Println(err)
		return
	}

	tfPluginClient, err := NewTFPluginClient(mnemonic, WithNetwork(network))
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

	master := workloads.K8sNode{
		VM: &workloads.VM{
			Name:     "K8sForTesting",
			NodeID:   nodeID,
			CPU:      2,
			MemoryMB: 1024,
		},
		DiskSizeGB: 5,
	}

	worker := workloads.K8sNode{
		VM: &workloads.VM{
			Name:     "worker1",
			NodeID:   nodeID,
			CPU:      2,
			MemoryMB: 1024,
		},
		DiskSizeGB: 5,
	}

	cluster := workloads.K8sCluster{
		Master:        &master,
		Workers:       []workloads.K8sNode{worker},
		Token:         "tokens",
		SSHKey:        "<ssh key goes here>",
		NetworkName:   n.Name,
		Flist:         workloads.K8sFlist,
		FlistChecksum: flistCheckSum,
		NodesIPRange:  make(map[uint32]gridtypes.IPNet),
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

	flistCheckSum, err := workloads.GetFlistChecksum(workloads.K8sFlist)
	if err != nil {
		fmt.Println(err)
		return
	}

	tfPluginClient, err := NewTFPluginClient(mnemonic, WithNetwork(network))
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

	master := workloads.K8sNode{
		VM: &workloads.VM{
			Name:     "mr1",
			NodeID:   nodeID,
			CPU:      2,
			MemoryMB: 1024,
		},
		DiskSizeGB: 5,
	}

	worker := workloads.K8sNode{
		VM: &workloads.VM{
			Name:     "worker1",
			NodeID:   nodeID,
			CPU:      2,
			MemoryMB: 1024,
		},
		DiskSizeGB: 5,
	}

	cluster1 := workloads.K8sCluster{
		Master:        &master,
		Workers:       []workloads.K8sNode{worker},
		Token:         "tokens",
		SSHKey:        "<ssh key goes here>",
		NetworkName:   n.Name,
		Flist:         workloads.K8sFlist,
		FlistChecksum: flistCheckSum,
		NodesIPRange:  make(map[uint32]gridtypes.IPNet),
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
	const nodeID = 11 // use any node with status up, use ExampleFilterNodes to get valid nodeID

	tfPluginClient, err := NewTFPluginClient(mnemonic, WithNetwork(network))
	if err != nil {
		fmt.Println(err)
		return
	}

	// should be a valid and existing k8s cluster deployment name
	deploymentName := "K8sForTesting"
	cluster, err := tfPluginClient.State.LoadK8sFromGrid(context.Background(), []uint32{nodeID}, deploymentName)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = tfPluginClient.K8sDeployer.Cancel(context.Background(), &cluster)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("deployment is canceled successfully")
}
