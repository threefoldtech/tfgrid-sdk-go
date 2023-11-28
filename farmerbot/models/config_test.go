// Package models for farmerbot models.
package models

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/constants"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/mocks"
)

func TestSetConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	sub := mocks.NewMockSub(ctrl)

	inputs := InputConfig{
		FarmID:        1,
		IncludedNodes: []uint32{1, 2},
		Power:         Power{WakeUpThreshold: 30},
	}

	t.Run("test valid json: no periodic wake up start, wakeup threshold (< min => min)", func(t *testing.T) {
		sub.EXPECT().GetFarm(inputs.FarmID).Return(&substrate.Farm{ID: 1, DedicatedFarm: true}, nil)
		nodes := []uint32{1, 2}
		sub.EXPECT().GetNodes(inputs.FarmID).Return(nodes, nil)
		sub.EXPECT().GetNode(nodes[0]).Return(&substrate.Node{ID: 1, TwinID: 1, Resources: substrate.Resources{
			HRU: 1, SRU: 1, CRU: 1, MRU: 1}}, nil)
		sub.EXPECT().GetDedicatedNodePrice(nodes[0]).Return(uint64(0), nil)
		sub.EXPECT().GetNode(nodes[1]).Return(&substrate.Node{ID: 2, TwinID: 2, Resources: substrate.Resources{
			HRU: 1, SRU: 1, CRU: 1, MRU: 1}}, nil)
		sub.EXPECT().GetDedicatedNodePrice(nodes[1]).Return(uint64(0), nil)

		c, err := SetConfig(sub, inputs)
		assert.NoError(t, err)
		assert.Equal(t, uint32(c.Farm.ID), uint32(1))
		assert.Equal(t, c.Nodes[0].Resources.OverProvisionCPU, constants.DefaultCPUProvision)
		assert.True(t, c.Nodes[0].Dedicated)
		assert.True(t, c.Nodes[1].Dedicated)
		assert.Equal(t, uint32(c.Nodes[0].ID), uint32(1))
		assert.Equal(t, uint32(c.Nodes[1].ID), uint32(2))
		assert.Equal(t, c.Power.WakeUpThreshold, constants.MinWakeUpThreshold)

		now := time.Now()
		assert.Equal(t, c.Power.PeriodicWakeUpStart.PeriodicWakeUpTime().Hour(), now.Hour())
		assert.Equal(t, c.Power.PeriodicWakeUpStart.PeriodicWakeUpTime().Minute(), now.Minute())
		assert.Equal(t, c.Power.PeriodicWakeUpLimit, constants.DefaultPeriodicWakeUPLimit)
	})

	t.Run("test valid json: wake up threshold (> max => max)", func(t *testing.T) {
		inputs.Power.WakeUpThreshold = 100

		// configs mocks
		sub.EXPECT().GetFarm(inputs.FarmID).Return(&substrate.Farm{ID: 1, DedicatedFarm: true}, nil)
		nodes := []uint32{1, 2}
		sub.EXPECT().GetNodes(inputs.FarmID).Return(nodes, nil)
		sub.EXPECT().GetNode(nodes[0]).Return(&substrate.Node{ID: 1, TwinID: 1, Resources: substrate.Resources{
			HRU: 1, SRU: 1, CRU: 1, MRU: 1}}, nil)
		sub.EXPECT().GetDedicatedNodePrice(nodes[0]).Return(uint64(0), nil)
		sub.EXPECT().GetNode(nodes[1]).Return(&substrate.Node{ID: 2, TwinID: 2, Resources: substrate.Resources{
			HRU: 1, SRU: 1, CRU: 1, MRU: 1}}, nil)
		sub.EXPECT().GetDedicatedNodePrice(nodes[1]).Return(uint64(0), nil)

		c, err := SetConfig(sub, inputs)
		assert.NoError(t, err)
		assert.Equal(t, c.Power.WakeUpThreshold, constants.MaxWakeUpThreshold)
	})

	t.Run("test valid json: wake up threshold (is 0 => default)", func(t *testing.T) {
		inputs.Power.WakeUpThreshold = 0

		// configs mocks
		sub.EXPECT().GetFarm(inputs.FarmID).Return(&substrate.Farm{ID: 1, DedicatedFarm: true}, nil)
		nodes := []uint32{1, 2}
		sub.EXPECT().GetNodes(inputs.FarmID).Return(nodes, nil)
		sub.EXPECT().GetNode(nodes[0]).Return(&substrate.Node{ID: 1, TwinID: 1, Resources: substrate.Resources{
			HRU: 1, SRU: 1, CRU: 1, MRU: 1}}, nil)
		sub.EXPECT().GetDedicatedNodePrice(nodes[0]).Return(uint64(0), nil)
		sub.EXPECT().GetNode(nodes[1]).Return(&substrate.Node{ID: 2, TwinID: 2, Resources: substrate.Resources{
			HRU: 1, SRU: 1, CRU: 1, MRU: 1}}, nil)
		sub.EXPECT().GetDedicatedNodePrice(nodes[1]).Return(uint64(0), nil)

		c, err := SetConfig(sub, inputs)
		assert.NoError(t, err)
		assert.Equal(t, c.Power.WakeUpThreshold, constants.DefaultWakeUpThreshold)
	})

	t.Run("test invalid json: cpu provision out of range", func(t *testing.T) {
		inputs.Power.OverProvisionCPU = 6

		_, err := SetConfig(sub, inputs)
		assert.Error(t, err)

		inputs.Power.OverProvisionCPU = 0
	})

	t.Run("test invalid json: failed to get farm", func(t *testing.T) {
		// configs mocks
		sub.EXPECT().GetFarm(inputs.FarmID).Return(&substrate.Farm{ID: 1}, errors.New("error"))

		_, err := SetConfig(sub, inputs)
		assert.Error(t, err)
	})

	t.Run("test invalid json: failed to get nodes", func(t *testing.T) {
		// configs mocks
		sub.EXPECT().GetFarm(inputs.FarmID).Return(&substrate.Farm{ID: 1}, nil)
		nodes := []uint32{1, 2}
		sub.EXPECT().GetNodes(inputs.FarmID).Return(nodes, errors.New("error"))

		_, err := SetConfig(sub, inputs)
		assert.Error(t, err)
	})

	t.Run("test invalid json: failed to get node", func(t *testing.T) {
		// configs mocks
		sub.EXPECT().GetFarm(inputs.FarmID).Return(&substrate.Farm{ID: 1}, nil)
		nodes := []uint32{1, 2}
		sub.EXPECT().GetNodes(inputs.FarmID).Return(nodes, nil)
		sub.EXPECT().GetNode(nodes[0]).Return(&substrate.Node{}, errors.New("error"))

		_, err := SetConfig(sub, inputs)
		assert.Error(t, err)
	})

	t.Run("test invalid json: failed to get dedicated price", func(t *testing.T) {
		// configs mocks
		sub.EXPECT().GetFarm(inputs.FarmID).Return(&substrate.Farm{ID: 1}, nil)
		nodes := []uint32{1, 2}
		sub.EXPECT().GetNodes(inputs.FarmID).Return(nodes, nil)
		sub.EXPECT().GetNode(nodes[0]).Return(&substrate.Node{}, nil)
		sub.EXPECT().GetDedicatedNodePrice(nodes[0]).Return(uint64(0), errors.New("error"))

		_, err := SetConfig(sub, inputs)
		assert.Error(t, err)
	})

	t.Run("test invalid json < 2 nodes are provided", func(t *testing.T) {
		inputs.ExcludedNodes = append(inputs.ExcludedNodes, 2)

		// configs mocks
		sub.EXPECT().GetFarm(inputs.FarmID).Return(&substrate.Farm{ID: 1}, nil)
		nodes := []uint32{1, 2}
		sub.EXPECT().GetNodes(inputs.FarmID).Return(nodes, nil)
		sub.EXPECT().GetNode(nodes[0]).Return(&substrate.Node{}, nil)
		sub.EXPECT().GetDedicatedNodePrice(nodes[0]).Return(uint64(0), nil)

		_, err := SetConfig(sub, inputs)
		assert.Error(t, err)

		inputs.ExcludedNodes = []uint32{}
	})

	t.Run("test invalid json no node ID", func(t *testing.T) {
		// configs mocks
		sub.EXPECT().GetFarm(inputs.FarmID).Return(&substrate.Farm{ID: 1}, nil)
		nodes := []uint32{1, 2}
		sub.EXPECT().GetNodes(inputs.FarmID).Return(nodes, nil)
		sub.EXPECT().GetNode(nodes[0]).Return(&substrate.Node{ID: 0}, nil)
		sub.EXPECT().GetDedicatedNodePrice(nodes[0]).Return(uint64(0), nil)
		sub.EXPECT().GetNode(nodes[1]).Return(&substrate.Node{ID: 2}, nil)
		sub.EXPECT().GetDedicatedNodePrice(nodes[1]).Return(uint64(0), nil)

		_, err := SetConfig(sub, inputs)
		assert.Error(t, err)
	})

	t.Run("test invalid json no node twin ID", func(t *testing.T) {
		// configs mocks
		sub.EXPECT().GetFarm(inputs.FarmID).Return(&substrate.Farm{ID: 1}, nil)
		nodes := []uint32{1, 2}
		sub.EXPECT().GetNodes(inputs.FarmID).Return(nodes, nil)
		sub.EXPECT().GetNode(nodes[0]).Return(&substrate.Node{ID: 1}, nil)
		sub.EXPECT().GetDedicatedNodePrice(nodes[0]).Return(uint64(0), nil)
		sub.EXPECT().GetNode(nodes[1]).Return(&substrate.Node{ID: 2}, nil)
		sub.EXPECT().GetDedicatedNodePrice(nodes[1]).Return(uint64(0), nil)

		_, err := SetConfig(sub, inputs)
		assert.Error(t, err)
	})

	t.Run("test invalid json no node sru", func(t *testing.T) {
		// configs mocks
		sub.EXPECT().GetFarm(inputs.FarmID).Return(&substrate.Farm{ID: 1}, nil)
		nodes := []uint32{1, 2}
		sub.EXPECT().GetNodes(inputs.FarmID).Return(nodes, nil)
		sub.EXPECT().GetNode(nodes[0]).Return(&substrate.Node{ID: 1, TwinID: 1, Resources: substrate.Resources{
			HRU: 1, SRU: 0, CRU: 1, MRU: 1}}, nil)
		sub.EXPECT().GetDedicatedNodePrice(nodes[0]).Return(uint64(0), nil)
		sub.EXPECT().GetNode(nodes[1]).Return(&substrate.Node{ID: 2, TwinID: 2}, nil)
		sub.EXPECT().GetDedicatedNodePrice(nodes[1]).Return(uint64(0), nil)

		_, err := SetConfig(sub, inputs)
		assert.Error(t, err)
	})

	t.Run("test invalid json no cru", func(t *testing.T) {
		// configs mocks
		sub.EXPECT().GetFarm(inputs.FarmID).Return(&substrate.Farm{ID: 1}, nil)
		nodes := []uint32{1, 2}
		sub.EXPECT().GetNodes(inputs.FarmID).Return(nodes, nil)
		sub.EXPECT().GetNode(nodes[0]).Return(&substrate.Node{ID: 1, TwinID: 1, Resources: substrate.Resources{
			HRU: 1, SRU: 1, CRU: 0, MRU: 1}}, nil)
		sub.EXPECT().GetDedicatedNodePrice(nodes[0]).Return(uint64(0), nil)
		sub.EXPECT().GetNode(nodes[1]).Return(&substrate.Node{ID: 2, TwinID: 2}, nil)
		sub.EXPECT().GetDedicatedNodePrice(nodes[1]).Return(uint64(0), nil)

		_, err := SetConfig(sub, inputs)
		assert.Error(t, err)
	})

	t.Run("test invalid json no mru", func(t *testing.T) {
		// configs mocks
		sub.EXPECT().GetFarm(inputs.FarmID).Return(&substrate.Farm{ID: 1}, nil)
		nodes := []uint32{1, 2}
		sub.EXPECT().GetNodes(inputs.FarmID).Return(nodes, nil)
		sub.EXPECT().GetNode(nodes[0]).Return(&substrate.Node{ID: 1, TwinID: 1, Resources: substrate.Resources{
			HRU: 1, SRU: 1, CRU: 1, MRU: 0}}, nil)
		sub.EXPECT().GetDedicatedNodePrice(nodes[0]).Return(uint64(0), nil)
		sub.EXPECT().GetNode(nodes[1]).Return(&substrate.Node{ID: 2, TwinID: 2}, nil)
		sub.EXPECT().GetDedicatedNodePrice(nodes[1]).Return(uint64(0), nil)

		_, err := SetConfig(sub, inputs)
		assert.Error(t, err)
	})

	t.Run("test invalid json no hru", func(t *testing.T) {
		// configs mocks
		sub.EXPECT().GetFarm(inputs.FarmID).Return(&substrate.Farm{ID: 1}, nil)
		nodes := []uint32{1, 2}
		sub.EXPECT().GetNodes(inputs.FarmID).Return(nodes, nil)
		sub.EXPECT().GetNode(nodes[0]).Return(&substrate.Node{ID: 1, TwinID: 1, Resources: substrate.Resources{
			HRU: 0, SRU: 1, CRU: 1, MRU: 1}}, nil)
		sub.EXPECT().GetDedicatedNodePrice(nodes[0]).Return(uint64(0), nil)
		sub.EXPECT().GetNode(nodes[1]).Return(&substrate.Node{ID: 2, TwinID: 2}, nil)
		sub.EXPECT().GetDedicatedNodePrice(nodes[1]).Return(uint64(0), nil)

		_, err := SetConfig(sub, inputs)
		assert.Error(t, err)
	})

	// TODO:
	// t.Run("test invalid json node over provision CPU", func(t *testing.T) {
	// 	nodeContent := `{ "ID": 1, "twin_id" : 1, "resources": { "overprovision_cpu": 5, "total": { "SRU": 1, "CRU": 1, "HRU": 1, "MRU": 1 } } }`
	// 	content := fmt.Sprintf(`
	// 	{
	// 		"nodes": [ %v, {} ],
	// 		"farm": %v,
	// 		"power": {}
	// 	}
	// 	`, nodeContent, farmContent)

	// 	_, err := ParseIntoConfig([]byte(content), "json")
	// 	assert.Error(t, err)
	// })

	t.Run("test invalid json no farm ID", func(t *testing.T) {
		// configs mocks
		sub.EXPECT().GetFarm(inputs.FarmID).Return(&substrate.Farm{ID: 0}, nil)
		nodes := []uint32{}
		sub.EXPECT().GetNodes(inputs.FarmID).Return(nodes, nil)

		_, err := SetConfig(sub, inputs)
		assert.Error(t, err)
	})
}

func TestConfigModel(t *testing.T) {
	config := Config{
		Nodes: []Node{{
			Node: substrate.Node{ID: 1},
		}, {
			Node: substrate.Node{ID: 2},
		}},
		Mutex: new(sync.Mutex),
	}

	t.Run("test get node by ID", func(t *testing.T) {
		node, err := config.GetNodeByNodeID(1)
		assert.NoError(t, err)
		assert.Equal(t, node.ID, config.Nodes[0].ID)
	})

	t.Run("test get node by ID (not found)", func(t *testing.T) {
		_, err := config.GetNodeByNodeID(10)
		assert.Error(t, err)
	})

	t.Run("test update node", func(t *testing.T) {
		err := config.UpdateNode(Node{Node: substrate.Node{ID: 1, TwinID: 1}})
		assert.NoError(t, err)
		assert.Equal(t, uint32(config.Nodes[0].TwinID), uint32(1))
	})

	t.Run("test update node (not found)", func(t *testing.T) {
		err := config.UpdateNode(Node{Node: substrate.Node{ID: 10}})
		assert.Error(t, err)
	})

	t.Run("test filter nodes (power state)", func(t *testing.T) {
		nodes := config.FilterNodesPower([]PowerState{ON})
		assert.Equal(t, len(nodes), len(config.Nodes))
	})

	t.Run("test filter nodes (power state)", func(t *testing.T) {
		nodes := config.FilterNodesPower([]PowerState{ShuttingDown})
		assert.Empty(t, nodes)
	})

	t.Run("test filter allowed nodes to shut down", func(t *testing.T) {
		nodes := config.FilterAllowedNodesToShutDown()
		assert.Equal(t, len(nodes), len(config.Nodes))
	})
}
