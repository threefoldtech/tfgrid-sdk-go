// Package integration for integration tests
package integration

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/client"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func ConvertGPUsToStr(gpus []client.GPU) (zosGPUs []zos.GPU) {
	for _, g := range gpus {
		zosGPUs = append(zosGPUs, zos.GPU(g.ID))
	}

	return
}

func TestVMWithGPUDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	publicKey, _, err := GenerateSSHKeyPair()
	assert.NoError(t, err)

	// TODO: add a filtered node
	nodeID := uint32(93)
	nodeClient, err := tfPluginClient.NcPool.GetNodeClient(tfPluginClient.SubstrateConn, nodeID)
	assert.NoError(t, err)

	gpus, err := nodeClient.GPUs(ctx)
	assert.NoError(t, err)

	network := workloads.ZNet{
		Name:        "gpuNetwork",
		Description: "network for testing gpu",
		Nodes:       []uint32{nodeID},
		IPRange: gridtypes.NewIPNet(net.IPNet{
			IP:   net.IPv4(10, 20, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}),
		AddWGAccess: false,
	}

	vm := workloads.VM{
		Name:       "gpu",
		Flist:      "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-22.04.flist",
		CPU:        2,
		Planetary:  true,
		Memory:     1024,
		GPUs:       ConvertGPUsToStr(gpus),
		Entrypoint: "/sbin/zinit init",
		EnvVars: map[string]string{
			"SSH_KEY": publicKey,
		},
		IP:          "10.20.2.5",
		NetworkName: network.Name,
	}

	err = tfPluginClient.NetworkDeployer.Deploy(ctx, &network)
	assert.NoError(t, err)

	defer func() {
		err = tfPluginClient.NetworkDeployer.Cancel(ctx, &network)
		assert.NoError(t, err)
	}()

	dl := workloads.NewDeployment("gpu", nodeID, "", nil, network.Name, nil, nil, []workloads.VM{vm}, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(ctx, &dl)
	assert.NoError(t, err)

	defer func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl)
		assert.NoError(t, err)
	}()

	_, err = tfPluginClient.State.LoadVMFromGrid(nodeID, vm.Name, dl.Name)
	assert.NoError(t, err)
}
