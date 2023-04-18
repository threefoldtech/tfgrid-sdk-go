// Package integration for integration tests
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func TestZDBDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	nodes, err := deployer.FilterNodes(ctx, tfPluginClient, nodeFilter)
	if err != nil {
		t.Skip("no available nodes found")
	}

	nodeID := uint32(nodes[0].NodeID)

	zdb := workloads.ZDB{
		Name:        "testName",
		Password:    "password",
		Public:      true,
		Size:        10,
		Description: "test des",
		Mode:        zos.ZDBModeUser,
	}

	dl := workloads.NewDeployment("zdb", nodeID, "", nil, "", nil, []workloads.ZDB{zdb}, nil, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(ctx, &dl)
	assert.NoError(t, err)

	z, err := tfPluginClient.State.LoadZdbFromGrid(nodeID, zdb.Name, dl.Name)
	assert.NoError(t, err)
	assert.NotEmpty(t, z.IPs)
	assert.NotEmpty(t, z.Namespace)
	assert.NotEmpty(t, z.Port)

	z.IPs = nil
	z.Port = 0
	z.Namespace = ""
	assert.Equal(t, zdb, z)

	// cancel all
	err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl)
	assert.NoError(t, err)

	_, err = tfPluginClient.State.LoadZdbFromGrid(nodeID, zdb.Name, dl.Name)
	assert.Error(t, err)
}
