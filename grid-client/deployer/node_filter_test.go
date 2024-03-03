package deployer

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func TestHasEnoughStorage(t *testing.T) {
	// free overall storage is 14 separated in two pools for SSD and 10 free HDD
	pools := []client.PoolMetrics{
		{
			Type: zos.SSDDevice,
			Size: 10,
			Used: 5,
		},
		{
			Type: zos.SSDDevice,
			Size: 10,
			Used: 1,
		},
		{
			Type: zos.SSDDevice,
			Size: 10,
			Used: 10,
		},
		{
			Type: zos.HDDDevice,
			Size: 10,
			Used: 0,
		},
	}
	t.Run("fails because order because disks order", func(t *testing.T) {
		poolsCopy := make([]client.PoolMetrics, len(pools))
		copy(poolsCopy, pools)

		// 11 < 14
		disks := []uint64{3, 2, 1, 5}
		check := hasEnoughStorage(poolsCopy, disks, zos.SSDDevice)
		assert.False(t, check)
	})
	t.Run("fails because order because disks size", func(t *testing.T) {
		poolsCopy := make([]client.PoolMetrics, len(pools))
		copy(poolsCopy, pools)

		// 20 > 14
		disks := []uint64{20}
		check := hasEnoughStorage(poolsCopy, disks, zos.SSDDevice)
		assert.False(t, check)
	})
	t.Run("should fit", func(t *testing.T) {
		poolsCopy := make([]client.PoolMetrics, len(pools))
		copy(poolsCopy, pools)

		// 12 < 14
		disks := []uint64{4, 3, 5}
		check := hasEnoughStorage(poolsCopy, disks, zos.SSDDevice)
		assert.True(t, check)
	})
	t.Run("hru", func(t *testing.T) {
		poolsCopy := make([]client.PoolMetrics, len(pools))
		copy(poolsCopy, pools)

		disks := []uint64{6, 4}
		check := hasEnoughStorage(poolsCopy, disks, zos.HDDDevice)
		assert.True(t, check)
	})
}

func ExampleFilterNodes() {
	const mnemonic = "<mnemonics goes here>"
	const network = "<dev, test, qa, main>"

	tfPluginClient, err := NewTFPluginClient(mnemonic, WithNetwork(network))
	if err != nil {
		fmt.Println(err)
		return
	}

	trueVal := true
	statusUp := "up"
	freeMRU := uint64(2048)
	freeSRU := uint64(2048)

	filter := types.NodeFilter{
		Status:  &statusUp,
		IPv4:    &trueVal,
		FreeMRU: &freeMRU,
		FreeSRU: &freeSRU,
		FarmIDs: []uint64{uint64(1)},
	}

	_, err = FilterNodes(context.Background(), tfPluginClient, filter, []uint64{freeSRU}, nil, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("nodes filtered successfully")
}
