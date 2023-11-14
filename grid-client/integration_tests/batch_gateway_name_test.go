// Package integration for integration tests
package integration

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"

	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func TestBatchGatewayNameDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	if !assert.NoError(t, err) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	publicKey, _, err := GenerateSSHKeyPair()
	if !assert.NoError(t, err) {
		return
	}

	nodeFilter := types.NodeFilter{
		Status:  &statusUp,
		FarmIDs: []uint64{1},
		Rented:  &falseVal,
		Domain:  &trueVal,
	}

	nodes, err := deployer.FilterNodes(ctx, tfPluginClient, nodeFilter, nil, nil, []uint64{minRootfs})
	if err != nil || len(nodes) < 2 {
		t.Skip("no available nodes found")
	}

	nodeID1 := uint32(nodes[0].NodeID)
	nodeID2 := uint32(nodes[1].NodeID)

	network := workloads.ZNet{
		Name:        "testNameGWNetwork",
		Description: "network for testing",
		Nodes:       []uint32{nodeID1, nodeID2},
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
		Planetary:  true,
		Memory:     1024,
		Entrypoint: "/sbin/zinit init",
		EnvVars: map[string]string{
			"SSH_KEY": publicKey,
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

	dl := workloads.NewDeployment("vm", nodeID1, "", nil, network.Name, nil, nil, []workloads.VM{vm}, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(ctx, &dl)
	if !assert.NoError(t, err) {
		return
	}

	defer func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl)
		assert.NoError(t, err)
	}()

	v, err := tfPluginClient.State.LoadVMFromGrid(nodeID1, vm.Name, dl.Name)
	if !assert.NoError(t, err) {
		return
	}

	backend := fmt.Sprintf("http://[%s]:9000", v.YggIP)
	gw1 := workloads.GatewayNameProxy{
		NodeID:         nodeID1,
		Name:           "test1",
		TLSPassthrough: false,
		Backends:       []zos.Backend{zos.Backend(backend)},
	}

	gw2 := workloads.GatewayNameProxy{
		NodeID:         nodeID2,
		Name:           "test2",
		TLSPassthrough: false,
		Backends:       []zos.Backend{zos.Backend(backend)},
	}

	err = tfPluginClient.GatewayNameDeployer.BatchDeploy(ctx, []*workloads.GatewayNameProxy{&gw1, &gw2})
	if !assert.NoError(t, err) {
		return
	}

	defer func() {
		err = tfPluginClient.GatewayNameDeployer.Cancel(ctx, &gw1)
		assert.NoError(t, err)

		err = tfPluginClient.GatewayNameDeployer.Cancel(ctx, &gw2)
		assert.NoError(t, err)
	}()

	result, err := tfPluginClient.State.LoadGatewayNameFromGrid(nodeID1, gw1.Name, gw1.Name)
	if !assert.NoError(t, err) || !assert.NotEmpty(t, result.FQDN) {
		return
	}

	result, err = tfPluginClient.State.LoadGatewayNameFromGrid(nodeID2, gw2.Name, gw2.Name)
	if !assert.NoError(t, err) || !assert.NotEmpty(t, result.FQDN) {
		return
	}
}
