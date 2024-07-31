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

func TestVMWithVolume(t *testing.T) {
	tfPluginClient, err := setup()
	if err != nil {
		t.Skipf("plugin creation failed: %v", err)
	}

	publicKey, privateKey, err := GenerateSSHKeyPair()
	require.NoError(t, err)

	nodes, err := deployer.FilterNodes(
		context.Background(),
		tfPluginClient,
		generateNodeFilter(WithFreeSRU(3)),
		[]uint64{*convertGBToBytes(1)},
		nil,
		[]uint64{*convertGBToBytes(minRootfs)},
		1,
	)
	if err != nil {
		t.Skipf("no available nodes found: %v", err)
	}

	nodeID := uint32(nodes[0].NodeID)

	network := generateBasicNetwork([]uint32{nodeID})

	volume := workloads.Volume{
		Name:   "volume",
		SizeGB: 1,
	}

	vm := workloads.VM{
		Name:        "vm",
		NetworkName: network.Name,
		CPU:         minCPU,
		Memory:      int(minMemory) * 1024,
		RootfsSize:  int(minRootfs) * 1024,
		Planetary:   true,
		Flist:       "https://hub.grid.tf/tf-official-apps/base:latest.flist",
		Entrypoint:  "/sbin/zinit init",
		EnvVars: map[string]string{
			"SSH_KEY": publicKey,
		},
		Mounts: []workloads.Mount{
			{DiskName: volume.Name, MountPoint: "/volume"},
		},
	}

	err = tfPluginClient.NetworkDeployer.Deploy(context.Background(), &network)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.NetworkDeployer.Cancel(context.Background(), &network)
		require.NoError(t, err)
	})

	dl := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID, "", nil, network.Name, nil, nil, []workloads.VM{vm}, nil, []workloads.Volume{volume})
	err = tfPluginClient.DeploymentDeployer.Deploy(context.Background(), &dl)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(context.Background(), &dl)
		require.NoError(t, err)
	})

	v, err := tfPluginClient.State.LoadVMFromGrid(context.Background(), nodeID, vm.Name, dl.Name)
	require.NoError(t, err)
	require.NotEmpty(t, v.PlanetaryIP)

	resVolume, err := tfPluginClient.State.LoadVolumeFromGrid(context.Background(), nodeID, volume.Name, dl.Name)
	require.NoError(t, err)
	require.Equal(t, volume, resVolume)
	res, err := RemoteRun("root", v.PlanetaryIP, "mount", privateKey)
	require.NoError(t, err)
	strings.Contains(res, "volume on /volume type virtiofs")
}
