// Package integration for integration tests
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

func TestDiskDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	nodes, err := deployer.FilterNodes(ctx, tfPluginClient, nodeFilter, []uint64{*convertGBToBytes(1)}, nil, nil)
	if err != nil {
		t.Skip("no available nodes found")
	}

	nodeID := uint32(nodes[0].NodeID)

	disk := workloads.Disk{
		Name:        generateRandString(10),
		SizeGB:      1,
		Description: "disk test",
	}

	dl := workloads.NewDeployment(generateRandString(10), nodeID, "", nil, "", []workloads.Disk{disk}, nil, nil, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(ctx, &dl)
	require.NoError(t, err)

	defer func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl)
		assert.NoError(t, err)
	}()

	resDisk, err := tfPluginClient.State.LoadDiskFromGrid(ctx, nodeID, disk.Name, dl.Name)
	require.NoError(t, err)
	require.Equal(t, disk, resDisk)
}
