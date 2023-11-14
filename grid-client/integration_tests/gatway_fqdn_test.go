// Package integration for integration tests
package integration

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"

	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func TestGatewayFQDNDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	if !assert.NoError(t, err) {
		return
	}

	if tfPluginClient.Network != "dev" {
		t.Skip("network is not dev")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	publicKey, privateKey, err := GenerateSSHKeyPair()
	if !assert.NoError(t, err) {
		return
	}

	nodeFilter := types.NodeFilter{
		Status:  &statusUp,
		FarmIDs: []uint64{1},
		Rented:  &falseVal,
	}

	nodes, err := deployer.FilterNodes(ctx, tfPluginClient, nodeFilter, nil, nil, []uint64{minRootfs})
	if err != nil {
		t.Skip("no available nodes found")
	}

	nodeID := uint32(nodes[0].NodeID)

	network := workloads.ZNet{
		Name:        "fqdnTestingNetwork",
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
	if !assert.NoError(t, err) {
		return
	}

	backend := fmt.Sprintf("http://[%s]:9000", v.YggIP)
	fqdn := "hamada1.3x0.me" // points to node 15 devnet
	gatewayNode := nodeID
	gw := workloads.GatewayFQDNProxy{
		NodeID:         gatewayNode,
		Name:           "test",
		TLSPassthrough: false,
		Backends:       []zos.Backend{zos.Backend(backend)},
		FQDN:           fqdn,
	}

	err = tfPluginClient.GatewayFQDNDeployer.Deploy(ctx, &gw)
	if !assert.NoError(t, err) {
		return
	}

	defer func() {
		err = tfPluginClient.GatewayFQDNDeployer.Cancel(ctx, &gw)
		assert.NoError(t, err)
	}()

	_, err = tfPluginClient.State.LoadGatewayFQDNFromGrid(gatewayNode, gw.Name, gw.Name)
	if !assert.NoError(t, err) {
		return
	}

	_, err = RemoteRun("root", v.YggIP, "apk add python3; python3 -m http.server 9000 --bind :: &> /dev/null &", privateKey)
	if !assert.NoError(t, err) {
		return
	}

	time.Sleep(3 * time.Second)
	response, err := http.Get(fmt.Sprintf("https://%s", gw.FQDN))
	if !assert.NoError(t, err) {
		return
	}

	body, err := io.ReadAll(response.Body)
	if !assert.NoError(t, err) {
		return
	}
	if body != nil {
		defer response.Body.Close()
	}
	assert.Contains(t, string(body), "Directory listing for")
}
