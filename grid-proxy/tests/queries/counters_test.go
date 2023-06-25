package test

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	proxytypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

func TestCounters(t *testing.T) {

	t.Run("counters up test", func(t *testing.T) {
		t.Parallel()

		f := proxytypes.StatsFilter{
			Status: &STATUS_UP,
		}

		localCounters, err := mockClient.Counters(f)
		assert.NoError(t, err)

		remoteCounters, err := proxyClient.Counters(f)
		assert.NoError(t, err)

		require.True(t, reflect.DeepEqual(localCounters, remoteCounters), serializeFilter(f), cmp.Diff(localCounters, remoteCounters))
	})

	t.Run("counters all test", func(t *testing.T) {
		t.Parallel()

		f := proxytypes.StatsFilter{}
		localCounters, err := mockClient.Counters(f)
		assert.NoError(t, err)

		remoteCounters, err := proxyClient.Counters(f)
		assert.NoError(t, err)

		require.True(t, reflect.DeepEqual(localCounters, remoteCounters), serializeFilter(f), cmp.Diff(localCounters, remoteCounters))
	})
}
