// Package integration for integration tests
package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

func TestBatchVMDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	publicKey, privateKey, err := GenerateSSHKeyPair()
	require.NoError(t, err)

	nodes, err := deployer.FilterNodes(
		ctx,
		tfPluginClient,
		generateNodeFilter(),
		nil,
		nil,
		nil,
		2,
	)
	if err != nil {
		t.Skipf("no available nodes found: %v", err)
	}

	nodeID1 := uint32(nodes[0].NodeID)
	nodeID2 := uint32(nodes[1].NodeID)

	network1 := generateBasicNetwork([]uint32{nodeID1})
	network2 := generateBasicNetwork([]uint32{nodeID2})

	vm1 := workloads.VM{
		Name:        "vm",
		NetworkName: network1.Name,
		CPU:         minCPU,
		Memory:      int(minMemory) * 1024,
		Planetary:   true,
		Flist:       "https://hub.grid.tf/tf-official-apps/base:latest.flist",
		Entrypoint:  "/sbin/zinit init",
		EnvVars: map[string]string{
			"SSH_KEY": publicKey,
		},
	}

	vm2 := workloads.VM{
		Name:        "vm",
		NetworkName: network2.Name,
		CPU:         minCPU,
		Memory:      int(minMemory) * 1024,
		Planetary:   true,
		Flist:       "https://hub.grid.tf/tf-official-apps/base:latest.flist",
		Entrypoint:  "/sbin/zinit init",
		EnvVars: map[string]string{
			"SSH_KEY": publicKey,
		},
	}

	err = tfPluginClient.NetworkDeployer.BatchDeploy(ctx, []*workloads.ZNet{&network1, &network2})
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.NetworkDeployer.Cancel(ctx, &network1)
		require.NoError(t, err)

		err = tfPluginClient.NetworkDeployer.Cancel(ctx, &network2)
		require.NoError(t, err)
	})

	dl1 := workloads.NewDeployment(fmt.Sprintf("dl1_%s", generateRandString(10)), nodeID1, "", nil, network1.Name, nil, nil, []workloads.VM{vm1}, nil)
	dl2 := workloads.NewDeployment(fmt.Sprintf("dl2_%s", generateRandString(10)), nodeID2, "", nil, network2.Name, nil, nil, []workloads.VM{vm2}, nil)
	err = tfPluginClient.DeploymentDeployer.BatchDeploy(ctx, []*workloads.Deployment{&dl1, &dl2})
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl1)
		require.NoError(t, err)

		err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl2)
		require.NoError(t, err)
	})

	v1, err := tfPluginClient.State.LoadVMFromGrid(ctx, nodeID1, vm1.Name, dl1.Name)
	require.NoError(t, err)
	require.NotEmpty(t, v1.PlanetaryIP)

	output, err := RemoteRun("root", v1.PlanetaryIP, "ls /", privateKey)
	require.NoError(t, err)
	require.Contains(t, output, "root")

	v2, err := tfPluginClient.State.LoadVMFromGrid(ctx, nodeID2, vm2.Name, dl2.Name)
	require.NoError(t, err)
	require.NotEmpty(t, v2.PlanetaryIP)

	output, err = RemoteRun("root", v2.PlanetaryIP, "ls /", privateKey)
	require.NoError(t, err)
	require.Contains(t, output, "root")
}
