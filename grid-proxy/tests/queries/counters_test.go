package test

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	proxytypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

var statuses = []string{"up", "standby", "down"}

var statsFilterRandomValues = map[string]func() interface{}{
	"Status": func() interface{} {
		return &statuses[rand.Intn(3)]
	},
}

func TestStats(t *testing.T) {
	t.Parallel()
	t.Run("stats up test", func(t *testing.T) {
		f := proxytypes.StatsFilter{
			Status: &STATUS_UP,
		}

		want, err := mockClient.Stats(context.Background(), f)
		require.NoError(t, err)

		got, err := gridProxyClient.Stats(context.Background(), f)
		require.NoError(t, err)

		require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))
	})

	t.Run("stats all test", func(t *testing.T) {
		f := proxytypes.StatsFilter{}
		want, err := mockClient.Stats(context.Background(), f)
		require.NoError(t, err)

		got, err := gridProxyClient.Stats(context.Background(), f)
		require.NoError(t, err)

		require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))
	})
}

func TestStatsFilter(t *testing.T) {
	t.Parallel()

	f := proxytypes.StatsFilter{}
	fp := &f
	v := reflect.ValueOf(fp).Elem()

	for i := 0; i < v.NumField(); i++ {
		generator, ok := statsFilterRandomValues[v.Type().Field(i).Name]
		require.True(t, ok, "Filter field %s has no random value generator", v.Type().Field(i).Name)

		randomFieldValue := generator()
		if v.Field(i).Type().Kind() != reflect.Slice {
			v.Field(i).Set(reflect.New(v.Field(i).Type().Elem()))
		}
		v.Field(i).Set(reflect.ValueOf(randomFieldValue))

		want, err := mockClient.Stats(context.Background(), f)
		require.NoError(t, err)

		got, err := gridProxyClient.Stats(context.Background(), f)
		require.NoError(t, err)

		require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))

		v.Field(i).Set(reflect.Zero(v.Field(i).Type()))
	}
}
