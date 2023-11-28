// Package manager provides how to manage nodes, nodes and power
package manager

import (
	"fmt"
	"testing"

	types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/mocks"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/models"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/parser"
)

var nodeCapacity = models.Capacity{
	CRU: 1,
	SRU: 1,
	MRU: 1,
	HRU: 1,
}

var configContent = `
{ 
	"included_nodes": [ 1, 2 ],
	"farm_id": 1, 
	"power": { "periodic_wake_up_start": "08:30AM", "wake_up_threshold": 30 }
}`

func TestNodeManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	sub := mocks.NewMockSub(ctrl)

	identity, err := substrate.NewIdentityFromSr25519Phrase("bad mnemonic")
	assert.Error(t, err)

	// configs
	farmID := uint32(1)
	sub.EXPECT().GetFarm(farmID).Return(&substrate.Farm{ID: 1, PublicIPs: []substrate.PublicIP{{IP: "1.1.1.1"}}}, nil)
	nodes := []uint32{1, 2}
	sub.EXPECT().GetNodes(farmID).Return(nodes, nil)
	sub.EXPECT().GetNode(nodes[0]).Return(&substrate.Node{ID: 1, TwinID: 1, Resources: substrate.Resources{
		HRU: 1, SRU: 1, CRU: 1, MRU: 1}}, nil)
	sub.EXPECT().GetDedicatedNodePrice(nodes[0]).Return(uint64(0), nil)
	sub.EXPECT().GetNode(nodes[1]).Return(&substrate.Node{ID: 2, TwinID: 2, Resources: substrate.Resources{
		HRU: 1, SRU: 1, CRU: 1, MRU: 1}}, nil)
	sub.EXPECT().GetDedicatedNodePrice(nodes[1]).Return(uint64(0), nil)

	inputs, err := parser.ParseIntoInputConfig([]byte(configContent), "json")
	assert.NoError(t, err)

	config, err := models.SetConfig(sub, inputs)
	assert.NoError(t, err)

	nodeManager := NewNodeManager(identity, sub, &config)

	nodeOptions := models.NodeOptions{
		PublicIPs: 1,
		Capacity:  nodeCapacity,
	}

	t.Run("test valid find node: found an ON node", func(t *testing.T) {
		nodeID, err := nodeManager.FindNode(nodeOptions)
		assert.NoError(t, err)
		assert.Equal(t, nodeID, uint32(config.Nodes[0].ID))
	})

	t.Run("test valid find node: found an ON node (first is OFF)", func(t *testing.T) {
		config.Nodes[0].PowerState = models.OFF

		nodeID, err := nodeManager.FindNode(nodeOptions)
		assert.NoError(t, err)
		assert.Equal(t, nodeID, uint32(config.Nodes[1].ID))

		config.Nodes[0].PowerState = models.ON
	})

	t.Run("test valid find node: found an OFF node", func(t *testing.T) {
		config.Nodes[0].PowerState = models.OFF
		config.Nodes[1].PowerState = models.OFF

		sub.EXPECT().SetNodePowerState(nodeManager.identity, true)

		nodeID, err := nodeManager.FindNode(nodeOptions)
		assert.NoError(t, err)
		assert.Equal(t, nodeID, uint32(config.Nodes[0].ID))

		config.Nodes[0].PowerState = models.ON
		config.Nodes[1].PowerState = models.ON
	})

	t.Run("test valid find node: node is rented (second node is found)", func(t *testing.T) {
		config.Nodes[0].HasActiveRentContract = true

		nodeID, err := nodeManager.FindNode(nodeOptions)
		assert.NoError(t, err)
		assert.Equal(t, nodeID, uint32(config.Nodes[1].ID))

		config.Nodes[0].HasActiveRentContract = false
	})

	t.Run("test valid find node: node is dedicated so second node is found", func(t *testing.T) {
		config.Nodes[0].Dedicated = true
		config.Nodes[0].Resources.Total = models.Capacity{}

		nodeID, err := nodeManager.FindNode(nodeOptions)
		assert.NoError(t, err)
		assert.Equal(t, nodeID, uint32(config.Nodes[1].ID))

		config.Nodes[0].Dedicated = false
		config.Nodes[0].Resources.Total = nodeCapacity
	})

	t.Run("test valid find node: options and nodes are dedicated and nodes are unused", func(t *testing.T) {
		config.Nodes[0].Dedicated = true
		config.Nodes[1].Dedicated = true

		nodeID, err := nodeManager.FindNode(nodeOptions)
		assert.NoError(t, err)
		assert.Equal(t, nodeID, uint32(config.Nodes[0].ID))

		config.Nodes[0].Dedicated = false
		config.Nodes[1].Dedicated = false
	})

	t.Run("test valid find node: no gpus with specified device/vendor in first node (second is found)", func(t *testing.T) {
		config.Nodes[1].GPUs = []models.GPU{
			{
				Device: "device",
				Vendor: "vendor",
			},
		}

		nodeID, err := nodeManager.FindNode(models.NodeOptions{GPUVendors: []string{"vendor"}, GPUDevices: []string{"device"}})
		assert.NoError(t, err)
		assert.Equal(t, nodeID, uint32(config.Nodes[1].ID))

		config.Nodes[1].GPUs = []models.GPU{}
	})

	t.Run("test invalid find node: no gpus in nodes", func(t *testing.T) {
		_, err = nodeManager.FindNode(models.NodeOptions{GPUVendors: []string{"vendor"}, GPUDevices: []string{"device"}})
		assert.Error(t, err)
	})

	t.Run("test invalid find node: found an OFF node but change power failed", func(t *testing.T) {
		config.Nodes[0].PowerState = models.OFF
		config.Nodes[1].PowerState = models.OFF

		sub.EXPECT().SetNodePowerState(nodeManager.identity, true).Return(types.Hash{}, fmt.Errorf("error"))

		_, err = nodeManager.FindNode(models.NodeOptions{})
		assert.Error(t, err)

		config.Nodes[0].PowerState = models.ON
		config.Nodes[1].PowerState = models.ON
	})

	t.Run("test invalid find node: no enough public ips", func(t *testing.T) {
		config.Farm.PublicIPs = []substrate.PublicIP{}

		_, err = nodeManager.FindNode(nodeOptions)
		assert.Error(t, err)

		config.Farm.PublicIPs = []substrate.PublicIP{{
			IP: "1.1.1.1",
		}}
	})

	t.Run("test invalid find node: certified so no nodes found", func(t *testing.T) {
		_, err = nodeManager.FindNode(models.NodeOptions{Certified: true})
		assert.Error(t, err)
	})

	t.Run("test invalid find node: publicConfig so no nodes found", func(t *testing.T) {
		_, err = nodeManager.FindNode(models.NodeOptions{PublicConfig: true})
		assert.Error(t, err)
	})

	t.Run("test invalid find node: dedicated so no nodes found", func(t *testing.T) {
		_, err = nodeManager.FindNode(models.NodeOptions{Dedicated: true})
		assert.Error(t, err)
	})

	t.Run("test invalid find node: node is excluded", func(t *testing.T) {
		nodeOptions := models.NodeOptions{NodeExclude: []uint32{uint32(config.Nodes[0].ID), uint32(config.Nodes[1].ID)}}
		_, err = nodeManager.FindNode(nodeOptions)
		assert.Error(t, err)
	})

	t.Run("test invalid find node: node cannot claim resources", func(t *testing.T) {
		config.Nodes[0].Resources.Total = models.Capacity{}
		config.Nodes[1].Resources.Total = models.Capacity{}

		_, err = nodeManager.FindNode(models.NodeOptions{Capacity: nodeCapacity})
		assert.Error(t, err)
		config.Nodes[0].Resources.Total = nodeCapacity
		config.Nodes[1].Resources.Total = nodeCapacity
	})

}
