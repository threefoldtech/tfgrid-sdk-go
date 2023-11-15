// Package integration for integration tests
package integration

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func TestPresearchDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	if !assert.NoError(t, err) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	publicKey, privateKey, err := GenerateSSHKeyPair()
	if !assert.NoError(t, err) {
		return
	}
	nodeFilter.IPv4 = &trueVal
	nodeFilter.FreeIPs = &value1

	nodes, err := deployer.FilterNodes(ctx, tfPluginClient, nodeFilter, []uint64{*convertGBToBytes(1)}, nil, []uint64{*convertGBToBytes(20)})
	if err != nil {
		t.Skip("no available nodes found")
	}

	if err != nil {
		return
	}

	nodeID := uint32(nodes[0].NodeID)

	network := workloads.ZNet{
		Name:        "presearchNetworkTest",
		Description: "network for testing",
		Nodes:       []uint32{nodeID},
		IPRange: gridtypes.NewIPNet(net.IPNet{
			IP:   net.IPv4(10, 20, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}),
		AddWGAccess: false,
	}
	disk := workloads.Disk{
		Name:   "diskTest",
		SizeGB: 1,
	}

	vm := workloads.VM{
		Name:       "presearchTest",
		Flist:      "https://hub.grid.tf/tf-official-apps/presearch-v2.2.flist",
		CPU:        2,
		PublicIP:   true,
		Planetary:  true,
		Memory:     1024,
		RootfsSize: 20 * 1024,
		Entrypoint: "/sbin/zinit init",
		EnvVars: map[string]string{
			"SSH_KEY":                     publicKey,
			"PRESEARCH_REGISTRATION_CODE": "e5083a8d0a6362c6cf7a3078bfac81e3",
		},
		Mounts: []workloads.Mount{
			{DiskName: disk.Name, MountPoint: "/var/lib/docker"},
		},
		NetworkName: network.Name,
	}

	err = tfPluginClient.NetworkDeployer.Deploy(ctx, &network)
	if !assert.NoError(t, err) {
		return
	}

	defer func() {
		err = tfPluginClient.NetworkDeployer.Cancel(ctx, &network)
		assert.NoError(t, err)
	}()

	dl := workloads.NewDeployment("presearch", nodeID, "", nil, network.Name, []workloads.Disk{disk}, nil, []workloads.VM{vm}, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(ctx, &dl)
	if !assert.NoError(t, err) {
		return
	}

	defer func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl)
		assert.NoError(t, err)
	}()

	v, err := tfPluginClient.State.LoadVMFromGrid(nodeID, vm.Name, dl.Name)
	if !assert.NoError(t, err) {
		return
	}

	publicIP := strings.Split(v.ComputedIP, "/")[0]
	if !assert.NotEmpty(t, publicIP) {
		return
	}
	if !TestConnection(publicIP, "22") {
		t.Errorf("public ip is not reachable")
	}

	yggIP := v.YggIP
	if !assert.NotEmpty(t, yggIP) {
		return
	}

	output, err := RemoteRun("root", yggIP, "cat /proc/1/environ", privateKey)
	if !assert.NoError(t, err) {
		return
	}
	if !assert.Contains(t, output, "PRESEARCH_REGISTRATION_CODE=e5083a8d0a6362c6cf7a3078bfac81e3") {
		return
	}

	ticker := time.NewTicker(2 * time.Second)
	for now := time.Now(); time.Since(now) < 1*time.Minute; {
		<-ticker.C
		output, err = RemoteRun("root", yggIP, "zinit list", privateKey)
		if err == nil && strings.Contains(output, "prenode: Success") {
			break
		}
	}

	assert.NoError(t, err)
	assert.Contains(t, output, "prenode: Success")
}
