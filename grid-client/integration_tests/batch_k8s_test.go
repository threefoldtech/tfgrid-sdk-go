package integration

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func TestBatchK8sDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	if !assert.NoError(t, err) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	publicKey, privateKey, err := GenerateSSHKeyPair()
	if !assert.NoError(t, err) {
		return
	}

	nodes, err := deployer.FilterNodes(
		ctx,
		tfPluginClient,
		nodeFilter,
		[]uint64{*convertGBToBytes(1), *convertGBToBytes(1)},
		nil,
		[]uint64{minRootfs, minRootfs},
	)
	if err != nil || len(nodes) < 2 {
		t.Skip("no available nodes found")
	}

	nodeID1 := uint32(nodes[0].NodeID)
	nodeID2 := uint32(nodes[1].NodeID)

	network := workloads.ZNet{
		Name:        "k8sTestingNetwork",
		Description: "network for testing",
		Nodes:       []uint32{nodeID1, nodeID2},
		IPRange: gridtypes.NewIPNet(net.IPNet{
			IP:   net.IPv4(10, 20, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}),
		AddWGAccess: true,
	}

	err = tfPluginClient.NetworkDeployer.Deploy(ctx, &network)
	if !assert.NoError(t, err) {
		return
	}

	defer func() {
		err = tfPluginClient.NetworkDeployer.Cancel(ctx, &network)
		assert.NoError(t, err)
	}()

	flist := "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist"
	flistCheckSum, err := workloads.GetFlistChecksum(flist)
	if !assert.NoError(t, err) {
		return
	}

	master1 := workloads.K8sNode{
		Name:          "K8sForTesting",
		Node:          nodeID1,
		DiskSize:      1,
		PublicIP:      false,
		PublicIP6:     false,
		Planetary:     true,
		Flist:         "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist",
		FlistChecksum: flistCheckSum,
		ComputedIP:    "",
		ComputedIP6:   "",
		YggIP:         "",
		IP:            "",
		CPU:           2,
		Memory:        1024,
	}

	master2 := workloads.K8sNode{
		Name:          "K8sForTesting2",
		Node:          nodeID2,
		DiskSize:      1,
		PublicIP:      false,
		PublicIP6:     false,
		Planetary:     true,
		Flist:         "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist",
		FlistChecksum: flistCheckSum,
		ComputedIP:    "",
		ComputedIP6:   "",
		YggIP:         "",
		IP:            "",
		CPU:           2,
		Memory:        1024,
	}

	workerNodeData1 := workloads.K8sNode{
		Name:          "worker1",
		Node:          nodeID1,
		DiskSize:      1,
		PublicIP:      false,
		PublicIP6:     false,
		Planetary:     false,
		Flist:         "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist",
		FlistChecksum: flistCheckSum,
		ComputedIP:    "",
		ComputedIP6:   "",
		YggIP:         "",
		IP:            "",
		CPU:           2,
		Memory:        1024,
	}

	workerNodeData2 := workloads.K8sNode{
		Name:          "worker2",
		Node:          nodeID2,
		DiskSize:      1,
		PublicIP:      false,
		PublicIP6:     false,
		Planetary:     false,
		Flist:         "https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist",
		FlistChecksum: flistCheckSum,
		ComputedIP:    "",
		ComputedIP6:   "",
		YggIP:         "",
		IP:            "",
		CPU:           2,
		Memory:        1024,
	}

	k8sCluster1 := workloads.K8sCluster{
		Master:      &master1,
		Workers:     []workloads.K8sNode{workerNodeData1},
		Token:       "tokens",
		SSHKey:      publicKey,
		NetworkName: network.Name,
	}

	k8sCluster2 := workloads.K8sCluster{
		Master:      &master2,
		Workers:     []workloads.K8sNode{workerNodeData2},
		Token:       "tokens",
		SSHKey:      publicKey,
		NetworkName: network.Name,
	}

	err = tfPluginClient.K8sDeployer.BatchDeploy(ctx, []*workloads.K8sCluster{&k8sCluster1, &k8sCluster2})
	if !assert.NoError(t, err) {
		return
	}

	defer func() {
		err = tfPluginClient.K8sDeployer.Cancel(ctx, &k8sCluster1)
		assert.NoError(t, err)

		err = tfPluginClient.K8sDeployer.Cancel(ctx, &k8sCluster2)
		assert.NoError(t, err)
	}()

	// cluster 1
	result, err := tfPluginClient.State.LoadK8sFromGrid([]uint32{nodeID1}, k8sCluster1.Master.Name)
	if !assert.NoError(t, err) {
		return
	}

	// check workers count
	if !assert.Equal(t, len(result.Workers), 1) {
		return
	}

	// Check that master is reachable
	masterIP := result.Master.YggIP
	if !assert.NotEmpty(t, masterIP) {
		return
	}

	// Check wireguard config in output
	wgConfig := network.AccessWGConfig
	if !assert.NotEmpty(t, wgConfig) {
		return
	}

	// ssh to master node
	if !AssertNodesAreReady(t, &result, privateKey) {
		return
	}

	// cluster 2
	result, err = tfPluginClient.State.LoadK8sFromGrid([]uint32{nodeID2}, k8sCluster2.Master.Name)
	if !assert.NoError(t, err) {
		return
	}

	// check workers count
	if !assert.Equal(t, len(result.Workers), 1) {
		return
	}

	// Check that master is reachable
	masterIP = result.Master.YggIP
	if !assert.NotEmpty(t, masterIP) {
		return
	}

	// Check wireguard config in output
	wgConfig = network.AccessWGConfig
	if !assert.NotEmpty(t, wgConfig) {
		return
	}

	// ssh to master node
	AssertNodesAreReady(t, &result, privateKey)
}
