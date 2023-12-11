package internal

// TODO:
// import (
// 	"errors"
// 	"fmt"
// 	"testing"
// 	"time"

// 	types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
// 	"github.com/golang/mock/gomock"
// 	"github.com/stretchr/testify/assert"
// 	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
// 	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/mocks"
// )

// var nodeCapacity = models.Capacity{
// 	CRU: 1,
// 	SRU: 1,
// 	MRU: 1,
// 	HRU: 1,
// }

// var configContent = `
// {
// 	"included_nodes": [ 1, 2 ],
// 	"farm_id": 1,
// 	"power": { "periodic_wake_up_start": "08:30AM", "wake_up_threshold": 30 }
// }`

// func TestPowerManager(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()
// 	sub := mocks.NewMockSub(ctrl)

// 	identity, err := substrate.NewIdentityFromSr25519Phrase("bad mnemonic")
// 	assert.Error(t, err)

// 	// configs
// 	farmID := uint32(1)
// 	sub.EXPECT().GetFarm(farmID).Return(&substrate.Farm{ID: 1}, nil)
// 	nodes := []uint32{1, 2}
// 	sub.EXPECT().GetNodes(farmID).Return(nodes, nil)
// 	sub.EXPECT().GetNode(nodes[0]).Return(&substrate.Node{ID: 1, TwinID: 1, Resources: substrate.Resources{
// 		HRU: 1, SRU: 1, CRU: 1, MRU: 1}}, nil)
// 	sub.EXPECT().GetDedicatedNodePrice(nodes[0]).Return(uint64(0), nil)
// 	sub.EXPECT().GetNode(nodes[1]).Return(&substrate.Node{ID: 2, TwinID: 2, Resources: substrate.Resources{
// 		HRU: 1, SRU: 1, CRU: 1, MRU: 1}}, nil)
// 	sub.EXPECT().GetDedicatedNodePrice(nodes[1]).Return(uint64(0), nil)

// 	inputs, err := parser.ParseIntoInputConfig([]byte(configContent), "json")
// 	assert.NoError(t, err)

// 	var config models.Config
// 	err = config.Set(sub, inputs)
// 	assert.NoError(t, err)

// 	powerManager := NewPowerManager(identity, sub, &config)

// 	t.Run("test valid power on: already on", func(t *testing.T) {
// 		err = powerManager.PowerOn(uint32(config.Nodes[0].ID))
// 		assert.NoError(t, err)
// 	})

// 	t.Run("test valid power on: already waking up", func(t *testing.T) {
// 		config.Nodes[0].PowerState = models.WakingUP

// 		err = powerManager.PowerOn(uint32(config.Nodes[0].ID))
// 		assert.NoError(t, err)

// 		config.Nodes[0].PowerState = models.ON
// 	})

// 	t.Run("test valid power on", func(t *testing.T) {
// 		config.Nodes[0].PowerState = models.OFF

// 		sub.EXPECT().SetNodePowerTarget(powerManager.identity, uint32(config.Nodes[0].ID), true).Return(types.Hash{}, nil)

// 		err = powerManager.PowerOn(uint32(config.Nodes[0].ID))
// 		assert.NoError(t, err)
// 	})

// 	t.Run("test invalid power on: node not found", func(t *testing.T) {
// 		err = powerManager.PowerOn(3)
// 		assert.Error(t, err)
// 	})

// 	t.Run("test invalid power on: set node failed", func(t *testing.T) {
// 		config.Nodes[0].PowerState = models.OFF

// 		sub.EXPECT().SetNodePowerTarget(powerManager.identity, uint32(config.Nodes[0].ID), true).Return(types.Hash{}, errors.New("error"))

// 		err = powerManager.PowerOn(uint32(config.Nodes[0].ID))
// 		assert.Error(t, err)

// 		config.Nodes[0].PowerState = models.ON
// 	})

// 	t.Run("test valid power off: already off", func(t *testing.T) {
// 		config.Nodes[0].PowerState = models.OFF

// 		err = powerManager.PowerOff(uint32(config.Nodes[0].ID))
// 		assert.NoError(t, err)

// 		config.Nodes[0].PowerState = models.ON
// 	})

// 	t.Run("test valid power off: already shutting down", func(t *testing.T) {
// 		config.Nodes[0].PowerState = models.ShuttingDown

// 		err = powerManager.PowerOff(uint32(config.Nodes[0].ID))
// 		assert.NoError(t, err)

// 		config.Nodes[0].PowerState = models.ON
// 	})

// 	t.Run("test valid power off", func(t *testing.T) {
// 		sub.EXPECT().SetNodePowerTarget(powerManager.identity, uint32(config.Nodes[0].ID), false).Return(types.Hash{}, nil)

// 		err = powerManager.PowerOff(uint32(config.Nodes[0].ID))
// 		assert.NoError(t, err)

// 		config.Nodes[0].PowerState = models.ON
// 	})

// 	t.Run("test invalid power off: one node is on and cannot be off", func(t *testing.T) {
// 		config.Nodes[0].PowerState = models.OFF

// 		err = powerManager.PowerOff(uint32(config.Nodes[1].ID))
// 		assert.Error(t, err)

// 		config.Nodes[0].PowerState = models.ON
// 		config.Nodes[1].PowerState = models.ON
// 	})

// 	t.Run("test invalid power off: node is set to never shutdown", func(t *testing.T) {
// 		config.Nodes[0].NeverShutDown = true

// 		err = powerManager.PowerOff(uint32(config.Nodes[0].ID))
// 		assert.Error(t, err)

// 		config.Nodes[0].NeverShutDown = false
// 	})

// 	t.Run("test invalid power off: node has public config", func(t *testing.T) {
// 		config.Nodes[0].PublicConfig.HasValue = true

// 		err = powerManager.PowerOff(uint32(config.Nodes[0].ID))
// 		assert.Error(t, err)

// 		config.Nodes[0].PublicConfig.HasValue = false
// 	})

// 	t.Run("test invalid power off: node not found", func(t *testing.T) {
// 		err = powerManager.PowerOff(3)
// 		assert.Error(t, err)
// 	})

// 	t.Run("test invalid power off: set node power failed", func(t *testing.T) {
// 		sub.EXPECT().SetNodePowerTarget(powerManager.identity, uint32(config.Nodes[0].ID), false).Return(types.Hash{}, errors.New("error"))

// 		err = powerManager.PowerOff(uint32(config.Nodes[0].ID))
// 		assert.Error(t, err)
// 	})

// 	t.Run("test valid periodic wake up: already on", func(t *testing.T) {
// 		err = powerManager.PeriodicWakeUp()
// 		assert.NoError(t, err)
// 	})

// 	t.Run("test valid periodic wake up: off nodes", func(t *testing.T) {
// 		config.Nodes[0].PowerState = models.OFF

// 		sub.EXPECT().SetNodePowerTarget(powerManager.identity, uint32(config.Nodes[0].ID), true).Return(types.Hash{}, nil)

// 		err = powerManager.PeriodicWakeUp()
// 		assert.NoError(t, err)

// 		config.Nodes[0].PowerState = models.ON
// 	})

// 	t.Run("test valid periodic wake up: off nodes (failed to set power)", func(t *testing.T) {
// 		config.Nodes[0].PowerState = models.OFF

// 		sub.EXPECT().SetNodePowerTarget(powerManager.identity, uint32(config.Nodes[0].ID), true).Return(types.Hash{}, errors.New("error"))

// 		err = powerManager.PeriodicWakeUp()
// 		assert.NoError(t, err)

// 		config.Nodes[0].PowerState = models.ON
// 	})

// 	t.Run("test power management: a node to shutdown", func(t *testing.T) {
// 		config.Nodes[0].LastTimePowerStateChanged = time.Time{}
// 		config.Nodes[1].LastTimePowerStateChanged = time.Time{}
// 		sub.EXPECT().SetNodePowerTarget(powerManager.identity, uint32(config.Nodes[0].ID), false).Return(types.Hash{}, nil)

// 		err = powerManager.PowerManagement()
// 		assert.NoError(t, err)

// 		config.Nodes[0].PowerState = models.ON
// 		config.Nodes[1].PowerState = models.ON

// 		config.Nodes[0].LastTimePowerStateChanged = time.Time{}
// 		config.Nodes[1].LastTimePowerStateChanged = time.Time{}
// 	})

// 	t.Run("test power management: a node to shutdown (failed set the first node)", func(t *testing.T) {
// 		sub.EXPECT().SetNodePowerTarget(powerManager.identity, uint32(config.Nodes[0].ID), false).Return(types.Hash{}, errors.New("error"))
// 		sub.EXPECT().SetNodePowerTarget(powerManager.identity, uint32(config.Nodes[1].ID), false).Return(types.Hash{}, nil)

// 		err = powerManager.PowerManagement()
// 		assert.NoError(t, err)

// 		config.Nodes[1].PowerState = models.ON
// 		config.Nodes[0].LastTimePowerStateChanged = time.Time{}
// 		config.Nodes[1].LastTimePowerStateChanged = time.Time{}
// 	})

// 	t.Run("test power management: nothing to shut down", func(t *testing.T) {
// 		config.Nodes[0].PowerState = models.OFF

// 		err = powerManager.PowerManagement()
// 		assert.NoError(t, err)

// 		config.Nodes[0].PowerState = models.ON
// 	})

// 	t.Run("test power management: cannot shutdown public config", func(t *testing.T) {
// 		config.Nodes[0].PublicConfig.HasValue = true
// 		config.Nodes[1].PublicConfig.HasValue = true

// 		err = powerManager.PowerManagement()
// 		assert.NoError(t, err)

// 		config.Nodes[0].PublicConfig.HasValue = false
// 		config.Nodes[1].PublicConfig.HasValue = false
// 	})

// 	t.Run("test power management: node is waking up", func(t *testing.T) {
// 		config.Nodes[0].PowerState = models.WakingUP

// 		err = powerManager.PowerManagement()
// 		assert.NoError(t, err)

// 		config.Nodes[0].PowerState = models.ON
// 	})

// 	t.Run("test power management: a node to wake up (node 1 is used and node 2 is off)", func(t *testing.T) {
// 		// add an on node used
// 		config.Nodes[0].Resources.Used = nodeCapacity
// 		config.Nodes[1].PowerState = models.OFF

// 		sub.EXPECT().SetNodePowerTarget(powerManager.identity, uint32(config.Nodes[1].ID), true).Return(types.Hash{}, nil)

// 		err = powerManager.PowerManagement()
// 		assert.NoError(t, err)

// 		config.Nodes[0].Resources.Used = models.Capacity{}
// 		config.Nodes[1].PowerState = models.ON
// 		config.Nodes[0].LastTimePowerStateChanged = time.Time{}
// 		config.Nodes[1].LastTimePowerStateChanged = time.Time{}
// 	})

// 	t.Run("test power management: a node to wake up (node 1 has rent contract)", func(t *testing.T) {
// 		config.Nodes[0].HasActiveRentContract = true
// 		config.Nodes[1].PowerState = models.OFF

// 		sub.EXPECT().SetNodePowerTarget(powerManager.identity, uint32(config.Nodes[1].ID), true).Return(types.Hash{}, nil)

// 		err = powerManager.PowerManagement()
// 		assert.NoError(t, err)

// 		config.Nodes[0].HasActiveRentContract = false
// 		config.Nodes[1].PowerState = models.ON
// 		config.Nodes[0].LastTimePowerStateChanged = time.Time{}
// 		config.Nodes[1].LastTimePowerStateChanged = time.Time{}
// 	})

// 	t.Run("test invalid power management: no nodes to wake up (usage is high)", func(t *testing.T) {
// 		config.Nodes[0].Resources.Used = config.Nodes[0].Resources.Total
// 		config.Nodes[1].Resources.Used = config.Nodes[1].Resources.Total

// 		err = powerManager.PowerManagement()
// 		assert.Error(t, err)

// 		config.Nodes[0].Resources.Used = models.Capacity{}
// 		config.Nodes[1].Resources.Used = models.Capacity{}
// 	})

// 	t.Run("test valid power management: second node has no resources (usage is low)", func(t *testing.T) {
// 		config.Nodes[1].Resources.Total = models.Capacity{}

// 		err = powerManager.PowerManagement()
// 		assert.NoError(t, err)
// 	})

// 	t.Run("test valid power management: no resources", func(t *testing.T) {
// 		config.Nodes[0].Resources.Total = models.Capacity{}
// 		config.Nodes[1].Resources.Total = models.Capacity{}

// 		err = powerManager.PowerManagement()
// 		assert.NoError(t, err)
// 	})

// 	t.Run("test power on all nodes: 1 node is off", func(t *testing.T) {
// 		config.Nodes[1].PowerState = models.OFF

// 		sub.EXPECT().SetNodePowerTarget(powerManager.identity, uint32(config.Nodes[1].ID), true).Return(types.Hash{}, nil)

// 		err = powerManager.PowerOnAllNodes()
// 		assert.NoError(t, err)
// 	})

// 	t.Run("test power on all nodes: set node power failed", func(t *testing.T) {
// 		config.Nodes[1].PowerState = models.OFF

// 		sub.EXPECT().SetNodePowerTarget(powerManager.identity, uint32(config.Nodes[1].ID), true).Return(types.Hash{}, errors.New("error"))

// 		err = powerManager.PowerOnAllNodes()
// 		assert.Error(t, err)
// 	})
// }
