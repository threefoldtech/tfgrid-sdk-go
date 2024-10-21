package integration

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/sethvargo/go-retry"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

func requireNodesAreReady(nodesNumber int, masterYggIP, privateKey string) error {
	// Check that the outputs not empty
	time.Sleep(40 * time.Second)

	var output string
	var err error
	if err := retry.Do(context.Background(), retry.WithMaxRetries(3, retry.NewConstant(1*time.Nanosecond)), func(ctx context.Context) error {
		output, err = RemoteRun("root", masterYggIP, "export KUBECONFIG=/etc/rancher/k3s/k3s.yaml && kubectl get node", privateKey)
		if err != nil {
			return err
		}

		output = strings.TrimSpace(output)
		fmt.Printf("output: %v\n", output)

		numberOfReadyNodes := strings.Count(output, "Ready")
		if numberOfReadyNodes != nodesNumber {
			return retry.RetryableError(fmt.Errorf("number of ready nodes is not equal to number of nodes only %d nodes are ready", numberOfReadyNodes))
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func TestK8sDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	if err != nil {
		t.Skipf("plugin creation failed: %v", err)
	}

	publicKey, _, err := GenerateSSHKeyPair()
	require.NoError(t, err)

	nodes, err := deployer.FilterNodes(
		context.Background(),
		tfPluginClient,
		generateNodeFilter(WithFreeSRU(3), WithFreeMRU(3*minMemory)),
		[]uint64{*convertGBToBytes(1), *convertGBToBytes(1), *convertGBToBytes(1)},
		nil,
		nil,
		2,
	)
	if err != nil {
		t.Skipf("no available nodes found: %v", err)
	}

	masterNodeID := uint32(nodes[0].NodeID)
	workerNodeID := uint32(nodes[1].NodeID)

	network, err := generateBasicNetwork([]uint32{masterNodeID, workerNodeID})
	require.NoError(t, err)

	err = tfPluginClient.NetworkDeployer.Deploy(context.Background(), &network)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.NetworkDeployer.Cancel(context.Background(), &network)
		require.NoError(t, err)
	})

	masterVM, err := generateBasicVM(fmt.Sprintf("master_%s", generateRandString(5)), masterNodeID, network.Name, "")
	require.NoError(t, err)

	master := workloads.K8sNode{
		VM:         &masterVM,
		DiskSizeGB: 1,
	}

	vm1, err := generateBasicVM(fmt.Sprintf("worker1_%s", generateRandString(5)), workerNodeID, network.Name, "")
	require.NoError(t, err)

	worker1NodeData := workloads.K8sNode{
		VM:         &vm1,
		DiskSizeGB: 1,
	}
	vm2, err := generateBasicVM(fmt.Sprintf("worker2_%s", generateRandString(5)), workerNodeID, network.Name, "")
	require.NoError(t, err)

	worker2NodeData := workloads.K8sNode{
		VM:         &vm2,
		DiskSizeGB: 1,
	}

	// deploy k8s cluster
	workers := []workloads.K8sNode{worker1NodeData, worker2NodeData}

	k8sCluster := workloads.K8sCluster{
		Master:      &master,
		Workers:     workers,
		Token:       "tokens",
		SSHKey:      publicKey,
		Flist:       workloads.K8sFlist,
		NetworkName: network.Name,
	}

	err = tfPluginClient.K8sDeployer.Deploy(context.Background(), &k8sCluster)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.K8sDeployer.Cancel(context.Background(), &k8sCluster)
		require.NoError(t, err)
	})

	k8s, err := tfPluginClient.State.LoadK8sFromGrid(context.Background(), []uint32{masterNodeID, workerNodeID}, k8sCluster.Master.Name)
	require.NoError(t, err)

	// check workers count
	require.Equal(t, len(k8s.Workers), 2)

	// Check that master is reachable
	masterIP := k8s.Master.MyceliumIP
	require.NotEmpty(t, masterIP)
	require.NotEmpty(t, k8s.Workers[0].MyceliumIP)
	require.NotEmpty(t, k8s.Workers[1].MyceliumIP)

	require.True(t, CheckConnection(k8s.Workers[0].MyceliumIP, "22"))
	require.True(t, CheckConnection(k8s.Workers[1].MyceliumIP, "22"))

	require.NotEmpty(t, k8s.Master.IP)
	require.NotEmpty(t, k8s.Workers[0].IP)
	require.NotEmpty(t, k8s.Workers[1].IP)

	require.Equal(t, len(slices.Compact([]string{k8s.Master.IP, k8s.Workers[0].IP, k8s.Workers[1].IP})), 3)

	// ssh to master node //TODO:
	// require.NoError(t, requireNodesAreReady(len(k8s.Workers)+1, masterIP, privateKey))

	// update k8s cluster (remove worker)
	k8sCluster.Workers = []workloads.K8sNode{worker1NodeData}

	err = tfPluginClient.K8sDeployer.Deploy(context.Background(), &k8sCluster)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.K8sDeployer.Cancel(context.Background(), &k8sCluster)
		require.NoError(t, err)
	})

	time.Sleep(10 * time.Second) // remove take some time to be reflected
	k8s, err = tfPluginClient.State.LoadK8sFromGrid(context.Background(), []uint32{masterNodeID, workerNodeID}, k8sCluster.Master.Name)
	require.NoError(t, err)

	// check workers count
	require.Equal(t, len(k8s.Workers), 1)

	// Check that master is reachable
	masterIP = k8s.Master.MyceliumIP
	require.NotEmpty(t, masterIP)
	require.NotEmpty(t, k8s.Workers[0].MyceliumIP)

	require.True(t, CheckConnection(k8s.Workers[0].MyceliumIP, "22"))

	// ssh to master node
	// require.NoError(t, requireNodesAreReady(len(k8s.Workers)+1, masterIP, privateKey))

	// update k8s cluster (add worker)
	k8sCluster.Workers = append(k8sCluster.Workers, worker2NodeData)
	err = tfPluginClient.K8sDeployer.Deploy(context.Background(), &k8sCluster)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.K8sDeployer.Cancel(context.Background(), &k8sCluster)
		require.NoError(t, err)
	})

	k8s, err = tfPluginClient.State.LoadK8sFromGrid(context.Background(), []uint32{masterNodeID, workerNodeID}, k8sCluster.Master.Name)
	require.NoError(t, err)
	require.Len(t, k8s.Workers, 2)

	masterIP = k8s.Master.MyceliumIP
	require.NotEmpty(t, masterIP)
	require.NotEmpty(t, k8s.Workers[0].MyceliumIP)
	require.NotEmpty(t, k8s.Workers[1].MyceliumIP)

	require.True(t, CheckConnection(k8s.Workers[0].MyceliumIP, "22"))
	require.True(t, CheckConnection(k8s.Workers[1].MyceliumIP, "22"))

	// require.NoError(t, requireNodesAreReady(len(k8s.Workers)+1, masterIP, privateKey))
}
