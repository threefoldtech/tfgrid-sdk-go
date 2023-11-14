// Package integration for integration tests
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func TestZDBDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	if !assert.NoError(t, err) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	nodeFilter := types.NodeFilter{
		Status:  &statusUp,
		FreeHRU: convertGBToBytes(2),
		FarmIDs: []uint64{1},
		Rented:  &falseVal,
	}
	nodes, err := deployer.FilterNodes(ctx, tfPluginClient, nodeFilter, nil, []uint64{*convertGBToBytes(10)}, nil)
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
	if !assert.NoError(t, err) {
		return
	}

	defer func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl)
		assert.NoError(t, err)
	}()

	z, err := tfPluginClient.State.LoadZdbFromGrid(nodeID, zdb.Name, dl.Name)
	assert.NoError(t, err)
	assert.NotEmpty(t, z.IPs)
	assert.NotEmpty(t, z.Namespace)
	assert.NotEmpty(t, z.Port)

	z.IPs = nil
	z.Port = 0
	z.Namespace = ""
	assert.Equal(t, zdb, z)
}
