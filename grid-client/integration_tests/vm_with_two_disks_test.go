// Package integration for integration tests
package integration

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func TestVMWithTwoDisk(t *testing.T) {
	tfPluginClient, err := setup()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	publicKey, privateKey, err := GenerateSSHKeyPair()
	require.NoError(t, err)

	nodes, err := deployer.FilterNodes(
		ctx,
		tfPluginClient,
		nodeFilter,
		[]uint64{*convertGBToBytes(2), *convertGBToBytes(1)},
		nil,
		[]uint64{minRootfs},
	)
	if err != nil {
		t.Skip("no available nodes found")
	}

	nodeID := uint32(nodes[0].NodeID)

	network := workloads.ZNet{
		Name:        generateRandString(10),
		Description: "network for testing",
		Nodes:       []uint32{nodeID},
		IPRange: gridtypes.NewIPNet(net.IPNet{
			IP:   net.IPv4(10, 20, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}),
		AddWGAccess: false,
	}

	disk1 := workloads.Disk{
		Name:   "diskTest1",
		SizeGB: 1,
	}
	disk2 := workloads.Disk{
		Name:   "diskTest2",
		SizeGB: 2,
	}

	vm := workloads.VM{
		Name:       "vm",
		Flist:      "https://hub.grid.tf/tf-official-apps/base:latest.flist",
		CPU:        2,
		Planetary:  true,
		Memory:     1024,
		Entrypoint: "/sbin/zinit init",
		EnvVars: map[string]string{
			"SSH_KEY": publicKey,
		},
		Mounts: []workloads.Mount{
			{DiskName: disk1.Name, MountPoint: "/disk1"},
			{DiskName: disk2.Name, MountPoint: "/disk2"},
		},
		IP:          "10.20.2.5",
		NetworkName: network.Name,
	}

	err = tfPluginClient.NetworkDeployer.Deploy(ctx, &network)
	require.NoError(t, err)

	defer func() {
		err = tfPluginClient.NetworkDeployer.Cancel(ctx, &network)
		assert.NoError(t, err)
	}()

	dl := workloads.NewDeployment(generateRandString(10), nodeID, "", nil, network.Name, []workloads.Disk{disk1, disk2}, nil, []workloads.VM{vm}, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(ctx, &dl)
	require.NoError(t, err)

	defer func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl)
		assert.NoError(t, err)
	}()

	v, err := tfPluginClient.State.LoadVMFromGrid(ctx, nodeID, vm.Name, dl.Name)
	require.NoError(t, err)

	resDisk1, err := tfPluginClient.State.LoadDiskFromGrid(ctx, nodeID, disk1.Name, dl.Name)
	require.NoError(t, err)
	require.Equal(t, disk1, resDisk1)

	resDisk2, err := tfPluginClient.State.LoadDiskFromGrid(ctx, nodeID, disk2.Name, dl.Name)
	require.NoError(t, err)
	require.Equal(t, disk2, resDisk2)

	planetaryIP := v.PlanetaryIP
	require.NotEmpty(t, planetaryIP)

	// Check that disk has been mounted successfully

	output, err := RemoteRun("root", planetaryIP, "df -h | grep -w /disk1", privateKey)
	require.NoError(t, err)
	require.Contains(t, output, fmt.Sprintf("%d.0G", disk1.SizeGB))

	output, err = RemoteRun("root", planetaryIP, "df -h | grep -w /disk2", privateKey)
	require.NoError(t, err)
	require.Contains(t, output, fmt.Sprintf("%d.0G", disk2.SizeGB))

	// create file -> d1, check file size, move file -> d2, check file size

	_, err = RemoteRun("root", planetaryIP, "dd if=/dev/vda bs=1M count=512 of=/disk1/test.txt", privateKey)
	require.NoError(t, err)

	res, err := RemoteRun("root", planetaryIP, "du /disk1/test.txt | head -n1 | awk '{print $1;}' | tr -d -c 0-9", privateKey)
	require.NoError(t, err)
	require.Equal(t, res, strconv.Itoa(512*1024))

	_, err = RemoteRun("root", planetaryIP, "mv /disk1/test.txt /disk2/", privateKey)
	require.NoError(t, err)

	res, err = RemoteRun("root", planetaryIP, "du /disk2/test.txt | head -n1 | awk '{print $1;}' | tr -d -c 0-9", privateKey)
	require.NoError(t, err)
	require.Equal(t, res, strconv.Itoa(512*1024))

	// create file -> d2, check file size, copy file -> d1, check file size

	_, err = RemoteRun("root", planetaryIP, "dd if=/dev/vdb bs=1M count=512 of=/disk2/test.txt", privateKey)
	require.NoError(t, err)

	res, err = RemoteRun("root", planetaryIP, "du /disk2/test.txt | head -n1 | awk '{print $1;}' | tr -d -c 0-9", privateKey)
	require.NoError(t, err)
	require.Equal(t, res, strconv.Itoa(512*1024))

	_, err = RemoteRun("root", planetaryIP, "cp /disk2/test.txt /disk1/", privateKey)
	require.NoError(t, err)

	res, err = RemoteRun("root", planetaryIP, "du /disk1/test.txt | head -n1 | awk '{print $1;}' | tr -d -c 0-9", privateKey)
	require.NoError(t, err)
	require.Equal(t, res, strconv.Itoa(512*1024))

	// copy same file -> d1 (not enough space)

	_, err = RemoteRun("root", planetaryIP, "cp /disk2/test.txt /disk1/test2.txt", privateKey)
	require.NoError(t, err)
}
