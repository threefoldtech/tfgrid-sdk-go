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
			RawQuery: "farm_ids=1,2&status=up,standby&size=10",
		},
	}
	expectedFilter := types.NodeFilter{
		FarmIDs: []uint64{1, 2},
		Status:  []string{"up", "standby"},
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
	expectedFilter := types.NodeFilter{
		Status: []string{"up"},
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

func TestDecodesMixedParams(t *testing.T) {
	r := &http.Request{
		URL: &url.URL{
			RawQuery: "status=up&farm_storage=1024&size=10",
		},
	}

	gotFilter := types.NodeFilter{}
	gotLimit := types.Limit{}
	err := parseQueryParams(r, &gotFilter, &gotLimit)

	assert.Error(t, err)
}
