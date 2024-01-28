package internal

import (
	"context"
	"testing"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/mocks"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

const (
	aliceSeed  = "0xe5be9a5092b81bca64be81d212e7f2f9eba183bb7a90954f7b76361f6edb5c0a"
	farmTwinID = 1
)

func TestFarmerbot(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	sub := mocks.NewMockSubstrate(ctrl)
	rmb := mocks.NewMockRMB(ctrl)

	ctx := context.Background()

	inputs := Config{
		FarmID:        1,
		IncludedNodes: []uint32{1, 2},
		Power:         power{WakeUpThreshold: 50},
	}

	farmerbot, err := NewFarmerBot(ctx, inputs, "dev", aliceSeed, peer.KeyTypeSr25519)
	assert.Error(t, err)
	farmerbot.rmbNodeClient = rmb

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

	t.Run("invalid identity", func(t *testing.T) {
		_, err := NewFarmerBot(ctx, Config{}, "dev", "invalid", peer.KeyTypeSr25519)
		assert.Error(t, err)
	})

	t.Run("test serve", func(t *testing.T) {
		farmerbot.substrateManager = substrate.NewManager(SubstrateURLs[DevNetwork]...)
		identity, err := substrate.NewIdentityFromSr25519Phrase(aliceSeed)
		assert.NoError(t, err)
		farmerbot.identity = identity

		err = farmerbot.serve(ctx)
		assert.True(t, errors.Is(err, substrate.ErrNotFound))
	})

	t.Run("test iterateOnNodes: update nodes and power off extra node (periodic wake up: already on)", func(t *testing.T) {
		mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, true, true, resources, []string{}, false, false)

		sub.EXPECT().SetNodePowerTarget(farmerbot.identity, gomock.Any(), false).Return(types.Hash{}, nil)

		err = farmerbot.iterateOnNodes(ctx, sub)
		assert.NoError(t, err)
	})

	t.Run("test iterateOnNodes: update nodes (periodic wake up: off node)", func(t *testing.T) {
		oldNode1.powerState = off
		oldNode2.powerState = off
		state.addNode(oldNode1)
		state.addNode(oldNode2)
		farmerbot.state = state

		mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, false, true, resources, []string{}, false, false)

		sub.EXPECT().SetNodePowerTarget(farmerbot.identity, gomock.Any(), true).Return(types.Hash{}, nil)

		err = farmerbot.iterateOnNodes(ctx, sub)
		assert.NoError(t, err)
	})

	t.Run("test iterateOnNodes: update nodes (periodic wake up: failed to set off node)", func(t *testing.T) {
		oldNode1.powerState = off
		oldNode2.powerState = off
		state.addNode(oldNode1)
		state.addNode(oldNode2)
		farmerbot.state = state

		mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, false, true, resources, []string{}, false, false)

		sub.EXPECT().SetNodePowerTarget(farmerbot.identity, gomock.Any(), true).Return(types.Hash{}, errors.New("err"))
		sub.EXPECT().SetNodePowerTarget(farmerbot.identity, gomock.Any(), true).Return(types.Hash{}, nil)

		err = farmerbot.iterateOnNodes(ctx, sub)
		assert.NoError(t, err)
	})
}
