// Package integration for integration tests
package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	node "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func ConvertGPUsToStr(gpus []node.GPU) (zosGPUs []zos.GPU) {
	for _, g := range gpus {
		zosGPUs = append(zosGPUs, zos.GPU(g.ID))
	}

	return
}

func TestVMWithGPUDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	if err != nil {
		t.Skipf("plugin creation failed: %v", err)
	}

	publicKey, privateKey, err := GenerateSSHKeyPair()
	require.NoError(t, err)

	nodes, err := deployer.FilterNodes(
		context.Background(),
		tfPluginClient,
		generateNodeFilter(WithFreeMRU(8), WithFreeSRU(20), WithGPU(), WithRentedBy(uint64(tfPluginClient.TwinID))),
		[]uint64{*convertGBToBytes(20)},
		nil,
		[]uint64{*convertGBToBytes(minRootfs)},
		1,
	)
	if err != nil {
		t.Skipf("no available nodes found: %v", err)
	}

	nodeID := uint32(nodes[0].NodeID)

	nodeClient, err := tfPluginClient.NcPool.GetNodeClient(tfPluginClient.SubstrateConn, nodeID)
	require.NoError(t, err)

	gpus, err := nodeClient.GPUs(context.Background())
	require.NoError(t, err)

	network := generateBasicNetwork([]uint32{nodeID})

	disk := workloads.Disk{
		Name:   "gpuDisk",
		SizeGB: 20,
	}

	vm := workloads.VM{
		Name:        "gpu",
		NetworkName: network.Name,
		CPU:         4,
		Memory:      1024 * 8,
		RootfsSize:  int(minRootfs) * 1024,
		Planetary:   true,
		GPUs:        ConvertGPUsToStr(gpus),
		Flist:       "https://hub.grid.tf/tf-official-vms/ubuntu-22.04.flist",
		Entrypoint:  "/init.sh",
		EnvVars: map[string]string{
			"SSH_KEY": publicKey,
		},
		Mounts: []workloads.Mount{
			{DiskName: disk.Name, MountPoint: "/data"},
		},
	}

	err = tfPluginClient.NetworkDeployer.Deploy(context.Background(), &network)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.NetworkDeployer.Cancel(context.Background(), &network)
		require.NoError(t, err)
	})

	dl := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID, "", nil, network.Name, []workloads.Disk{disk}, nil, []workloads.VM{vm}, nil, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(context.Background(), &dl)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(context.Background(), &dl)
		require.NoError(t, err)
	})

	vm, err = tfPluginClient.State.LoadVMFromGrid(context.Background(), nodeID, vm.Name, dl.Name)
	require.NoError(t, err)
	require.Equal(t, vm.GPUs, ConvertGPUsToStr(gpus))
	require.NotEmpty(t, vm.PlanetaryIP)

	time.Sleep(30 * time.Second)
	output, err := RemoteRun("root", vm.PlanetaryIP, "lspci -v", privateKey)
	require.NoError(t, err)
	require.Contains(t, output, gpus[0].Vendor)
}
