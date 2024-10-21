// Package integration for integration tests
package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

func TestTwoVMsSameNetworkWithPublicIPV6(t *testing.T) {
	tfPluginClient, err := setup()
	if err != nil {
		t.Skipf("plugin creation failed: %v", err)
	}

	publicKey, privateKey, err := GenerateSSHKeyPair()
	require.NoError(t, err)

	nodes, err := deployer.FilterNodes(
		context.Background(),
		tfPluginClient,
		generateNodeFilter(WithFreeSRU(minRootfs), WithIPV6()),
		nil,
		nil,
		[]uint64{*convertGBToBytes(minRootfs), *convertGBToBytes(minRootfs)},
		1,
	)
	if err != nil {
		t.Skipf("no available nodes found: %v", err)
	}

	nodeID := uint32(nodes[0].NodeID)

	network, err := generateBasicNetwork([]uint32{nodeID})
	require.NoError(t, err)

	vm1, err := generateBasicVM("vm1", nodeID, network.Name, publicKey)
	require.NoError(t, err)

	vm1.Flist = ubuntuFlist
	vm1.RootfsSizeMB = minRootfs * 1024
	vm1.PublicIP6 = true

	vm2, err := generateBasicVM("vm2", nodeID, network.Name, publicKey)
	require.NoError(t, err)

	vm2.Flist = ubuntuFlist
	vm2.RootfsSizeMB = minRootfs * 1024
	vm2.PublicIP6 = true

	err = tfPluginClient.NetworkDeployer.Deploy(context.Background(), &network)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.NetworkDeployer.Cancel(context.Background(), &network)
		require.NoError(t, err)
	})

	dl := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID, "", nil, network.Name, nil, nil, []workloads.VM{vm1, vm2}, nil, nil, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(context.Background(), &dl)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(context.Background(), &dl)
		require.NoError(t, err)
	})

	v1, err := tfPluginClient.State.LoadVMFromGrid(context.Background(), nodeID, vm1.Name, dl.Name)
	require.NoError(t, err)

	v2, err := tfPluginClient.State.LoadVMFromGrid(context.Background(), nodeID, vm2.Name, dl.Name)
	require.NoError(t, err)

	myceliumIP1 := v1.MyceliumIP
	myceliumIP2 := v2.MyceliumIP

	require.NotEmpty(t, myceliumIP1)
	require.NotEmpty(t, myceliumIP2)

	_, err = RemoteRun("root", myceliumIP1, "apt install -y netcat", privateKey)
	require.NoError(t, err)

	_, err = RemoteRun("root", myceliumIP2, "apt install -y netcat", privateKey)
	require.NoError(t, err)

	// check myceliumIP2 from vm1
	_, err = RemoteRun("root", myceliumIP1, fmt.Sprintf("nc -z %s 22", myceliumIP2), privateKey)
	require.NoError(t, err)

	// check myceliumIP1 from vm2
	_, err = RemoteRun("root", myceliumIP2, fmt.Sprintf("nc -z %s 22", myceliumIP1), privateKey)
	require.NoError(t, err)

	privateIP1 := v1.IP
	privateIP2 := v2.IP

	require.NotEmpty(t, privateIP1)
	require.NotEmpty(t, privateIP2)
	require.NotEqual(t, privateIP1, privateIP2)

	// check privateIP2 from vm1
	_, err = RemoteRun("root", myceliumIP1, fmt.Sprintf("nc -z %s 22", privateIP2), privateKey)
	require.NoError(t, err)

	// check privateIP1 from vm2
	_, err = RemoteRun("root", myceliumIP2, fmt.Sprintf("nc -z %s 22", privateIP1), privateKey)
	require.NoError(t, err)

	publicIP6_1 := strings.Split(v1.ComputedIP6, "/")[0]
	publicIP6_2 := strings.Split(v2.ComputedIP6, "/")[0]

	require.NotEmpty(t, publicIP6_1)
	require.NotEmpty(t, publicIP6_2)

	// check publicIP62 from vm1
	_, err = RemoteRun("root", myceliumIP1, fmt.Sprintf("nc -z %s 22", publicIP6_2), privateKey)
	require.NoError(t, err)

	// check publicIP61 from vm2
	_, err = RemoteRun("root", myceliumIP2, fmt.Sprintf("nc -z %s 22", publicIP6_1), privateKey)
	require.NoError(t, err)
}
