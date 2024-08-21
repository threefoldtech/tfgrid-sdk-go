package parser

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/internal"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/mocks"
)

var testCases = []struct {
	name               string
	includedNodes      []uint32
	priorityNodes      []uint32
	excludedNodes      []uint32
	neverShutdownNodes []uint32
	shouldFail         bool
}{
	{
		name:               "test include, exclude, priority nodes and never shutdown nodes found in farm nodes",
		includedNodes:      []uint32{20, 21, 22, 30, 31, 32, 40, 41},
		priorityNodes:      []uint32{20, 21},
		excludedNodes:      []uint32{23, 24, 34},
		neverShutdownNodes: []uint32{22, 30},
		shouldFail:         false,
	},
	{
		name:          "test included node doesn't exist in farm nodes",
		includedNodes: []uint32{26, 27},
		shouldFail:    true,
	},
	{
		name:          "test priority node doesn't exist in included nodes",
		includedNodes: []uint32{21},
		priorityNodes: []uint32{20, 21},
		shouldFail:    true,
	},
	{
		name:               "test never shutdown node doesn't exist in included nodes",
		includedNodes:      []uint32{21},
		neverShutdownNodes: []uint32{20, 21},
		shouldFail:         true,
	}, {
		name:          "test overlapping nodes in included and excluded nodes",
		includedNodes: []uint32{21},
		excludedNodes: []uint32{20, 21},
		shouldFail:    true,
	}, {
		name:          "test overlapping nodes in priority and excluded nodes",
		includedNodes: []uint32{21},
		priorityNodes: []uint32{21},
		excludedNodes: []uint32{20, 21},
		shouldFail:    true,
	}, {
		name:               "test overlapping nodes in never shutdown and excluded nodes",
		includedNodes:      []uint32{21},
		excludedNodes:      []uint32{20, 21},
		neverShutdownNodes: []uint32{21},
		shouldFail:         true,
	}, {
		name:          "test excluded node doesn't exist in farm nodes",
		excludedNodes: []uint32{26, 27},
		shouldFail:    true,
	}, {
		name:               "test all nodes included and other nodes exist in included nodes",
		priorityNodes:      []uint32{21},
		excludedNodes:      []uint32{22},
		neverShutdownNodes: []uint32{20},
		shouldFail:         false,
	}, {
		name:          "test all nodes included and priority nodes doesn't exist in included nodes",
		priorityNodes: []uint32{27, 26},
		shouldFail:    true,
	}, {
		name:               "test all nodes included and shutdown nodes doesn't exist in included nodes",
		neverShutdownNodes: []uint32{27, 26},
		shouldFail:         true,
	}, {
		name:               "test all nodes included and overlapping node between shutdown and excluded nodes",
		neverShutdownNodes: []uint32{21, 20},
		excludedNodes:      []uint32{21, 20},
		shouldFail:         true,
	}, {
		name:          "test all nodes included and overlapping node between priority and excluded nodes",
		excludedNodes: []uint32{21, 20},
		priorityNodes: []uint32{21, 20},
		shouldFail:    true,
	},
}

func TestValidateInput(t *testing.T) {
	config := internal.Config{FarmID: uint32(25)}
	ctrl := gomock.NewController(t)
	mockGetNodes := mocks.NewMockSubstrate(ctrl)
	mockGetNodes.EXPECT().GetNodes(config.FarmID).Times(13).Return([]uint32{20, 21, 22, 23, 24, 30, 31, 32, 34, 40, 41}, nil)
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			config.IncludedNodes = test.includedNodes
			config.ExcludedNodes = test.excludedNodes
			config.PriorityNodes = test.priorityNodes
			config.NeverShutDownNodes = test.neverShutdownNodes
			got := validateInput(config, mockGetNodes)
			if test.shouldFail {
				assert.Error(t, got)
			} else {
				assert.NoError(t, got)
			}
		})
	}

}
