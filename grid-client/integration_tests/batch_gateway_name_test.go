// Package integration for integration tests
package integration

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"

	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func TestBatchGatewayNameDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
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
		2,
	)
	if err != nil {
		t.Skipf("no available nodes found: %v", err)
	}

	nodeID1 := uint32(nodes[0].NodeID)
	nodeID2 := uint32(nodes[1].NodeID)

	network := generateBasicNetwork([]uint32{nodeID1, nodeID2})

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

	dl := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID1, "", nil, network.Name, nil, nil, []workloads.VM{vm}, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(ctx, &dl)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl)
		require.NoError(t, err)
	})

	v, err := tfPluginClient.State.LoadVMFromGrid(ctx, nodeID1, vm.Name, dl.Name)
	require.NoError(t, err)

	backend := fmt.Sprintf("http://[%s]:9000", v.PlanetaryIP)
	gw1 := workloads.GatewayNameProxy{
		NodeID:         nodeID1,
		Name:           generateRandString(10),
		TLSPassthrough: false,
		Backends:       []zos.Backend{zos.Backend(backend)},
	}

	gw2 := workloads.GatewayNameProxy{
		NodeID:         nodeID2,
		Name:           generateRandString(10),
		TLSPassthrough: false,
		Backends:       []zos.Backend{zos.Backend(backend)},
	}

	err = tfPluginClient.GatewayNameDeployer.BatchDeploy(ctx, []*workloads.GatewayNameProxy{&gw1, &gw2})
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.GatewayNameDeployer.Cancel(ctx, &gw1)
		require.NoError(t, err)

		err = tfPluginClient.GatewayNameDeployer.Cancel(ctx, &gw2)
		require.NoError(t, err)
	})

	g1, err := tfPluginClient.State.LoadGatewayNameFromGrid(ctx, nodeID1, gw1.Name, gw1.Name)
	require.NoError(t, err)
	require.NotEmpty(t, g1.FQDN)

	g2, err := tfPluginClient.State.LoadGatewayNameFromGrid(ctx, nodeID2, gw2.Name, gw2.Name)
	require.NoError(t, err)
	require.NotEmpty(t, g2.FQDN)

	_, err = RemoteRun("root", v.PlanetaryIP, "apk add python3; python3 -m http.server 9000 --bind :: &> /dev/null &", privateKey)
	require.NoError(t, err)

	time.Sleep(3 * time.Second)
	require.NoError(t, testFQDN(g1.FQDN))
	require.NoError(t, testFQDN(g2.FQDN))
}

func testFQDN(fqdn string) error {
	cl := &http.Client{
		Timeout: 10 * time.Second,
	}

	response, err := cl.Get(fmt.Sprintf("https://%s", fqdn))
	if err != nil {
		return err
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if body != nil {
		defer response.Body.Close()
	}

	if !strings.Contains(string(body), "Directory listing for") {
		return fmt.Errorf("response body doesn't contain 'Directory listing for'")
	}

	return nil
}
