// Package integration for integration tests
package integration

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"

	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func TestGatewayFQDNDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	require.NoError(t, err)

	if tfPluginClient.Network != "dev" {
		t.Skip("test is not supported in any network but dev")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	publicKey, privateKey, err := GenerateSSHKeyPair()
	require.NoError(t, err)

	nodes, err := deployer.FilterNodes(
		ctx,
		tfPluginClient,
		generateNodeFilter(WithDomain()),
		nil,
		nil,
		nil,
		1,
	)
	if err != nil {
		t.Skipf("no available nodes found: %v", err)
	}

	nodeID := uint32(nodes[0].NodeID)

	network := generateBasicNetwork([]uint32{nodeID})

	vm := workloads.VM{
		Name:        "vm",
		NetworkName: network.Name,
		CPU:         minCPU,
		Memory:      int(minMemory) * 1024,
		Planetary:   true,
		Flist:       "https://hub.grid.tf/tf-official-apps/base:latest.flist",
		Entrypoint:  "/sbin/zinit init",
		EnvVars: map[string]string{
			"SSH_KEY": publicKey,
		},
	}

	err = tfPluginClient.NetworkDeployer.Deploy(ctx, &network)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.NetworkDeployer.Cancel(ctx, &network)
		require.NoError(t, err)
	})

	dl := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID, "", nil, network.Name, nil, nil, []workloads.VM{vm}, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(ctx, &dl)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl)
		require.NoError(t, err)
	})

	v, err := tfPluginClient.State.LoadVMFromGrid(ctx, nodeID, vm.Name, dl.Name)
	require.NoError(t, err)

	backend := fmt.Sprintf("http://[%s]:9000", v.PlanetaryIP)
	fqdn := "hamada1.3x0.me" // points to node 15 devnet
	gatewayNode := nodeID
	gw := workloads.GatewayFQDNProxy{
		NodeID:         gatewayNode,
		Name:           generateRandString(10),
		TLSPassthrough: false,
		Backends:       []zos.Backend{zos.Backend(backend)},
		FQDN:           fqdn,
	}

	err = tfPluginClient.GatewayFQDNDeployer.Deploy(ctx, &gw)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.GatewayFQDNDeployer.Cancel(ctx, &gw)
		require.NoError(t, err)
	})

	_, err = tfPluginClient.State.LoadGatewayFQDNFromGrid(ctx, gatewayNode, gw.Name, gw.Name)
	require.NoError(t, err)

	_, err = RemoteRun("root", v.PlanetaryIP, "apk add python3; python3 -m http.server 9000 --bind :: &> /dev/null &", privateKey)
	require.NoError(t, err)

	time.Sleep(3 * time.Second)

	cl := &http.Client{
		Timeout: 10 * time.Second,
	}
	response, err := cl.Get(fmt.Sprintf("https://%s", gw.FQDN))
	require.NoError(t, err)

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	if body != nil {
		defer response.Body.Close()
	}
	require.Contains(t, string(body), "Directory listing for")
}
