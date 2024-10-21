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

func TestGatewayNameDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	if err != nil {
		t.Skipf("plugin creation failed: %v", err)
	}

	publicKey, privateKey, err := GenerateSSHKeyPair()
	require.NoError(t, err)

	nodes, err := deployer.FilterNodes(
		context.Background(),
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

	gwNodeID := uint32(nodes[0].NodeID)

	nodes, err = deployer.FilterNodes(
		context.Background(),
		tfPluginClient,
		generateNodeFilter(),
		nil,
		nil,
		nil,
		1,
	)
	if err != nil {
		t.Skipf("no available nodes found: %v", err)
	}

	nodeID := uint32(nodes[0].NodeID)

	network, err := generateBasicNetwork([]uint32{nodeID})
	require.NoError(t, err)
	vm, err := generateBasicVM("vm", nodeID, network.Name, publicKey)
	require.NoError(t, err)

	err = tfPluginClient.NetworkDeployer.Deploy(context.Background(), &network)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.NetworkDeployer.Cancel(context.Background(), &network)
		require.NoError(t, err)
	})

	dl := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID, "", nil, network.Name, nil, nil, []workloads.VM{vm}, nil, nil, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(context.Background(), &dl)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(context.Background(), &dl)
		require.NoError(t, err)
	})

	v, err := tfPluginClient.State.LoadVMFromGrid(context.Background(), nodeID, vm.Name, dl.Name)
	require.NoError(t, err)

	backend := fmt.Sprintf("http://[%s]:9000", v.MyceliumIP)
	gw := workloads.GatewayNameProxy{
		NodeID:         gwNodeID,
		Name:           generateRandString(10),
		TLSPassthrough: false,
		Backends:       []zos.Backend{zos.Backend(backend)},
	}

	err = tfPluginClient.GatewayNameDeployer.Deploy(context.Background(), &gw)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.GatewayNameDeployer.Cancel(context.Background(), &gw)
		require.NoError(t, err)
	})

	result, err := tfPluginClient.State.LoadGatewayNameFromGrid(context.Background(), gwNodeID, gw.Name, gw.Name)
	require.NoError(t, err)
	require.NotEmpty(t, result.FQDN)

	_, err = RemoteRun("root", v.MyceliumIP, "apk add python3; python3 -m http.server 9000 --bind :: &> /dev/null &", privateKey)
	require.NoError(t, err)

	time.Sleep(3 * time.Second)
	cl := &http.Client{
		Timeout: 10 * time.Second,
	}
	response, err := cl.Get(fmt.Sprintf("https://%s", result.FQDN))
	require.NoError(t, err)

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	if body != nil {
		defer response.Body.Close()
	}
	require.Contains(t, string(body), "Directory listing for")
}
