// Package manager provides how to manage powers, powers and power
package manager

import (
	"errors"
	"testing"
	"time"

	types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/mocks"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/models"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/parser"
)

func TestPowerManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	sub := mocks.NewMockSub(ctrl)

	var identity substrate.Identity
	sub.EXPECT().NewIdentityFromSr25519Phrase("bad mnemonic").Return(identity, errors.New("error"))
	identity, err := sub.NewIdentityFromSr25519Phrase("bad mnemonic")
	assert.Error(t, err)

	config, err := parser.ParseIntoConfig([]byte(configContent), "json")
	assert.NoError(t, err)

	powerManager := NewPowerManager(identity, sub, &config)

	t.Run("test valid power on: already on", func(t *testing.T) {
		err = powerManager.PowerOn(config.Nodes[0].ID)
		assert.NoError(t, err)
	})

	t.Run("test valid power on: already waking up", func(t *testing.T) {
		config.Nodes[0].PowerState = models.WakingUP

		err = powerManager.PowerOn(config.Nodes[0].ID)
		assert.NoError(t, err)

		config.Nodes[0].PowerState = models.ON
	})

	t.Run("test valid power on", func(t *testing.T) {
		config.Nodes[0].PowerState = models.OFF

		sub.EXPECT().SetNodePowerState(powerManager.identity, true).Return(types.Hash{}, nil)

		err = powerManager.PowerOn(config.Nodes[0].ID)
		assert.NoError(t, err)
	})

	t.Run("test invalid power on: node not found", func(t *testing.T) {
		err = powerManager.PowerOn(3)
		assert.Error(t, err)
	})

	t.Run("test invalid power on: set node failed", func(t *testing.T) {
		config.Nodes[0].PowerState = models.OFF

		sub.EXPECT().SetNodePowerState(powerManager.identity, true).Return(types.Hash{}, errors.New("error"))

		err = powerManager.PowerOn(config.Nodes[0].ID)
		assert.Error(t, err)

		config.Nodes[0].PowerState = models.ON
	})

	t.Run("test valid power off: already off", func(t *testing.T) {
		config.Nodes[0].PowerState = models.OFF

		err = powerManager.PowerOff(config.Nodes[0].ID)
		assert.NoError(t, err)

		config.Nodes[0].PowerState = models.ON
	})

	t.Run("test valid power off: already shutting down", func(t *testing.T) {
		config.Nodes[0].PowerState = models.ShuttingDown

		err = powerManager.PowerOff(config.Nodes[0].ID)
		assert.NoError(t, err)

		config.Nodes[0].PowerState = models.ON
	})

	t.Run("test valid power off", func(t *testing.T) {
		sub.EXPECT().SetNodePowerState(powerManager.identity, false).Return(types.Hash{}, nil)

		err = powerManager.PowerOff(config.Nodes[0].ID)
		assert.NoError(t, err)

		config.Nodes[0].PowerState = models.ON
	})

	t.Run("test invalid power off: one node is on and cannot be off", func(t *testing.T) {
		config.Nodes[0].PowerState = models.OFF

		err = powerManager.PowerOff(config.Nodes[1].ID)
		assert.Error(t, err)

		config.Nodes[0].PowerState = models.ON
		config.Nodes[1].PowerState = models.ON
	})

	t.Run("test invalid power off: node is set to never shutdown", func(t *testing.T) {
		config.Nodes[0].NeverShutDown = true

		err = powerManager.PowerOff(config.Nodes[0].ID)
		assert.Error(t, err)

		config.Nodes[0].NeverShutDown = false
	})

	t.Run("test invalid power off: node has public config", func(t *testing.T) {
		config.Nodes[0].PublicConfig = true

		err = powerManager.PowerOff(config.Nodes[0].ID)
		assert.Error(t, err)

		config.Nodes[0].PublicConfig = false
	})

	t.Run("test invalid power off: node not found", func(t *testing.T) {
		err = powerManager.PowerOff(3)
		assert.Error(t, err)
	})

	t.Run("test invalid power off: set node power failed", func(t *testing.T) {
		sub.EXPECT().SetNodePowerState(powerManager.identity, false).Return(types.Hash{}, errors.New("error"))

		err = powerManager.PowerOff(config.Nodes[0].ID)
		assert.Error(t, err)
	})

	t.Run("test valid periodic wake up: already on", func(t *testing.T) {
		err = powerManager.PeriodicWakeUp()
		assert.NoError(t, err)
	})

	t.Run("test valid periodic wake up: off nodes", func(t *testing.T) {
		config.Nodes[0].PowerState = models.OFF

		sub.EXPECT().SetNodePowerState(powerManager.identity, true).Return(types.Hash{}, nil)

		err = powerManager.PeriodicWakeUp()
		assert.NoError(t, err)

		config.Nodes[0].PowerState = models.ON
	})

	t.Run("test valid periodic wake up: off nodes (failed to set power)", func(t *testing.T) {
		config.Nodes[0].PowerState = models.OFF

		sub.EXPECT().SetNodePowerState(powerManager.identity, true).Return(types.Hash{}, errors.New("error"))

		err = powerManager.PeriodicWakeUp()
		assert.NoError(t, err)

		config.Nodes[0].PowerState = models.ON
	})

	t.Run("test power management: a node to shutdown", func(t *testing.T) {
		config.Nodes[0].LastTimePowerStateChanged = time.Time{}
		config.Nodes[1].LastTimePowerStateChanged = time.Time{}
		sub.EXPECT().SetNodePowerState(powerManager.identity, false).Return(types.Hash{}, nil)

		err = powerManager.PowerManagement()
		assert.NoError(t, err)

		config.Nodes[0].PowerState = models.ON
		config.Nodes[1].PowerState = models.ON

		config.Nodes[0].LastTimePowerStateChanged = time.Time{}
		config.Nodes[1].LastTimePowerStateChanged = time.Time{}
	})

	t.Run("test power management: a node to shutdown (failed set the first node)", func(t *testing.T) {
		sub.EXPECT().SetNodePowerState(powerManager.identity, false).Return(types.Hash{}, errors.New("error"))
		sub.EXPECT().SetNodePowerState(powerManager.identity, false).Return(types.Hash{}, nil)

		err = powerManager.PowerManagement()
		assert.NoError(t, err)

		config.Nodes[1].PowerState = models.ON
		config.Nodes[0].LastTimePowerStateChanged = time.Time{}
		config.Nodes[1].LastTimePowerStateChanged = time.Time{}
	})

	t.Run("test power management: nothing to shut down", func(t *testing.T) {
		config.Nodes[0].PowerState = models.OFF

		err = powerManager.PowerManagement()
		assert.NoError(t, err)

		config.Nodes[0].PowerState = models.ON
	})

	t.Run("test power management: cannot shutdown public config", func(t *testing.T) {
		config.Nodes[0].PublicConfig = true
		config.Nodes[1].PublicConfig = true

		err = powerManager.PowerManagement()
		assert.NoError(t, err)

		config.Nodes[0].PublicConfig = false
		config.Nodes[1].PublicConfig = false
	})

	t.Run("test power management: node is waking up", func(t *testing.T) {
		config.Nodes[0].PowerState = models.WakingUP

		err = powerManager.PowerManagement()
		assert.NoError(t, err)

		config.Nodes[0].PowerState = models.ON
	})

	t.Run("test power management: a node to wake up (node 1 is used and node 2 is off)", func(t *testing.T) {
		// add an on node used
		config.Nodes[0].Resources.Used = nodeCapacity
		config.Nodes[1].PowerState = models.OFF

		sub.EXPECT().SetNodePowerState(powerManager.identity, true).Return(types.Hash{}, nil)

		err = powerManager.PowerManagement()
		assert.NoError(t, err)

		config.Nodes[0].Resources.Used = models.Capacity{}
		config.Nodes[1].PowerState = models.ON
		config.Nodes[0].LastTimePowerStateChanged = time.Time{}
		config.Nodes[1].LastTimePowerStateChanged = time.Time{}
	})

	t.Run("test power management: a node to wake up (node 1 has rent contract)", func(t *testing.T) {
		config.Nodes[0].HasActiveRentContract = true
		config.Nodes[1].PowerState = models.OFF

		sub.EXPECT().SetNodePowerState(powerManager.identity, true).Return(types.Hash{}, nil)

		err = powerManager.PowerManagement()
		assert.NoError(t, err)

		config.Nodes[0].HasActiveRentContract = false
		config.Nodes[1].PowerState = models.ON
		config.Nodes[0].LastTimePowerStateChanged = time.Time{}
		config.Nodes[1].LastTimePowerStateChanged = time.Time{}
	})

	t.Run("test invalid power management: no nodes to wake up (usage is high)", func(t *testing.T) {
		config.Nodes[0].Resources.Used = config.Nodes[0].Resources.Total
		config.Nodes[1].Resources.Used = config.Nodes[1].Resources.Total

		err = powerManager.PowerManagement()
		assert.Error(t, err)

		config.Nodes[0].Resources.Used = models.Capacity{}
		config.Nodes[1].Resources.Used = models.Capacity{}
	})

	t.Run("test valid power management: second node has no resources (usage is low)", func(t *testing.T) {
		config.Nodes[1].Resources.Total = models.Capacity{}

		err = powerManager.PowerManagement()
		assert.NoError(t, err)
	})

	t.Run("test valid power management: no resources", func(t *testing.T) {
		config.Nodes[0].Resources.Total = models.Capacity{}
		config.Nodes[1].Resources.Total = models.Capacity{}

		err = powerManager.PowerManagement()
		assert.NoError(t, err)
	})

	t.Run("test power on all nodes: 1 node is off", func(t *testing.T) {
		config.Nodes[1].PowerState = models.OFF

		sub.EXPECT().SetNodePowerState(powerManager.identity, true).Return(types.Hash{}, nil)

		err = powerManager.PowerOnAllNodes()
		assert.NoError(t, err)
	})

	t.Run("test power on all nodes: set node power failed", func(t *testing.T) {
		config.Nodes[1].PowerState = models.OFF

		sub.EXPECT().SetNodePowerState(powerManager.identity, true).Return(types.Hash{}, errors.New("error"))

		err = powerManager.PowerOnAllNodes()
		assert.Error(t, err)
	})
}
