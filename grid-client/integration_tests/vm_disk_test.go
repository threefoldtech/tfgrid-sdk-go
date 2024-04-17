// Package integration for integration tests
package integration

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

func TestVMWithTwoDisk(t *testing.T) {
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
		[]uint64{*convertGBToBytes(2), *convertGBToBytes(1)},
		nil,
		[]uint64{*convertGBToBytes(minRootfs)},
		1,
	)
	if err != nil {
		t.Skipf("no available nodes found: %v", err)
	}

	nodeID := uint32(nodes[0].NodeID)

	network := generateBasicNetwork([]uint32{nodeID})

	disk1 := workloads.Disk{
		Name:   "diskTest1",
		SizeGB: 1,
	}
	disk2 := workloads.Disk{
		Name:   "diskTest2",
		SizeGB: 2,
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
			{DiskName: disk1.Name, MountPoint: "/disk1"},
			{DiskName: disk2.Name, MountPoint: "/disk2"},
		},
	}

	err = tfPluginClient.NetworkDeployer.Deploy(context.Background(), &network)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.NetworkDeployer.Cancel(context.Background(), &network)
		require.NoError(t, err)
	})

	dl := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID, "", nil, network.Name, []workloads.Disk{disk1, disk2}, nil, []workloads.VM{vm}, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(context.Background(), &dl)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(context.Background(), &dl)
		require.NoError(t, err)
	})

	v, err := tfPluginClient.State.LoadVMFromGrid(context.Background(), nodeID, vm.Name, dl.Name)
	require.NoError(t, err)
	require.NotEmpty(t, v.PlanetaryIP)

	resDisk1, err := tfPluginClient.State.LoadDiskFromGrid(context.Background(), nodeID, disk1.Name, dl.Name)
	require.NoError(t, err)
	require.Equal(t, disk1, resDisk1)

	resDisk2, err := tfPluginClient.State.LoadDiskFromGrid(context.Background(), nodeID, disk2.Name, dl.Name)
	require.NoError(t, err)
	require.Equal(t, disk2, resDisk2)

	// Check that disk has been mounted successfully
	output, err := RemoteRun("root", v.PlanetaryIP, "df -h | grep -w /disk1", privateKey)
	require.NoError(t, err)
	require.Contains(t, output, fmt.Sprintf("%d.0G", disk1.SizeGB))

	output, err = RemoteRun("root", v.PlanetaryIP, "df -h | grep -w /disk2", privateKey)
	require.NoError(t, err)
	require.Contains(t, output, fmt.Sprintf("%d.0G", disk2.SizeGB))

	// create file -> d1, check file size, move file -> d2, check file size

	_, err = RemoteRun("root", v.PlanetaryIP, "dd if=/dev/vda bs=1M count=512 of=/disk1/test.txt", privateKey)
	require.NoError(t, err)

	res, err := RemoteRun("root", v.PlanetaryIP, "du /disk1/test.txt | head -n1 | awk '{print $1;}' | tr -d -c 0-9", privateKey)
	require.NoError(t, err)
	require.Equal(t, res, strconv.Itoa(512*1024))

	_, err = RemoteRun("root", v.PlanetaryIP, "mv /disk1/test.txt /disk2/", privateKey)
	require.NoError(t, err)

	res, err = RemoteRun("root", v.PlanetaryIP, "du /disk2/test.txt | head -n1 | awk '{print $1;}' | tr -d -c 0-9", privateKey)
	require.NoError(t, err)
	require.Equal(t, res, strconv.Itoa(512*1024))

	// create file -> d2, check file size, copy file -> d1, check file size

	_, err = RemoteRun("root", v.PlanetaryIP, "dd if=/dev/vdb bs=1M count=512 of=/disk2/test.txt", privateKey)
	require.NoError(t, err)

	res, err = RemoteRun("root", v.PlanetaryIP, "du /disk2/test.txt | head -n1 | awk '{print $1;}' | tr -d -c 0-9", privateKey)
	require.NoError(t, err)
	require.Equal(t, res, strconv.Itoa(512*1024))

	_, err = RemoteRun("root", v.PlanetaryIP, "cp /disk2/test.txt /disk1/", privateKey)
	require.NoError(t, err)

	res, err = RemoteRun("root", v.PlanetaryIP, "du /disk1/test.txt | head -n1 | awk '{print $1;}' | tr -d -c 0-9", privateKey)
	require.NoError(t, err)
	require.Equal(t, res, strconv.Itoa(512*1024))

	// copy same file -> d1 (not enough space)

	_, err = RemoteRun("root", v.PlanetaryIP, "cp /disk2/test.txt /disk1/test2.txt", privateKey)
	require.Error(t, err)
	require.Contains(t, err.Error(), "No space left on device")
}
