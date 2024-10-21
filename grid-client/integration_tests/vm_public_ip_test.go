// Package integration for integration tests
package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

func TestVMDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	if err != nil {
		t.Skipf("plugin creation failed: %v", err)
	}

	publicKey, privateKey, err := GenerateSSHKeyPair()
	require.NoError(t, err)

	nodes, err := deployer.FilterNodes(
		context.Background(),
		tfPluginClient,
		generateNodeFilter(WithFreeIPs(1), WithIPV4()),
		nil, nil,
		[]uint64{*convertGBToBytes(minRootfs)},
		1,
	)
	if err != nil {
		t.Skipf("no available nodes found: %v", err)
	}

	nodeID := uint32(nodes[0].NodeID)

	network, err := generateBasicNetwork([]uint32{nodeID})
	require.NoError(t, err)

	vm, err := generateBasicVM("vm", nodeID, network.Name, publicKey)
	require.NoError(t, err)

	vm.IP = "10.20.2.5"
	vm.RootfsSizeMB = minRootfs * 1024
	vm.PublicIP = true

	err = tfPluginClient.NetworkDeployer.Deploy(context.Background(), &network)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.NetworkDeployer.Cancel(context.Background(), &network)
		require.NoError(t, err)
	})

	dl := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID, "", nil, network.Name, nil, nil, []workloads.VM{vm}, nil, nil, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(context.Background(), &dl)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(context.Background(), &dl)
		require.NoError(t, err)
	})

	v, err := tfPluginClient.State.LoadVMFromGrid(context.Background(), nodeID, vm.Name, dl.Name)
	require.NoError(t, err)
	require.Equal(t, v.IP, "10.20.2.5")

	publicIP := strings.Split(v.ComputedIP, "/")[0]
	require.NotEmpty(t, publicIP)
	// sometimes it fails because of assigning same previously used IPs
	if !CheckConnection(publicIP, "22") {
		time.Sleep(10 * time.Second)
	}
	require.True(t, CheckConnection(publicIP, "22"))

	output, err := RemoteRun("root", publicIP, "ls /", privateKey)
	require.NoError(t, err)
	require.Contains(t, output, "root")

	myCeliumIP := v.MyceliumIP
	require.NotEmpty(t, myCeliumIP)
	require.True(t, CheckConnection(myCeliumIP, "22"))

	output, err = RemoteRun("root", myCeliumIP, "ls /", privateKey)
	require.NoError(t, err)
	require.Contains(t, output, "root")
}
