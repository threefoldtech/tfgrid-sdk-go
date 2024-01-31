package integration

import (
	"context"
	"reflect"
	"strings"

	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func AssertNodesAreReady(t *testing.T, k8sCluster *workloads.K8sCluster, privateKey string) {
	t.Helper()

	masterYggIP := k8sCluster.Master.PlanetaryIP
	assert.NotEmpty(t, masterYggIP)

	// Check that the outputs not empty
	time.Sleep(30 * time.Second)
	output, err := RemoteRun("root", masterYggIP, "export KUBECONFIG=/etc/rancher/k3s/k3s.yaml && kubectl get node", privateKey)
	output = strings.TrimSpace(output)
	assert.Empty(t, err)

	nodesNumber := reflect.ValueOf(k8sCluster.Workers).Len() + 1
	numberOfReadyNodes := strings.Count(output, "Ready")
	assert.True(t, numberOfReadyNodes == nodesNumber, "number of ready nodes is not equal to number of nodes only %d nodes are ready", numberOfReadyNodes)
}

func TestK8sDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	publicKey, privateKey, err := GenerateSSHKeyPair()
	assert.NoError(t, err)

	nodes, err := deployer.FilterNodes(
		ctx,
		tfPluginClient,
		nodeFilter,
		[]uint64{*convertGBToBytes(1), *convertGBToBytes(1)},
		nil,
		[]uint64{minRootfs},
	)
	if err != nil || len(nodes) < 2 {
		t.Skip("no available nodes found")
	}

	masterNodeID := uint32(nodes[0].NodeID)
	workerNodeID := uint32(nodes[1].NodeID)

	network := workloads.ZNet{
		Name:        "k8sTestingNetwork",
		Description: "network for testing",
		Nodes:       []uint32{masterNodeID, workerNodeID},
		IPRange: gridtypes.NewIPNet(net.IPNet{
			IP:   net.IPv4(10, 20, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}),
		AddWGAccess: true,
	}

	err = tfPluginClient.NetworkDeployer.Deploy(ctx, &network)
	assert.NoError(t, err)

	defer func() {
		err = tfPluginClient.NetworkDeployer.Cancel(ctx, &network)
		assert.NoError(t, err)
	}()

	flist := "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist"
	flistCheckSum, err := workloads.GetFlistChecksum(flist)
	assert.NoError(t, err)

	master := workloads.K8sNode{
		Name:          "K8sForTesting",
		Node:          masterNodeID,
		DiskSize:      1,
		PublicIP:      false,
		PublicIP6:     false,
		Planetary:     true,
		Flist:         "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist",
		FlistChecksum: flistCheckSum,
		ComputedIP:    "",
		ComputedIP6:   "",
		PlanetaryIP:   "",
		IP:            "",
		CPU:           2,
		Memory:        1024,
	}

	workerNodeData1 := workloads.K8sNode{
		Name:          "worker1",
		Node:          workerNodeID,
		DiskSize:      1,
		PublicIP:      false,
		PublicIP6:     false,
		Planetary:     false,
		Flist:         "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist",
		FlistChecksum: flistCheckSum,
		ComputedIP:    "",
		ComputedIP6:   "",
		PlanetaryIP:   "",
		IP:            "",
		CPU:           2,
		Memory:        1024,
	}

	workerNodeData2 := workloads.K8sNode{
		Name:          "worker2",
		Node:          workerNodeID,
		DiskSize:      1,
		PublicIP:      false,
		PublicIP6:     false,
		Planetary:     false,
		Flist:         "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist",
		FlistChecksum: flistCheckSum,
		ComputedIP:    "",
		ComputedIP6:   "",
		PlanetaryIP:   "",
		IP:            "",
		CPU:           2,
		Memory:        1024,
	}

	t.Run("deploy k8s cluster", func(t *testing.T) {
		master := master
		workers := []workloads.K8sNode{workerNodeData1, workerNodeData2}

		k8sCluster := workloads.K8sCluster{
			Master:      &master,
			Workers:     workers,
			Token:       "tokens",
			SSHKey:      publicKey,
			NetworkName: network.Name,
		}

		err = tfPluginClient.K8sDeployer.Deploy(ctx, &k8sCluster)
		assert.NoError(t, err)

		defer func() {
			err = tfPluginClient.K8sDeployer.Cancel(ctx, &k8sCluster)
			assert.NoError(t, err)
		}()

		result, err := tfPluginClient.State.LoadK8sFromGrid([]uint32{masterNodeID, workerNodeID}, k8sCluster.Master.Name)
		assert.NoError(t, err)

		// check workers count
		assert.Equal(t, len(result.Workers), 2)

		// Check that master is reachable
		masterIP := result.Master.PlanetaryIP
		assert.NotEmpty(t, masterIP)

		// Check wireguard config in output
		wgConfig := network.AccessWGConfig
		assert.NotEmpty(t, wgConfig)

		// ssh to master node
		AssertNodesAreReady(t, &result, privateKey)
	})

	t.Run("update k8s cluster", func(t *testing.T) {
		workerNodeData1 := workerNodeData1
		workerNodeData2 := workerNodeData2

		workerNodeData1.Node = master.Node
		workers := []workloads.K8sNode{workerNodeData1}

		k8sCluster := workloads.K8sCluster{
			Master:      &master,
			Workers:     workers,
			Token:       "tokens",
			SSHKey:      publicKey,
			NetworkName: network.Name,
		}

		err = tfPluginClient.K8sDeployer.Deploy(ctx, &k8sCluster)
		assert.NoError(t, err)

		defer func() {
			err = tfPluginClient.K8sDeployer.Cancel(ctx, &k8sCluster)
			assert.NoError(t, err)
		}()

		result, err := tfPluginClient.State.LoadK8sFromGrid([]uint32{masterNodeID}, k8sCluster.Master.Name)
		assert.NoError(t, err)

		// check workers count
		assert.Equal(t, len(result.Workers), 1)

		// Check that master is reachable
		masterIP := result.Master.PlanetaryIP
		assert.NotEmpty(t, masterIP)

		// ssh to master node
		AssertNodesAreReady(t, &result, privateKey)

		// add another worker node
		k8sCluster.Workers = append(k8sCluster.Workers, workerNodeData2)
		err = tfPluginClient.K8sDeployer.Deploy(ctx, &k8sCluster)
		assert.NoError(t, err)

		result, err = tfPluginClient.State.LoadK8sFromGrid([]uint32{masterNodeID, workerNodeID}, k8sCluster.Master.Name)
		assert.NoError(t, err)

		assert.Len(t, result.Workers, 2)

		AssertNodesAreReady(t, &result, privateKey)
	})

}
