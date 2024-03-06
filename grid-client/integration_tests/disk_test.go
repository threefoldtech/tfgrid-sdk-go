// Package integration for integration tests
package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
)

func TestDiskDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	nodes, err := deployer.FilterNodes(
		ctx,
		tfPluginClient,
		generateNodeFilter(WithFreeSRU(1)),
		[]uint64{*convertGBToBytes(1)},
		nil,
		nil,
		1,
	)
	if err != nil {
		t.Skipf("no available nodes found: %v", err)
	}

	nodeID := uint32(nodes[0].NodeID)

	disk := workloads.Disk{
		Name:        generateRandString(10),
		SizeGB:      1,
		Description: "disk test",
	}

	dl := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID, "", nil, "", []workloads.Disk{disk}, nil, nil, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(ctx, &dl)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl)
		require.NoError(t, err)
	})

	resDisk, err := tfPluginClient.State.LoadDiskFromGrid(ctx, nodeID, disk.Name, dl.Name)
	require.NoError(t, err)
	require.Equal(t, disk, resDisk)
}
