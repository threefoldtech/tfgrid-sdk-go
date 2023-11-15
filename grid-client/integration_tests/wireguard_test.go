// Package integration for integration tests
package integration

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"golang.org/x/sync/errgroup"
)

func TestWG(t *testing.T) {
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
	if err != nil {
		t.Skip("no available nodes found")
	}

	nodeID := uint32(nodes[0].NodeID)

	network := workloads.ZNet{
		Name:        "WireguardNetwork",
		Description: "network for testing",
		Nodes:       []uint32{nodeID},
		IPRange: gridtypes.NewIPNet(net.IPNet{
			IP:   net.IPv4(10, 20, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}),
		AddWGAccess: true,
	}

	vm := workloads.VM{
		Name:       "vm",
		Flist:      "https://hub.grid.tf/tf-official-apps/base:latest.flist",
		CPU:        2,
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

	// wireguard
	n, err := tfPluginClient.State.LoadNetworkFromGrid(dl.NetworkName)
	if !assert.NoError(t, err) {
		return
	}

	wgConfig := n.AccessWGConfig
	if !assert.NotEmpty(t, wgConfig) {
		return
	}

	tempDir := t.TempDir()
	conf, err := UpWg(wgConfig, tempDir)
	if !assert.NoError(t, err) {
		return
	}

	defer func() {
		_, err := DownWG(conf)
		assert.NoError(t, err)
	}()

	if !assert.True(t, TestConnection(v.IP, "22")) {
		return
	}

	err = AreWgIPsReachable(wgConfig, []string{v.IP}, privateKey)
	if !assert.NoError(t, err) {
		return
	}
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
		return "", errors.Wrapf(err, "could not execute wg-quick up with "+f.Name())
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
