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

func TestVMDeployment(t *testing.T) {
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

	nodeFilter := nodeFilter
	nodeFilter.IPv4 = &trueVal
	nodeFilter.FreeIPs = &value1

	nodes, err := deployer.FilterNodes(ctx, tfPluginClient, nodeFilter, nil, nil, []uint64{minRootfs})
	if err != nil {
		t.Skip("no available nodes found")
	}

	nodeID := uint32(nodes[0].NodeID)

	network := workloads.ZNet{
		Name:        "vmTestingNetwork",
		Description: "network for testing",
		Nodes:       []uint32{nodeID},
		IPRange: gridtypes.NewIPNet(net.IPNet{
			IP:   net.IPv4(10, 20, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}),
		AddWGAccess: false,
	}

	vm := workloads.VM{
		Name:       "vm",
		Flist:      "https://hub.grid.tf/tf-official-apps/base:latest.flist",
		CPU:        2,
		PublicIP:   true,
		Planetary:  true,
		Memory:     1024,
		Entrypoint: "/sbin/zinit init",
		EnvVars: map[string]string{
			"SSH_KEY": publicKey,
		},
		IP:          "10.20.2.5",
		NetworkName: network.Name,
	}

	t.Run("check single vm with public ip", func(t *testing.T) {
		err = tfPluginClient.NetworkDeployer.Deploy(ctx, &network)
		if !assert.NoError(t, err) {
			return
		}

		defer func() {
			err = tfPluginClient.NetworkDeployer.Cancel(ctx, &network)
			assert.NoError(t, err)
		}()

		dl := workloads.NewDeployment("vm", nodeID, "", nil, network.Name, nil, nil, []workloads.VM{vm}, nil)
		err = tfPluginClient.DeploymentDeployer.Deploy(ctx, &dl)
		if !assert.NoError(t, err) {
			return
		}

		defer func() {
			err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl)
			assert.NoError(t, err)
		}()

		v, err := tfPluginClient.State.LoadVMFromGrid(nodeID, vm.Name, dl.Name)
		if !assert.NoError(t, err) || !assert.Equal(t, v.IP, "10.20.2.5") {
			return
		}

		publicIP := strings.Split(v.ComputedIP, "/")[0]
		if !assert.NotEmpty(t, publicIP) || !assert.True(t, TestConnection(publicIP, "22")) {
			return
		}

		yggIP := v.YggIP
		if !assert.NotEmpty(t, yggIP) {
			return
		}

		output, err := RemoteRun("root", yggIP, "ls /", privateKey)
		if !assert.NoError(t, err) || !assert.Contains(t, output, "root") {
			return
		}
	})
}
