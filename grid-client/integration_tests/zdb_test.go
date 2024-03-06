// Package integration for integration tests
package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func TestZDBDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	zdbSize := 10

	nodes, err := deployer.FilterNodes(
		ctx,
		tfPluginClient,
		generateNodeFilter(WithFreeHRU(uint64(zdbSize))),
		nil,
		[]uint64{*convertGBToBytes(uint64(zdbSize))},
		nil,
		1,
	)
	if err != nil {
		t.Skipf("no available nodes found: %v", err)
	}

	nodeID := uint32(nodes[0].NodeID)

	zdb := workloads.ZDB{
		Name:        fmt.Sprintf("zdb_%s", generateRandString(10)),
		Password:    "password",
		Public:      true,
		Size:        zdbSize,
		Description: "test zdb",
		Mode:        zos.ZDBModeUser,
	}

	dl := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID, "", nil, "", nil, []workloads.ZDB{zdb}, nil, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(ctx, &dl)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl)
		require.NoError(t, err)
	})

	z, err := tfPluginClient.State.LoadZdbFromGrid(ctx, nodeID, zdb.Name, dl.Name)
	require.NoError(t, err)
	require.NotEmpty(t, z.IPs)
	require.NotEmpty(t, z.Namespace)
	require.NotEmpty(t, z.Port)

	zdbEndpoint := fmt.Sprintf("[%s]:%v", z.IPs[1], z.Port)

	redisDB := redis.NewClient(&redis.Options{
		Addr: zdbEndpoint,
	})
	_, err = redisDB.Do("SELECT", z.Namespace, z.Password).Result()
	require.NoError(t, err)

	_, err = redisDB.Set("key1", "val1", 0).Result()
	require.NoError(t, err)

	res, err := redisDB.Get("key1").Result()
	require.NoError(t, err)
	require.Equal(t, res, "val1")
}
