// Package integration for integration tests
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

func TestDiskDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	nodes, err := deployer.FilterNodes(ctx, tfPluginClient, nodeFilter)
	assert.NoError(t, err)

	nodeID := uint32(nodes[0].NodeID)

	disk := workloads.Disk{
		Name:        "testName",
		SizeGB:      1,
		Description: "disk test",
	}

	dl := workloads.NewDeployment("disk", nodeID, "", nil, "", []workloads.Disk{disk}, nil, nil, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(ctx, &dl)
	assert.NoError(t, err)

	resDisk, err := tfPluginClient.State.LoadDiskFromGrid(nodeID, disk.Name, dl.Name)
	assert.NoError(t, err)
	assert.Equal(t, disk, resDisk)

	// cancel all
	err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl)
	assert.NoError(t, err)

	_, err = tfPluginClient.State.LoadDiskFromGrid(nodeID, disk.Name, dl.Name)
	assert.Error(t, err)
}
