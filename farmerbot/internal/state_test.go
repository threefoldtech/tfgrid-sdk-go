package internal

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/mocks"
	zos "github.com/threefoldtech/zos/client"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func mockRMBAndSubstrateCalls(
	ctx context.Context, sub *mocks.MockSubstrate, rmb *mocks.MockRMB,
	inputs Config, on bool, noFarm bool,
	resources gridtypes.Capacity, errs []string, emptyNode, emptyTwin bool,
) {
	farmErr, nodesErr, nodeErr, dedicatedErr, rentErr, powerErr, statsErr, poolsErr, gpusErr := mocksErr(errs)

	// farm calls
	if !noFarm {
		sub.EXPECT().GetFarm(inputs.FarmID).Return(&substrate.Farm{ID: 1, DedicatedFarm: true, PublicIPs: []substrate.PublicIP{{IP: "1.1.1.1"}}}, farmErr)
		if farmErr != nil {
			return
		}
	}

	sub.EXPECT().GetNodes(inputs.FarmID).Return(append(inputs.IncludedNodes, inputs.ExcludedNodes...), nodesErr)
	if nodesErr != nil {
		return
	}

	// node calls
	for _, nodeID := range inputs.IncludedNodes {
		nodeIDVal := types.U32(nodeID)
		if emptyNode {
			nodeIDVal = 0
		}

		twinIDVal := types.U32(nodeID)
		if emptyTwin {
			twinIDVal = 0
		}

		sub.EXPECT().GetNode(nodeID).Return(&substrate.Node{ID: nodeIDVal, TwinID: twinIDVal}, nodeErr)
		if nodeErr != nil {
			return
		}

		sub.EXPECT().GetDedicatedNodePrice(nodeID).Return(uint64(0), dedicatedErr)
		if dedicatedErr != nil {
			return
		}

		sub.EXPECT().GetNodeRentContract(nodeID).Return(uint64(0), rentErr)
		if rentErr != nil && rentErr != substrate.ErrNotFound {
			return
		}

		sub.EXPECT().GetPowerTarget(nodeID).Return(substrate.NodePower{
			State: substrate.PowerState{
				IsUp:   on,
				IsDown: !on,
			},
			Target: substrate.Power{
				IsUp:   on,
				IsDown: !on,
			},
		}, powerErr)
		if powerErr != nil {
			return
		}

		rmb.EXPECT().Statistics(ctx, uint32(twinIDVal)).Return(zos.Counters{Total: resources}, statsErr)
		if statsErr != nil {
			return
		}

		rmb.EXPECT().GetStoragePools(ctx, uint32(twinIDVal)).Return([]pkg.PoolMetrics{}, poolsErr)
		if poolsErr != nil {
			return
		}

		rmb.EXPECT().ListGPUs(ctx, uint32(twinIDVal)).Return([]zos.GPU{}, gpusErr)
		if gpusErr != nil {
			return
		}
	}
}

func TestSetConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	sub := mocks.NewMockSubstrate(ctrl)
	rmb := mocks.NewMockRMB(ctrl)

	ctx := context.Background()

	inputs := Config{
		FarmID:        1,
		IncludedNodes: []uint32{1, 2},
		Power:         power{WakeUpThreshold: 30},
	}

	resources := gridtypes.Capacity{HRU: 1, SRU: 1, CRU: 1, MRU: 1}

	t.Run("test valid state: no periodic wake up start, wakeup threshold (< min => min)", func(t *testing.T) {
		mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, true, false, resources, []string{}, false, false)

		state, err := newState(ctx, sub, rmb, inputs)
		assert.NoError(t, err)
		assert.Equal(t, uint32(state.farm.ID), uint32(1))
		assert.True(t, state.nodes[1].dedicated)
		assert.True(t, state.nodes[2].dedicated)
		assert.Equal(t, uint32(state.nodes[1].ID), uint32(1))
		assert.Equal(t, uint32(state.nodes[2].ID), uint32(2))

		now := time.Now()
		assert.Equal(t, state.config.Power.PeriodicWakeUpStart.PeriodicWakeUpTime().Hour(), now.Hour())
		assert.Equal(t, state.config.Power.PeriodicWakeUpStart.PeriodicWakeUpTime().Minute(), now.Minute())
		assert.Equal(t, state.config.Power.PeriodicWakeUpLimit, defaultPeriodicWakeUPLimit)
		assert.Equal(t, state.config.Power.OverProvisionCPU, defaultCPUProvision)
		assert.Equal(t, state.config.Power.WakeUpThreshold, minWakeUpThreshold)
	})

	t.Run("test valid state: wake up threshold (> max => max)", func(t *testing.T) {
		mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, true, false, resources, []string{}, false, false)

		inputs.Power.WakeUpThreshold = 100

		state, err := newState(ctx, sub, rmb, inputs)
		assert.NoError(t, err)
		assert.Equal(t, state.config.Power.WakeUpThreshold, maxWakeUpThreshold)
	})

	t.Run("test valid state: wake up threshold (is 0 => default)", func(t *testing.T) {
		mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, true, false, resources, []string{}, false, false)

		inputs.Power.WakeUpThreshold = 0

		state, err := newState(ctx, sub, rmb, inputs)
		assert.NoError(t, err)
		assert.Equal(t, state.config.Power.WakeUpThreshold, defaultWakeUpThreshold)
	})

	t.Run("test valid state: invalid wake up threshold", func(t *testing.T) {
		mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, true, false, resources, []string{}, false, false)
		inputs.Power.WakeUpThreshold = 110

		_, err := newState(ctx, sub, rmb, inputs)
		assert.Error(t, err)

		inputs.Power.WakeUpThreshold = defaultWakeUpThreshold
	})

	t.Run("test valid state: nodes are off in substrate", func(t *testing.T) {
		mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, false, false, resources, []string{}, false, false)

		_, err := newState(ctx, sub, rmb, inputs)
		assert.NoError(t, err)
	})

	t.Run("test invalid state: cpu provision out of range", func(t *testing.T) {
		inputs.Power.OverProvisionCPU = 6

		_, err := newState(ctx, sub, rmb, inputs)
		assert.Error(t, err)

		inputs.Power.OverProvisionCPU = 0
	})

	t.Run("test invalid state: failed substrate and rmb calls", func(t *testing.T) {
		calls := []string{"farm", "nodes", "node", "dedicated", "rent", "power", "stats", "pools", "gpus"}

		for _, call := range calls {
			mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, true, false, resources, []string{call}, false, false)

			_, err := newState(ctx, sub, rmb, inputs)
			assert.Error(t, err)
		}
	})

	t.Run("test invalid state: rent contract doesn't exist", func(t *testing.T) {
		mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, true, false, resources, []string{"rentNotExist"}, false, false)

		state, err := newState(ctx, sub, rmb, inputs)
		assert.NoError(t, err)
		assert.False(t, state.nodes[1].hasActiveRentContract)
	})

	t.Run("test valid excluded node", func(t *testing.T) {
		inputs.ExcludedNodes = append(inputs.ExcludedNodes, 3)

		mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, true, false, resources, []string{}, false, false)

		_, err := newState(ctx, sub, rmb, inputs)
		assert.NoError(t, err)

		inputs.ExcludedNodes = []uint32{}
	})

	t.Run("test invalid state < 2 nodes are provided", func(t *testing.T) {
		inputs.ExcludedNodes = append(inputs.ExcludedNodes, 2)
		inputs.IncludedNodes = []uint32{1}

		mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, true, false, resources, []string{}, false, false)

		_, err := newState(ctx, sub, rmb, inputs)
		assert.Error(t, err)

		inputs.ExcludedNodes = []uint32{}
		inputs.IncludedNodes = []uint32{1, 2}
	})

	t.Run("test invalid state no farm ID", func(t *testing.T) {
		sub.EXPECT().GetFarm(inputs.FarmID).Return(&substrate.Farm{ID: 0}, nil)
		sub.EXPECT().GetNodes(inputs.FarmID).Return([]uint32{}, nil)

		_, err := newState(ctx, sub, rmb, inputs)
		assert.Error(t, err)
	})

	t.Run("test invalid state no node ID", func(t *testing.T) {
		mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, true, false, resources, []string{}, true, false)

		_, err := newState(ctx, sub, rmb, inputs)
		assert.Error(t, err)
	})

	t.Run("test invalid state no twin ID", func(t *testing.T) {
		mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, true, false, resources, []string{}, false, true)

		_, err := newState(ctx, sub, rmb, inputs)
		assert.Error(t, err)
	})

	t.Run("test invalid state no node sru", func(t *testing.T) {
		resources := gridtypes.Capacity{HRU: 1, SRU: 0, CRU: 1, MRU: 1}
		mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, true, false, resources, []string{}, false, false)

		_, err := newState(ctx, sub, rmb, inputs)
		assert.Error(t, err)
	})

	t.Run("test invalid state no cru", func(t *testing.T) {
		resources := gridtypes.Capacity{HRU: 1, SRU: 1, CRU: 0, MRU: 1}
		mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, true, false, resources, []string{}, false, false)

		_, err := newState(ctx, sub, rmb, inputs)
		assert.Error(t, err)
	})

	t.Run("test invalid state no mru", func(t *testing.T) {
		resources := gridtypes.Capacity{HRU: 1, SRU: 1, CRU: 1, MRU: 0}
		mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, true, false, resources, []string{}, false, false)

		_, err := newState(ctx, sub, rmb, inputs)
		assert.Error(t, err)
	})

	t.Run("test invalid state no hru", func(t *testing.T) {
		resources := gridtypes.Capacity{HRU: 0, SRU: 1, CRU: 1, MRU: 1}
		mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, true, false, resources, []string{}, false, false)

		_, err := newState(ctx, sub, rmb, inputs)
		assert.Error(t, err)
	})
}

func TestStateModel(t *testing.T) {
	state := state{
		nodes: map[uint32]node{1: {
			Node: substrate.Node{ID: 1},
		}, 2: {
			Node: substrate.Node{ID: 2},
		}},
	}

	t.Run("test update node", func(t *testing.T) {
		err := state.updateNode(node{Node: substrate.Node{ID: 1, TwinID: 1}})
		assert.NoError(t, err)
		assert.Equal(t, uint32(state.nodes[1].TwinID), uint32(1))
	})

	t.Run("test update node (not found)", func(t *testing.T) {
		err := state.updateNode(node{Node: substrate.Node{ID: 10}})
		assert.Error(t, err)
	})

	t.Run("test filter nodes (power state)", func(t *testing.T) {
		nodes := state.filterNodesPower([]powerState{on})
		assert.Equal(t, len(nodes), len(state.nodes))
	})

	t.Run("test filter nodes (power state)", func(t *testing.T) {
		nodes := state.filterNodesPower([]powerState{shuttingDown})
		assert.Empty(t, nodes)
	})

	t.Run("test filter allowed nodes to shut down", func(t *testing.T) {
		nodes := state.filterAllowedNodesToShutDown()
		assert.Equal(t, len(nodes), len(state.nodes))
	})

	t.Run("test update node", func(t *testing.T) {
		state.addNode(node{Node: substrate.Node{ID: 1, TwinID: 1}})
		assert.Equal(t, uint32(state.nodes[1].TwinID), uint32(1))

		state.deleteNode(1)
		assert.Empty(t, state.nodes[1])
	})
}

func mocksErr(errs []string) (farmErr, nodesErr, nodeErr, dedicatedErr, rentErr, powerErr, statsErr, poolsErr, gpusErr error) {
	// errors
	if slices.Contains(errs, "farm") {
		farmErr = fmt.Errorf("error")
	}

	if slices.Contains(errs, "nodes") {
		nodesErr = fmt.Errorf("error")
	}

	if slices.Contains(errs, "node") {
		nodeErr = fmt.Errorf("error")
	}

	if slices.Contains(errs, "dedicated") {
		dedicatedErr = fmt.Errorf("error")
	}

	if slices.Contains(errs, "rent") {
		rentErr = fmt.Errorf("error")
	}

	if slices.Contains(errs, "rentNotExist") {
		rentErr = substrate.ErrNotFound
	}

	if slices.Contains(errs, "power") {
		powerErr = fmt.Errorf("error")
	}

	if slices.Contains(errs, "stats") {
		statsErr = fmt.Errorf("error")
	}

	if slices.Contains(errs, "pools") {
		poolsErr = fmt.Errorf("error")
	}

	if slices.Contains(errs, "gpus") {
		gpusErr = fmt.Errorf("error")
	}

	return
}
