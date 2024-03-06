package integration

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

func requireNodesAreReady(t *testing.T, k8sCluster *workloads.K8sCluster, privateKey string) {
	t.Helper()

	masterYggIP := k8sCluster.Master.PlanetaryIP
	require.NotEmpty(t, masterYggIP)

	// Check that the outputs not empty
	time.Sleep(40 * time.Second)
	output, err := RemoteRun("root", masterYggIP, "export KUBECONFIG=/etc/rancher/k3s/k3s.yaml && kubectl get node", privateKey)
	output = strings.TrimSpace(output)
	require.NoError(t, err)

	nodesNumber := reflect.ValueOf(k8sCluster.Workers).Len() + 1
	fmt.Printf("output: %v\n", output)
	numberOfReadyNodes := strings.Count(output, "Ready")
	require.True(t, numberOfReadyNodes == nodesNumber, "number of ready nodes is not equal to number of nodes only %d nodes are ready", numberOfReadyNodes)
}

func TestK8sDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	publicKey, privateKey, err := GenerateSSHKeyPair()
	require.NoError(t, err)

	nodes, err := deployer.FilterNodes(
		ctx,
		tfPluginClient,
		generateNodeFilter(WithFreeSRU(3), WithFreeMRU(*convertGBToBytes(3 * minMemory))),
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

	network := generateBasicNetwork([]uint32{masterNodeID, workerNodeID})

	err = tfPluginClient.NetworkDeployer.Deploy(ctx, &network)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.NetworkDeployer.Cancel(ctx, &network)
		require.NoError(t, err)
	})

	k8sFlist := "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist"

	master := workloads.K8sNode{
		Name:      fmt.Sprintf("master_%s", generateRandString(5)),
		Node:      masterNodeID,
		DiskSize:  1,
		CPU:       minCPU,
		Memory:    int(minMemory) * 1024,
		Planetary: true,
		Flist:     k8sFlist,
	}

	workerNodeData1 := workloads.K8sNode{
		Name:      fmt.Sprintf("worker1_%s", generateRandString(5)),
		Node:      workerNodeID,
		DiskSize:  1,
		CPU:       minCPU,
		Memory:    int(minMemory) * 1024,
		Planetary: true,
		Flist:     k8sFlist,
	}

	workerNodeData2 := workloads.K8sNode{
		Name:      fmt.Sprintf("worker2_%s", generateRandString(5)),
		Node:      workerNodeID,
		DiskSize:  1,
		CPU:       minCPU,
		Memory:    int(minMemory) * 1024,
		Planetary: true,
		Flist:     k8sFlist,
	}

	// deploy k8s cluster
	workers := []workloads.K8sNode{workerNodeData1, workerNodeData2}

	k8sCluster := workloads.K8sCluster{
		Master:      &master,
		Workers:     workers,
		Token:       "tokens",
		SSHKey:      publicKey,
		NetworkName: network.Name,
	}

	err = tfPluginClient.K8sDeployer.Deploy(ctx, &k8sCluster)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.K8sDeployer.Cancel(ctx, &k8sCluster)
		require.NoError(t, err)
	})

	k8s, err := tfPluginClient.State.LoadK8sFromGrid(ctx, []uint32{masterNodeID, workerNodeID}, k8sCluster.Master.Name)
	require.NoError(t, err)

	// check workers count
	require.Equal(t, len(k8s.Workers), 2)

	// Check that master is reachable
	masterIP := k8s.Master.PlanetaryIP
	require.NotEmpty(t, masterIP)
	require.NotEmpty(t, k8s.Workers[0].PlanetaryIP)
	require.NotEmpty(t, k8s.Workers[1].PlanetaryIP)

	require.True(t, TestConnection(k8s.Workers[0].PlanetaryIP, "22"))
	require.True(t, TestConnection(k8s.Workers[1].PlanetaryIP, "22"))

	require.NotEmpty(t, k8s.Master.IP)
	require.NotEmpty(t, k8s.Workers[0].IP)
	require.NotEmpty(t, k8s.Workers[1].IP)

	require.Equal(t, len(slices.Compact([]string{k8s.Master.IP, k8s.Workers[0].IP, k8s.Workers[1].IP})), 3)

	// ssh to master node
	requireNodesAreReady(t, &k8s, privateKey)

	//update k8s cluster (remove worker)
	k8sCluster.Workers = []workloads.K8sNode{workerNodeData1}

	err = tfPluginClient.K8sDeployer.Deploy(ctx, &k8sCluster)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.K8sDeployer.Cancel(ctx, &k8sCluster)
		require.NoError(t, err)
	})

	time.Sleep(10 * time.Second) // remove take some time to be reflected
	k8s, err = tfPluginClient.State.LoadK8sFromGrid(ctx, []uint32{masterNodeID, workerNodeID}, k8sCluster.Master.Name)
	require.NoError(t, err)

	// check workers count
	require.Equal(t, len(k8s.Workers), 1)

	// Check that master is reachable
	masterIP = k8s.Master.PlanetaryIP
	require.NotEmpty(t, masterIP)
	require.NotEmpty(t, k8s.Workers[0].PlanetaryIP)

	require.True(t, TestConnection(k8s.Workers[0].PlanetaryIP, "22"))

	// ssh to master node
	requireNodesAreReady(t, &k8s, privateKey)

	//update k8s cluster (add worker)
	k8sCluster.Workers = append(k8sCluster.Workers, workerNodeData2)
	err = tfPluginClient.K8sDeployer.Deploy(ctx, &k8sCluster)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.K8sDeployer.Cancel(ctx, &k8sCluster)
		require.NoError(t, err)
	})

	k8s, err = tfPluginClient.State.LoadK8sFromGrid(ctx, []uint32{masterNodeID, workerNodeID}, k8sCluster.Master.Name)
	require.NoError(t, err)
	require.Len(t, k8s.Workers, 2)

	masterIP = k8s.Master.PlanetaryIP
	require.NotEmpty(t, masterIP)
	require.NotEmpty(t, k8s.Workers[0].PlanetaryIP)
	require.NotEmpty(t, k8s.Workers[1].PlanetaryIP)

	require.True(t, TestConnection(k8s.Workers[0].PlanetaryIP, "22"))
	require.True(t, TestConnection(k8s.Workers[1].PlanetaryIP, "22"))

	requireNodesAreReady(t, &k8s, privateKey)
}
