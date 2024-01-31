// Package integration for integration tests
package integration

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func TestTwoVMsSameNetwork(t *testing.T) {
	tfPluginClient, err := setup()
	assert.NoError(t, err)

	publicKey, privateKey, err := GenerateSSHKeyPair()
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	nodes, err := deployer.FilterNodes(ctx, tfPluginClient, nodeFilter, nil, nil, []uint64{minRootfs, minRootfs})
	if err != nil {
		t.Skip("no available nodes found")
	}

	nodeID := uint32(nodes[0].NodeID)

	network := workloads.ZNet{
		Name:        "vmsTestingNetwork",
		Description: "network for testing",
		Nodes:       []uint32{nodeID},
		IPRange: gridtypes.NewIPNet(net.IPNet{
			IP:   net.IPv4(10, 20, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}),
		AddWGAccess: false,
	}

	vm1 := workloads.VM{
		Name:       "vm1",
		Flist:      "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-22.04.flist",
		CPU:        2,
		PublicIP6:  true,
		Planetary:  true,
		Memory:     1024,
		Entrypoint: "/sbin/zinit init",
		EnvVars: map[string]string{
			"SSH_KEY": publicKey,
		},
		IP:          "10.20.2.5",
		NetworkName: network.Name,
	}

	vm2 := workloads.VM{
		Name:       "vm2",
		Flist:      "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-22.04.flist",
		CPU:        2,
		PublicIP6:  true,
		Planetary:  true,
		Memory:     1024,
		Entrypoint: "/sbin/zinit init",
		EnvVars: map[string]string{
			"SSH_KEY": publicKey,
		},
		IP:          "10.20.2.6",
		NetworkName: network.Name,
	}

	err = tfPluginClient.NetworkDeployer.Deploy(ctx, &network)
	assert.NoError(t, err)

	defer func() {
		err = tfPluginClient.NetworkDeployer.Cancel(ctx, &network)
		assert.NoError(t, err)
	}()

	t.Run("public ipv6 and yggdrasil", func(t *testing.T) {
		dl := workloads.NewDeployment("vm", nodeID, "", nil, network.Name, nil, nil, []workloads.VM{vm1, vm2}, nil)
		err = tfPluginClient.DeploymentDeployer.Deploy(ctx, &dl)
		assert.NoError(t, err)

		defer func() {
			err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl)
			assert.NoError(t, err)
		}()

		v1, err := tfPluginClient.State.LoadVMFromGrid(nodeID, vm1.Name, dl.Name)
		assert.NoError(t, err)

		v2, err := tfPluginClient.State.LoadVMFromGrid(nodeID, vm2.Name, dl.Name)
		assert.NoError(t, err)

		yggIP1 := v1.PlanetaryIP
		yggIP2 := v2.PlanetaryIP

		assert.NotEmpty(t, yggIP1)
		assert.NotEmpty(t, yggIP1)

		privateIP1 := v1.IP
		privateIP2 := v2.IP

		publicIP6_1 := strings.Split(v1.ComputedIP6, "/")[0]
		publicIP6_2 := strings.Split(v2.ComputedIP6, "/")[0]

		_, err = RemoteRun("root", yggIP1, "apt install -y netcat", privateKey)
		assert.NoError(t, err)

		_, err = RemoteRun("root", yggIP2, "apt install -y netcat", privateKey)
		assert.NoError(t, err)

		// check privateIP2 from vm1
		_, err = RemoteRun("root", yggIP1, fmt.Sprintf("nc -z %s 22", privateIP2), privateKey)
		assert.NoError(t, err)

		// check privateIP1 from vm2
		_, err = RemoteRun("root", yggIP2, fmt.Sprintf("nc -z %s 22", privateIP1), privateKey)
		assert.NoError(t, err)

		// check yggIP2 from vm1
		_, err = RemoteRun("root", yggIP1, fmt.Sprintf("nc -z %s 22", yggIP2), privateKey)
		assert.NoError(t, err)

		// check yggIP1 from vm2
		_, err = RemoteRun("root", yggIP2, fmt.Sprintf("nc -z %s 22", yggIP1), privateKey)
		assert.NoError(t, err)

		// check publicIP62 from vm1
		_, err = RemoteRun("root", yggIP1, fmt.Sprintf("nc -z %s 22", publicIP6_2), privateKey)
		assert.NoError(t, err)

		// check publicIP61 from vm2
		_, err = RemoteRun("root", yggIP2, fmt.Sprintf("nc -z %s 22", publicIP6_1), privateKey)
		assert.NoError(t, err)
	})
}
