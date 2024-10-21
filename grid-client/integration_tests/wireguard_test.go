// Package integration for integration tests
package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"golang.org/x/sync/errgroup"
)

func TestWG(t *testing.T) {
	tfPluginClient, err := setup()
	if err != nil {
		t.Skipf("plugin creation failed: %v", err)
	}

	publicKey, privateKey, err := GenerateSSHKeyPair()
	require.NoError(t, err)

	nodes, err := deployer.FilterNodes(
		context.Background(),
		tfPluginClient,
		generateNodeFilter(),
		nil,
		nil,
		[]uint64{*convertGBToBytes(minRootfs)},
		1,
	)
	if err != nil {
		t.Skipf("no available nodes found: %v", err)
	}

	nodeID := uint32(nodes[0].NodeID)

	network, err := generateBasicNetwork([]uint32{nodeID})
	require.NoError(t, err)
	network.AddWGAccess = true

	vm, err := generateBasicVM("vm", nodeID, network.Name, publicKey)
	require.NoError(t, err)
	vm.RootfsSizeMB = minRootfs * 1024

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

	// wireguard
	n, err := tfPluginClient.State.LoadNetworkFromGrid(context.Background(), dl.NetworkName)
	require.NoError(t, err)

	wgConfig := n.AccessWGConfig
	require.NotEmpty(t, wgConfig)

	tempDir := t.TempDir()
	conf, err := UpWg(wgConfig, tempDir)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err := DownWG(conf)
		require.NoError(t, err)
	})

	require.True(t, CheckConnection(v.IP, "22"))
	require.NoError(t, AreWgIPsReachable(wgConfig, []string{v.IP}, privateKey))
}

// UpWg used for setting up the wireguard interface
func UpWg(wgConfig, wgConfDir string) (string, error) {
	f, err := os.Create(path.Join(wgConfDir, "test.conf"))
	if err != nil {
		return "", errors.Wrapf(err, "could not create file")
	}

	_, err = f.WriteString(wgConfig)
	if err != nil {
		return "", errors.Wrapf(err, "could not write on file")
	}

	_, err = exec.Command("wg-quick", "up", f.Name()).Output()
	if err != nil {
		return "", errors.Wrapf(err, "could not execute wg-quick up with %s", f.Name())
	}

	return f.Name(), nil
}

// DownWG used for tearing down the wireguard interface
func DownWG(confFile string) (string, error) {
	cmd := exec.Command("wg-quick", "down", confFile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "could not execute wg-quick down with output %s", out)
	}
	return string(out), nil
}

// AreWgIPsReachable used to check if wire guard ip is reachable
func AreWgIPsReachable(wgConfig string, ipsToCheck []string, privateKey string) error {
	g := new(errgroup.Group)
	for _, ip := range ipsToCheck {
		ip := ip
		g.Go(func() error {
			output, err := RemoteRun("root", ip, "ifconfig", privateKey)
			if err != nil {
				return errors.Wrapf(err, "could not connect as a root user to the machine with ip %s with output %s", ip, output)
			}
			if !strings.Contains(output, ip) {
				return errors.Wrapf(err, "ip %s could not be verified. ifconfig output: %s", ip, output)
			}
			return nil
		})
	}
	return g.Wait()
}
