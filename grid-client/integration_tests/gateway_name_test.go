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

	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func TestGatewayNameDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	publicKey, privateKey, err := GenerateSSHKeyPair()
	assert.NoError(t, err)

	nodeFilter.Domain = &trueVal
	nodes, err := deployer.FilterNodes(ctx, tfPluginClient, nodeFilter, nil, nil, []uint64{minRootfs})
	if err != nil || len(nodes) < 2 {
		t.Skip("no available nodes found")
	}

	nodeID := uint32(nodes[0].NodeID)
	gwNodeID := uint32(nodes[1].NodeID)

	network := workloads.ZNet{
		Name:        "testingNameGatewayNetwork",
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
	assert.NoError(t, err)

	defer func() {
		err = tfPluginClient.NetworkDeployer.Cancel(ctx, &network)
		assert.NoError(t, err)
	}()

	dl := workloads.NewDeployment("vm", nodeID, "", nil, network.Name, nil, nil, []workloads.VM{vm}, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(ctx, &dl)
	assert.NoError(t, err)

	defer func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl)
		assert.NoError(t, err)
	}()

	v, err := tfPluginClient.State.LoadVMFromGrid(ctx, nodeID, vm.Name, dl.Name)
	assert.NoError(t, err)

	backend := fmt.Sprintf("http://[%s]:9000", v.PlanetaryIP)
	gw := workloads.GatewayNameProxy{
		NodeID:         gwNodeID,
		Name:           "test",
		TLSPassthrough: false,
		Backends:       []zos.Backend{zos.Backend(backend)},
	}

	err = tfPluginClient.GatewayNameDeployer.Deploy(ctx, &gw)
	assert.NoError(t, err)

	defer func() {
		err = tfPluginClient.GatewayNameDeployer.Cancel(ctx, &gw)
		assert.NoError(t, err)
	}()

	result, err := tfPluginClient.State.LoadGatewayNameFromGrid(ctx, gwNodeID, gw.Name, gw.Name)
	assert.NoError(t, err)

	assert.NotEmpty(t, result.FQDN)

	_, err = RemoteRun("root", v.PlanetaryIP, "apk add python3; python3 -m http.server 9000 --bind :: &> /dev/null &", privateKey)
	assert.NoError(t, err)

	time.Sleep(3 * time.Second)
	cl := &http.Client{
		Timeout: 10 * time.Second,
	}
	response, err := cl.Get(fmt.Sprintf("https://%s", result.FQDN))
	assert.NoError(t, err)

	body, err := io.ReadAll(response.Body)
	assert.NoError(t, err)
	if body != nil {
		defer response.Body.Close()
	}
	assert.Contains(t, string(body), "Directory listing for")
}
