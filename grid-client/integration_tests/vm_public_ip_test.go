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
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	publicKey, privateKey, err := GenerateSSHKeyPair()
	require.NoError(t, err)

	nodes, err := deployer.FilterNodes(
		ctx,
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

	network := generateBasicNetwork([]uint32{nodeID})

	vm := workloads.VM{
		Name:        "vm",
		NetworkName: network.Name,
		IP:          "10.20.2.5",
		CPU:         minCPU,
		Memory:      int(minMemory) * 1024,
		RootfsSize:  int(minRootfs) * 1024,
		PublicIP:    true,
		Planetary:   true,
		Flist:       "https://hub.grid.tf/tf-official-apps/base:latest.flist",
		Entrypoint:  "/sbin/zinit init",
		EnvVars: map[string]string{
			"SSH_KEY": publicKey,
		},
	}

	err = tfPluginClient.NetworkDeployer.Deploy(ctx, &network)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.NetworkDeployer.Cancel(ctx, &network)
		require.NoError(t, err)
	})

	dl := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID, "", nil, network.Name, nil, nil, []workloads.VM{vm}, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(ctx, &dl)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl)
		require.NoError(t, err)
	})

	v, err := tfPluginClient.State.LoadVMFromGrid(ctx, nodeID, vm.Name, dl.Name)
	require.NoError(t, err)
	require.Equal(t, v.IP, "10.20.2.5")

	publicIP := strings.Split(v.ComputedIP, "/")[0]
	require.NotEmpty(t, publicIP)
	require.True(t, TestConnection(publicIP, "22"))

	output, err := RemoteRun("root", publicIP, "ls /", privateKey)
	require.NoError(t, err)
	require.Contains(t, output, "root")

	planetaryIP := v.PlanetaryIP
	require.NotEmpty(t, planetaryIP)
	require.True(t, TestConnection(planetaryIP, "22"))

	output, err = RemoteRun("root", planetaryIP, "ls /", privateKey)
	require.NoError(t, err)
	require.Contains(t, output, "root")
}
