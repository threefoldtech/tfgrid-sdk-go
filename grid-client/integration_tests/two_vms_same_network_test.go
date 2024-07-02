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

	network := generateBasicNetwork([]uint32{nodeID})

	vm1 := workloads.VM{
		Name:        "vm1",
		NetworkName: network.Name,
		CPU:         minCPU,
		Memory:      int(minMemory) * 1024,
		RootfsSize:  int(minRootfs) * 1024,
		PublicIP6:   true,
		Planetary:   true,
		Flist:       "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-22.04.flist",
		Entrypoint:  "/sbin/zinit init",
		EnvVars: map[string]string{
			"SSH_KEY": publicKey,
		},
	}

	vm2 := workloads.VM{
		Name:        "vm2",
		NetworkName: network.Name,
		CPU:         minCPU,
		Memory:      int(minMemory) * 1024,
		RootfsSize:  int(minRootfs) * 1024,
		PublicIP6:   true,
		Planetary:   true,
		Flist:       "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-22.04.flist",
		Entrypoint:  "/sbin/zinit init",
		EnvVars: map[string]string{
			"SSH_KEY": publicKey,
		},
	}

	err = tfPluginClient.NetworkDeployer.Deploy(context.Background(), &network)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.NetworkDeployer.Cancel(context.Background(), &network)
		require.NoError(t, err)
	})

	dl := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID, "", nil, network.Name, nil, nil, []workloads.VM{vm1, vm2}, nil, nil)
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

	yggIP1 := v1.PlanetaryIP
	yggIP2 := v2.PlanetaryIP

	require.NotEmpty(t, yggIP1)
	require.NotEmpty(t, yggIP2)

	_, err = RemoteRun("root", yggIP1, "apt install -y netcat", privateKey)
	require.NoError(t, err)

	_, err = RemoteRun("root", yggIP2, "apt install -y netcat", privateKey)
	require.NoError(t, err)

	// check yggIP2 from vm1
	_, err = RemoteRun("root", yggIP1, fmt.Sprintf("nc -z %s 22", yggIP2), privateKey)
	require.NoError(t, err)

	// check yggIP1 from vm2
	_, err = RemoteRun("root", yggIP2, fmt.Sprintf("nc -z %s 22", yggIP1), privateKey)
	require.NoError(t, err)

	privateIP1 := v1.IP
	privateIP2 := v2.IP

	require.NotEmpty(t, privateIP1)
	require.NotEmpty(t, privateIP2)
	require.NotEqual(t, privateIP1, privateIP2)

	// check privateIP2 from vm1
	_, err = RemoteRun("root", yggIP1, fmt.Sprintf("nc -z %s 22", privateIP2), privateKey)
	require.NoError(t, err)

	// check privateIP1 from vm2
	_, err = RemoteRun("root", yggIP2, fmt.Sprintf("nc -z %s 22", privateIP1), privateKey)
	require.NoError(t, err)

	publicIP6_1 := strings.Split(v1.ComputedIP6, "/")[0]
	publicIP6_2 := strings.Split(v2.ComputedIP6, "/")[0]

	require.NotEmpty(t, publicIP6_1)
	require.NotEmpty(t, publicIP6_2)

	// check publicIP62 from vm1
	_, err = RemoteRun("root", yggIP1, fmt.Sprintf("nc -z %s 22", publicIP6_2), privateKey)
	require.NoError(t, err)

	// check publicIP61 from vm2
	_, err = RemoteRun("root", yggIP2, fmt.Sprintf("nc -z %s 22", publicIP6_1), privateKey)
	require.NoError(t, err)
}
