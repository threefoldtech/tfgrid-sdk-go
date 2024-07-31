package parser

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/internal"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/mocks"
)

func TestValidateInput(t *testing.T) {
	config := internal.Config{FarmID: uint32(25)}
	ctrl := gomock.NewController(t)
	mockGetNodes := mocks.NewMockSubstrate(ctrl)
	mockGetNodes.EXPECT().GetNodes(config.FarmID).Times(6).Return([]uint32{20, 21, 22, 23, 24, 30, 31, 32, 34, 40, 41}, nil)
	t.Run("test valid include, exclude and priority nodes", func(t *testing.T) {
		config.IncludedNodes = []uint32{20, 21, 22, 30, 31, 32, 40, 41}
		config.ExcludedNodes = []uint32{23, 24, 34}
		config.PriorityNodes = []uint32{20, 21}
		got := validateInput(config, mockGetNodes)
		assert.NoError(t, got)
	})
	t.Run("test invalid include", func(t *testing.T) {
		config.IncludedNodes = []uint32{26, 27}
		got := validateInput(config, mockGetNodes)
		assert.Error(t, got)
	})
	t.Run("test invalid exclude", func(t *testing.T) {
		config.ExcludedNodes = []uint32{26, 27}
		got := validateInput(config, mockGetNodes)
		assert.Error(t, got)
	})
	t.Run("test invalid priority", func(t *testing.T) {
		config.IncludedNodes = []uint32{21}
		config.PriorityNodes = []uint32{20, 21}
		got := validateInput(config, mockGetNodes)
		assert.Error(t, got)
	})
	t.Run("test overlapping nodes in include and exclude", func(t *testing.T) {
		config.IncludedNodes = []uint32{21}
		config.ExcludedNodes = []uint32{20, 21}
		got := validateInput(config, mockGetNodes)
		assert.Error(t, got)
	})
	t.Run("test overlapping nodes in priority and exclude", func(t *testing.T) {
		config.IncludedNodes = []uint32{21}
		config.PriorityNodes = []uint32{21}
		config.ExcludedNodes = []uint32{20, 21}
		got := validateInput(config, mockGetNodes)
		assert.Error(t, got)
	})

}
