package internal

import (
	"context"
	"fmt"
	"testing"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/mocks"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/pkg"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

func reset(t *testing.T, farmerbot FarmerBot, oldNode1, oldNode2 node, oldFarm substrate.Farm) {
	t.Helper()

	assert.NoError(t, farmerbot.updateNode(oldNode1))
	assert.NoError(t, farmerbot.updateNode(oldNode2))
	farmerbot.farm = oldFarm
}

func TestFindNode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	sub := mocks.NewMockSubstrate(ctrl)
	rmb := mocks.NewMockRMB(ctrl)

	ctx := context.Background()

	inputs := Config{
		FarmID:        1,
		IncludedNodes: []uint32{1, 2},
		Power: power{WakeUpThresholdPercentages: ThresholdPercentages{
			CRU: 30,
			SRU: 30,
			MRU: 30,
			HRU: 30,
		}},
	}

	farmerbot, err := NewFarmerBot(ctx, inputs, "dev", aliceSeed, peer.KeyTypeSr25519)
	assert.Error(t, err)

	// mock state
	resources := pkg.Capacity{HRU: convertGBToBytes(1), SRU: convertGBToBytes(1), CRU: 1, MRU: convertGBToBytes(1)}
	mockRMBAndSubstrateCalls(ctx, sub, rmb, inputs, true, false, resources, []string{}, false, false)

	state, err := newState(ctx, sub, rmb, inputs, farmTwinID)
	assert.NoError(t, err)
	farmerbot.state = state

	node := farmerbot.nodes[0]
	node.dedicated = false
	farmerbot.nodes[0] = node
	node2 := farmerbot.nodes[1]
	node2.dedicated = false
	farmerbot.nodes[1] = node2

	oldNode1 := farmerbot.nodes[0]
	oldNode2 := farmerbot.nodes[1]
	oldFarm := farmerbot.farm

	nodeOptions := NodeFilterOption{
		PublicIPs: 1,
		SRU:       1,
		MRU:       1,
		CRU:       1,
		HRU:       1,
	}

	t.Run("test valid find node: found an ON node", func(t *testing.T) {
		nodeID, err := farmerbot.findNode(sub, nodeOptions)
		assert.NoError(t, err)

		_, node, err := farmerbot.getNode(nodeID)
		assert.NoError(t, err)
		assert.Contains(t, farmerbot.nodes, node)

		reset(t, farmerbot, oldNode1, oldNode2, oldFarm)
	})

	t.Run("test valid find node: found an ON node, trying to power off fails because resources is claimed", func(t *testing.T) {
		nodeID, err := farmerbot.findNode(sub, nodeOptions)
		assert.NoError(t, err)

		_, node, err := farmerbot.getNode(nodeID)
		assert.NoError(t, err)
		assert.Contains(t, farmerbot.nodes, node)

		err = farmerbot.powerOff(sub, nodeID)
		assert.Error(t, err)

		reset(t, farmerbot, oldNode1, oldNode2, oldFarm)
	})

	t.Run("test valid find node: found an ON node (first is OFF)", func(t *testing.T) {
		node := farmerbot.nodes[0]
		node.powerState = off
		farmerbot.nodes[0] = node

		nodeID, err := farmerbot.findNode(sub, nodeOptions)
		assert.NoError(t, err)
		assert.Equal(t, nodeID, uint32(farmerbot.nodes[1].ID))

		reset(t, farmerbot, oldNode1, oldNode2, oldFarm)
	})

	t.Run("test valid find node: node is rented (second node is found)", func(t *testing.T) {
		node := farmerbot.nodes[0]
		node.hasActiveRentContract = true
		farmerbot.nodes[0] = node

		nodeID, err := farmerbot.findNode(sub, nodeOptions)
		assert.NoError(t, err)
		assert.Contains(t, farmerbot.config.IncludedNodes, nodeID)

		reset(t, farmerbot, oldNode1, oldNode2, oldFarm)
	})

	t.Run("test valid find node: node is dedicated so node is found", func(t *testing.T) {
		node := farmerbot.nodes[0]
		node.dedicated = true
		farmerbot.nodes[0] = node

		nodeID, err := farmerbot.findNode(sub, nodeOptions)
		assert.NoError(t, err)
		assert.Contains(t, farmerbot.config.IncludedNodes, nodeID)

		reset(t, farmerbot, oldNode1, oldNode2, oldFarm)
	})

	t.Run("test valid find node: options and nodes are dedicated and nodes are unused", func(t *testing.T) {
		nodeID, err := farmerbot.findNode(sub, nodeOptions)
		assert.NoError(t, err)
		assert.Contains(t, farmerbot.config.IncludedNodes, nodeID)

		reset(t, farmerbot, oldNode1, oldNode2, oldFarm)
	})

	t.Run("test valid find node: no gpus with specified device/vendor in first node (second is found)", func(t *testing.T) {
		node2 := farmerbot.nodes[1]
		node2.gpus = []pkg.GPU{
			{
				Device: "device",
				Vendor: "vendor",
			},
		}
		farmerbot.nodes[1] = node2

		nodeID, err := farmerbot.findNode(sub, NodeFilterOption{GPUVendors: []string{"vendor"}, GPUDevices: []string{"device"}})
		assert.NoError(t, err)
		assert.Equal(t, nodeID, uint32(farmerbot.nodes[1].ID))

		reset(t, farmerbot, oldNode1, oldNode2, oldFarm)
	})

	t.Run("test invalid find node: no gpus in nodes", func(t *testing.T) {
		_, err := farmerbot.findNode(sub, NodeFilterOption{GPUVendors: []string{"vendor"}, GPUDevices: []string{"device"}})
		assert.Error(t, err)
	})

	t.Run("test invalid find node: found an OFF node but change power failed", func(t *testing.T) {
		node := farmerbot.nodes[0]
		node.powerState = off
		node2 := farmerbot.nodes[1]
		node2.powerState = off
		farmerbot.nodes[0] = node
		farmerbot.nodes[1] = node2

		sub.EXPECT().SetNodePowerTarget(farmerbot.identity, gomock.Any(), true).Return(types.Hash{}, fmt.Errorf("error"))

		_, err := farmerbot.findNode(sub, nodeOptions)
		assert.Error(t, err)

		reset(t, farmerbot, oldNode1, oldNode2, oldFarm)
	})

	t.Run("test invalid find node: no enough public ips", func(t *testing.T) {
		farmerbot.farm.PublicIPs = []substrate.PublicIP{}

		_, err := farmerbot.findNode(sub, nodeOptions)
		assert.Error(t, err)

		farmerbot.farm = oldFarm
	})

	t.Run("test invalid find node: certified so no nodes found", func(t *testing.T) {
		_, err := farmerbot.findNode(sub, NodeFilterOption{Certified: true})
		assert.Error(t, err)
	})

	t.Run("test invalid find node: publicConfig so no nodes found", func(t *testing.T) {
		_, err := farmerbot.findNode(sub, NodeFilterOption{PublicConfig: true})
		assert.Error(t, err)
	})

	t.Run("test invalid find node: dedicated so no nodes found", func(t *testing.T) {
		_, err := farmerbot.findNode(sub, NodeFilterOption{Dedicated: true})
		assert.Error(t, err)
	})

	t.Run("test valid find node: nodes are dedicated and used, no nodes found", func(t *testing.T) {
		node := farmerbot.nodes[0]
		node.dedicated = true
		farmerbot.nodes[0] = node
		node2 := farmerbot.nodes[1]
		node2.dedicated = true
		farmerbot.nodes[1] = node2

		_, err := farmerbot.findNode(sub, NodeFilterOption{})
		assert.Error(t, err)

		reset(t, farmerbot, oldNode1, oldNode2, oldFarm)
	})

	t.Run("test invalid find node: node is excluded", func(t *testing.T) {
		_, err := farmerbot.findNode(sub, NodeFilterOption{NodesExcluded: []uint32{uint32(farmerbot.nodes[0].ID), uint32(farmerbot.nodes[1].ID)}})
		assert.Error(t, err)
	})

	t.Run("test invalid find node: node cannot claim resources", func(t *testing.T) {
		node := farmerbot.nodes[0]
		node.resources.total = capacity{}
		node2 := farmerbot.nodes[1]
		node2.resources.total = capacity{}
		farmerbot.nodes[0] = node
		farmerbot.nodes[1] = node2

		_, err := farmerbot.findNode(sub, nodeOptions)
		assert.Error(t, err)
	})
}
