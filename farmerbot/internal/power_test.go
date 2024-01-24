package internal

import (
	"context"
	"testing"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/mocks"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func TestPower(t *testing.T) {
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

	farmerbot, err := NewFarmerBot(ctx, inputs, "dev", aliceSeed, peer.KeyTypeSr25519)
	assert.Error(t, err)

	// mock state
	resources := gridtypes.Capacity{HRU: 1, SRU: 1, CRU: 1, MRU: 1}
	mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, true, false, resources, []string{}, false, false)

	state, err := newState(ctx, sub, rmb, inputs, farmTwinID)
	assert.NoError(t, err)
	farmerbot.state = state

	node := farmerbot.nodes[2]
	node.dedicated = false
	farmerbot.nodes[2] = node
	node2 := farmerbot.nodes[2]
	node2.dedicated = false
	farmerbot.nodes[2] = node2

	oldNode1 := farmerbot.nodes[1]
	oldNode2 := farmerbot.nodes[2]

	t.Run("test valid power on: already on", func(t *testing.T) {
		err = farmerbot.powerOn(sub, uint32(state.nodes[1].ID))
		assert.NoError(t, err)
	})

	t.Run("test valid power on: already waking up", func(t *testing.T) {
		testNode := state.nodes[1]
		testNode.powerState = wakingUp
		state.addNode(testNode)

		err = farmerbot.powerOn(sub, uint32(state.nodes[1].ID))
		assert.NoError(t, err)

		state.addNode(oldNode1)
	})

	t.Run("test valid power on", func(t *testing.T) {
		testNode := state.nodes[1]
		testNode.powerState = off
		state.addNode(testNode)

		sub.EXPECT().SetNodePowerTarget(farmerbot.identity, uint32(state.nodes[1].ID), true).Return(types.Hash{}, nil)

		err = farmerbot.powerOn(sub, uint32(state.nodes[1].ID))
		assert.NoError(t, err)

		state.addNode(oldNode1)
	})

	t.Run("test invalid power on: node not found", func(t *testing.T) {
		err = farmerbot.powerOn(sub, 3)
		assert.Error(t, err)
	})

	t.Run("test invalid power on: set node failed", func(t *testing.T) {
		testNode := state.nodes[1]
		testNode.powerState = off
		state.addNode(testNode)

		sub.EXPECT().SetNodePowerTarget(farmerbot.identity, uint32(state.nodes[1].ID), true).Return(types.Hash{}, errors.New("error"))

		err = farmerbot.powerOn(sub, uint32(state.nodes[1].ID))
		assert.Error(t, err)

		state.addNode(oldNode1)
	})

	t.Run("test valid power off: already off", func(t *testing.T) {
		testNode := state.nodes[1]
		testNode.powerState = off
		state.addNode(testNode)

		err = farmerbot.powerOff(sub, uint32(state.nodes[1].ID))
		assert.NoError(t, err)

		state.addNode(oldNode1)
	})

	t.Run("test valid power off: already shutting down", func(t *testing.T) {
		testNode := state.nodes[1]
		testNode.powerState = shuttingDown
		state.addNode(testNode)

		err = farmerbot.powerOff(sub, uint32(state.nodes[1].ID))
		assert.NoError(t, err)

		state.addNode(oldNode1)
	})

	t.Run("test valid power off", func(t *testing.T) {
		sub.EXPECT().SetNodePowerTarget(farmerbot.identity, uint32(state.nodes[1].ID), false).Return(types.Hash{}, nil)

		err = farmerbot.powerOff(sub, uint32(state.nodes[1].ID))
		assert.NoError(t, err)

		state.addNode(oldNode1)
	})

	t.Run("test invalid power off: one node is on and cannot be off", func(t *testing.T) {
		testNode := state.nodes[1]
		testNode.powerState = off
		state.addNode(testNode)

		err = farmerbot.powerOff(sub, uint32(state.nodes[2].ID))
		assert.Error(t, err)

		state.addNode(oldNode1)
		state.addNode(oldNode2)
	})

	t.Run("test invalid power off: node is set to never shutdown", func(t *testing.T) {
		testNode := state.nodes[1]
		testNode.neverShutDown = true
		state.addNode(testNode)

		err = farmerbot.powerOff(sub, uint32(state.nodes[1].ID))
		assert.Error(t, err)

		state.addNode(oldNode1)
	})

	t.Run("test invalid power off: node has public config", func(t *testing.T) {
		testNode := state.nodes[1]
		testNode.PublicConfig.HasValue = true
		state.addNode(testNode)

		err = farmerbot.powerOff(sub, uint32(state.nodes[1].ID))
		assert.Error(t, err)

		state.addNode(oldNode1)
	})

	t.Run("test invalid power off: node has rent contract", func(t *testing.T) {
		testNode := state.nodes[1]
		testNode.hasActiveRentContract = true
		state.addNode(testNode)

		err = farmerbot.powerOff(sub, uint32(state.nodes[1].ID))
		assert.Error(t, err)

		state.addNode(oldNode1)
	})

	t.Run("test invalid power off: node power changed", func(t *testing.T) {
		testNode := state.nodes[1]
		testNode.lastTimePowerStateChanged = time.Now()
		state.addNode(testNode)

		err = farmerbot.powerOff(sub, uint32(state.nodes[1].ID))
		assert.Error(t, err)

		state.addNode(oldNode1)
	})

	t.Run("test invalid power off: node has used resources", func(t *testing.T) {
		testNode := state.nodes[1]
		testNode.resources.used = testNode.resources.total
		state.addNode(testNode)

		err = farmerbot.powerOff(sub, uint32(state.nodes[1].ID))
		assert.Error(t, err)

		state.addNode(oldNode1)
	})

	t.Run("test invalid power off: node not found", func(t *testing.T) {
		err = farmerbot.powerOff(sub, 3)
		assert.Error(t, err)
	})

	t.Run("test invalid power off: set node power failed", func(t *testing.T) {
		sub.EXPECT().SetNodePowerTarget(farmerbot.identity, uint32(state.nodes[1].ID), false).Return(types.Hash{}, errors.New("error"))
		sub.EXPECT().GetPowerTarget(gomock.Any()).Return(substrate.NodePower{}, nil)

		err = farmerbot.powerOff(sub, uint32(state.nodes[1].ID))
		assert.Error(t, err)
		assert.Equal(t, state.nodes[1].powerState, on)
		state.addNode(oldNode1)
	})

	t.Run("test invalid power off: set and get node power failed", func(t *testing.T) {
		sub.EXPECT().SetNodePowerTarget(farmerbot.identity, uint32(state.nodes[1].ID), false).Return(types.Hash{}, errors.New("error"))
		sub.EXPECT().GetPowerTarget(gomock.Any()).Return(substrate.NodePower{}, errors.New("error"))

		err = farmerbot.powerOff(sub, uint32(state.nodes[1].ID))
		assert.Error(t, err)
		assert.Equal(t, state.nodes[1].powerState, on)
		state.addNode(oldNode1)
	})

	t.Run("test invalid power off: set node power failed but target is changed", func(t *testing.T) {
		sub.EXPECT().SetNodePowerTarget(farmerbot.identity, uint32(state.nodes[1].ID), false).Return(types.Hash{}, errors.New("error"))
		sub.EXPECT().GetPowerTarget(uint32(state.nodes[1].ID)).Return(substrate.NodePower{
			Target: substrate.Power{IsDown: true},
		}, nil)

		err = farmerbot.powerOff(sub, uint32(state.nodes[1].ID))
		assert.Error(t, err)
		assert.Equal(t, state.nodes[1].powerState, shuttingDown)
		state.addNode(oldNode1)
	})

	t.Run("test power management: a node to shutdown (failed set the first node)", func(t *testing.T) {
		sub.EXPECT().SetNodePowerTarget(farmerbot.identity, gomock.Any(), false).Return(types.Hash{}, errors.New("error"))
		sub.EXPECT().GetPowerTarget(gomock.Any()).Return(substrate.NodePower{}, nil)
		sub.EXPECT().SetNodePowerTarget(farmerbot.identity, gomock.Any(), false).Return(types.Hash{}, nil)

		err = farmerbot.manageNodesPower(sub)
		assert.NoError(t, err)

		state.addNode(oldNode1)
		state.addNode(oldNode2)
	})

	t.Run("test power management: a node to shutdown (failed set the first node but power target changes)", func(t *testing.T) {
		sub.EXPECT().SetNodePowerTarget(farmerbot.identity, gomock.Any(), false).Return(types.Hash{}, errors.New("error"))
		sub.EXPECT().GetPowerTarget(gomock.Any()).Return(substrate.NodePower{
			Target: substrate.Power{IsDown: true},
		}, nil)

		err = farmerbot.manageNodesPower(sub)
		assert.NoError(t, err)

		state.addNode(oldNode1)
		state.addNode(oldNode2)
	})

	t.Run("test power management: nothing to shut down", func(t *testing.T) {
		testNode := state.nodes[1]
		testNode.powerState = off
		state.addNode(testNode)

		err = farmerbot.manageNodesPower(sub)
		assert.NoError(t, err)

		state.addNode(oldNode1)
	})

	t.Run("test power management: cannot shutdown public config", func(t *testing.T) {
		testNode := state.nodes[1]
		testNode.PublicConfig.HasValue = true
		state.addNode(testNode)
		testNode = state.nodes[2]
		testNode.PublicConfig.HasValue = true
		state.addNode(testNode)

		err = farmerbot.manageNodesPower(sub)
		assert.NoError(t, err)

		state.addNode(oldNode1)
		state.addNode(oldNode2)
	})

	t.Run("test power management: node is waking up", func(t *testing.T) {
		testNode := state.nodes[1]
		testNode.powerState = wakingUp
		state.addNode(testNode)

		err = farmerbot.manageNodesPower(sub)
		assert.NoError(t, err)

		state.addNode(oldNode1)
	})

	t.Run("test power management: a node to wake up (node 1 is used and node 2 is off)", func(t *testing.T) {
		testNode := state.nodes[1]
		testNode.resources.used = testNode.resources.total
		state.addNode(testNode)
		testNode = state.nodes[2]
		testNode.powerState = off
		state.addNode(testNode)

		sub.EXPECT().SetNodePowerTarget(farmerbot.identity, uint32(state.nodes[2].ID), true).Return(types.Hash{}, nil)

		err = farmerbot.manageNodesPower(sub)
		assert.NoError(t, err)

		state.addNode(oldNode1)
		state.addNode(oldNode2)
	})

	t.Run("test power management: a node to wake up (node 1 has rent contract)", func(t *testing.T) {
		testNode := state.nodes[1]
		testNode.hasActiveRentContract = true
		state.addNode(testNode)
		testNode = state.nodes[2]
		testNode.powerState = off
		state.addNode(testNode)

		sub.EXPECT().SetNodePowerTarget(farmerbot.identity, uint32(state.nodes[2].ID), true).Return(types.Hash{}, nil)

		err = farmerbot.manageNodesPower(sub)
		assert.NoError(t, err)

		state.addNode(oldNode1)
		state.addNode(oldNode2)
	})

	t.Run("test invalid power management: no nodes to wake up (usage is high)", func(t *testing.T) {
		testNode := state.nodes[1]
		testNode.resources.used = testNode.resources.total
		state.addNode(testNode)
		testNode = state.nodes[2]
		testNode.resources.used = testNode.resources.total
		state.addNode(testNode)

		err = farmerbot.manageNodesPower(sub)
		assert.Error(t, err)

		state.addNode(oldNode1)
		state.addNode(oldNode2)
	})

	t.Run("test valid power management: second node has no resources (usage is low)", func(t *testing.T) {
		testNode := state.nodes[1]
		testNode.resources.used = testNode.resources.total
		testNode.resources.used.cru = 0
		state.addNode(testNode)

		err = farmerbot.manageNodesPower(sub)
		assert.NoError(t, err)

		state.addNode(oldNode1)
		state.addNode(oldNode2)
	})

	t.Run("test power management: total resources is 0 (nothing happens)", func(t *testing.T) {
		testNode := state.nodes[1]
		testNode.resources.total = capacity{}
		state.addNode(testNode)
		testNode = state.nodes[2]
		testNode.resources.total = capacity{}
		state.addNode(testNode)

		err = farmerbot.manageNodesPower(sub)
		assert.NoError(t, err)

		state.addNode(oldNode1)
		state.addNode(oldNode2)
	})
}
