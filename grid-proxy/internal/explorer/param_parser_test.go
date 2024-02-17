package explorer

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

func TestDecodesNodeFilterAndLimitParams(t *testing.T) {
	r := &http.Request{
		URL: &url.URL{
			RawQuery: "farm_ids=1,2&status=up&size=10",
		},
	}
	ids := []uint64{1, 2}
	status := "up"
	expectedFilter := types.NodeFilter{
		FarmIDs: ids,
		Status:  &status,
	}
	expectedLimit := types.Limit{
		Size: 10,
	}

	gotFilter := types.NodeFilter{}
	gotLimit := types.Limit{}
	err := parseQueryParams(r, &gotFilter, &gotLimit)
	assert.NoError(t, err)

	assert.Equal(t, expectedFilter, gotFilter)
	assert.Equal(t, expectedLimit, gotLimit)
}

func TestEmptyParam(t *testing.T) {
	r := &http.Request{
		URL: &url.URL{
			RawQuery: "free_mru=&status=up&size=10",
		},
	}
	status := "up"
	expectedFilter := types.NodeFilter{
		Status: &status,
	}
	expectedLimit := types.Limit{
		Size: 10,
	}

	gotFilter := types.NodeFilter{}
	gotLimit := types.Limit{}
	err := parseQueryParams(r, &gotFilter, &gotLimit)
	assert.NoError(t, err)

	assert.Equal(t, expectedFilter, gotFilter)
	assert.Equal(t, expectedLimit, gotLimit)
}

func TestUnknownParam(t *testing.T) {
	r := &http.Request{
		URL: &url.URL{
			RawQuery: "free_space=10",
		},
	}

	gotFilter := types.NodeFilter{}
	gotLimit := types.Limit{}
	err := parseQueryParams(r, &gotFilter, &gotLimit)

	assert.Error(t, err)
}
