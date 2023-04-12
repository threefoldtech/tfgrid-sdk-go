package deployer

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/grid_proxy_server/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/gridify/internal/mocks"
)

func TestFindNode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	filter := buildNodeFilter()

	clientMock := mocks.NewMockTFPluginClientInterface(ctrl)
	t.Run("error finding available nodes", func(t *testing.T) {
		clientMock.
			EXPECT().
			FilterNodes(filter, gomock.Any()).
			Return([]types.Node{}, 0, errors.New("error"))

		_, err := findNode(clientMock)
		assert.Error(t, err)
	})
	t.Run("no available nodes", func(t *testing.T) {
		clientMock.
			EXPECT().
			FilterNodes(filter, gomock.Any()).
			Return([]types.Node{}, 0, nil)

		_, err := findNode(clientMock)
		assert.Error(t, err)
	})
	t.Run("found nodes", func(t *testing.T) {
		clientMock.
			EXPECT().
			FilterNodes(filter, gomock.Any()).
			Return([]types.Node{{NodeID: 10}}, 0, nil)

		nodeID, err := findNode(clientMock)
		assert.NoError(t, err)
		assert.Equal(t, nodeID, uint32(10))
	})
}

func TestBuildPortlessBackend(t *testing.T) {
	ip := "10.10.10.10/24"
	got := buildPortlessBackend(ip)
	assert.Equal(t, got, "http://10.10.10.10")
}
