// Package integration for integration tests
package integration

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func TestBatchVMDeployment(t *testing.T) {
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

	nodes, err := deployer.FilterNodes(ctx, tfPluginClient, nodeFilter, nil, nil, []uint64{minRootfs})
	if err != nil || len(nodes) < 2 {
		t.Skip("no available nodes found")
	}

	nodeID1 := uint32(nodes[0].NodeID)
	nodeID2 := uint32(nodes[1].NodeID)

	network1 := workloads.ZNet{
		Name:        generateRandString(10),
		Description: "network for testing",
		Nodes:       []uint32{nodeID1},
		IPRange: gridtypes.NewIPNet(net.IPNet{
			IP:   net.IPv4(10, 20, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}),
		AddWGAccess: false,
	}

	network2 := network1
	network2.Nodes = []uint32{nodeID2}
	network2.Name += "2"

	vm1 := workloads.VM{
		Name:       "vm",
		Flist:      "https://hub.grid.tf/tf-official-apps/base:latest.flist",
		CPU:        2,
		Planetary:  true,
		Memory:     1024,
		Entrypoint: "/sbin/zinit init",
		EnvVars: map[string]string{
			"SSH_KEY": publicKey,
		},
		NetworkName: network1.Name,
	}

	vm2 := workloads.VM{
		Name:       "vm",
		Flist:      "https://hub.grid.tf/tf-official-apps/base:latest.flist",
		CPU:        2,
		Planetary:  true,
		Memory:     1024,
		Entrypoint: "/sbin/zinit init",
		EnvVars: map[string]string{
			"SSH_KEY": publicKey,
		},
		NetworkName: network2.Name,
	}

	err = tfPluginClient.NetworkDeployer.BatchDeploy(ctx, []*workloads.ZNet{&network1, &network2})
	if !assert.NoError(t, err) {
		return
	}

	defer func() {
		err = tfPluginClient.NetworkDeployer.Cancel(ctx, &network1)
		assert.NoError(t, err)

		err = tfPluginClient.NetworkDeployer.Cancel(ctx, &network2)
		assert.NoError(t, err)
	}()

	dl1 := workloads.NewDeployment(generateRandString(10), nodeID1, "", nil, network1.Name, nil, nil, []workloads.VM{vm1}, nil)
	dl2 := workloads.NewDeployment(generateRandString(10), nodeID2, "", nil, network2.Name, nil, nil, []workloads.VM{vm2}, nil)
	err = tfPluginClient.DeploymentDeployer.BatchDeploy(ctx, []*workloads.Deployment{&dl1, &dl2})
	if !assert.NoError(t, err) {
		return
	}

	defer func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl1)
		assert.NoError(t, err)

		err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl2)
		assert.NoError(t, err)
	}()

	v1, err := tfPluginClient.State.LoadVMFromGrid(ctx, nodeID1, vm1.Name, dl1.Name)
	if !assert.NoError(t, err) || !assert.NotEmpty(t, v1.PlanetaryIP) {
		return
	}

	output, err := RemoteRun("root", v1.PlanetaryIP, "ls /", privateKey)
	if !assert.NoError(t, err) || !assert.Contains(t, output, "root") {
		return
	}

	v2, err := tfPluginClient.State.LoadVMFromGrid(ctx, nodeID2, vm2.Name, dl2.Name)
	if !assert.NoError(t, err) || !assert.NotEmpty(t, v2.PlanetaryIP) {
		return
	}

	output, err = RemoteRun("root", v2.PlanetaryIP, "ls /", privateKey)
	if !assert.NoError(t, err) || !assert.Contains(t, output, "root") {
		return
	}
}
