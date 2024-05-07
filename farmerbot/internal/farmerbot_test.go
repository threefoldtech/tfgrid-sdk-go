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
	proxyTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
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
	proxy := mocks.NewMockProxyClient(ctrl)
	sub := mocks.NewMockSubstrate(ctrl)
	rmb := mocks.NewMockRMB(ctrl)

	ctx := context.Background()

	inputs := Config{
		FarmID:        1,
		IncludedNodes: []uint32{1, 2},
		PriorityNodes: []uint32{2},
		Power: power{WakeUpThresholdPercentages: ThresholdPercentages{
			CRU: 50,
			SRU: 50,
			MRU: 50,
			HRU: 50,
		}},
	}

	farmerbot, err := NewFarmerBot(ctx, inputs, "dev", aliceSeed, peer.KeyTypeSr25519)
	assert.Error(t, err)
	farmerbot.rmbNodeClient = rmb
	farmerbot.gridProxyClient = proxy

	// mock state
	resources := gridtypes.Capacity{HRU: 1, SRU: 1, CRU: 1, MRU: 1}
	mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, true, false, resources, []string{}, false, false)

	state, err := newState(ctx, sub, rmb, inputs, farmTwinID)
	assert.NoError(t, err)
	farmerbot.state = state

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

	t.Run("test iterateOnNodes: update nodes and power off extra node (respect priority - periodic wake up: already on)", func(t *testing.T) {
		mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, true, true, resources, []string{}, false, false)

		sub.EXPECT().SetNodePowerTarget(farmerbot.identity, uint32(2), false).Return(types.Hash{}, nil)

		err = farmerbot.iterateOnNodes(ctx, sub)
		assert.NoError(t, err)
	})

	t.Run("test iterateOnNodes: update nodes (periodic wake up: off node)", func(t *testing.T) {
		proxy.EXPECT().Node(ctx, gomock.Any()).Return(proxyTypes.NodeWithNestedCapacity{}, nil).AnyTimes()

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
