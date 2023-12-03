package test

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	proxytypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	mock "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/tests/queries/mock_client"
)

type TwinsAggregate struct {
	twinIDs    []uint64
	accountIDs []string
	relays     []string
	publicKeys []string
	twins      map[uint64]mock.Twin
}

const (
	TWINS_TESTS = 200
)

var twinFilterRandomValueGenerator = map[string]func(agg TwinsAggregate) interface{}{
	"TwinID": func(agg TwinsAggregate) interface{} {
		return &agg.twinIDs[rand.Intn(len(agg.twinIDs))]
	},
	"AccountID": func(agg TwinsAggregate) interface{} {
		return &agg.accountIDs[rand.Intn(len(agg.accountIDs))]
	},
	"Relay": func(agg TwinsAggregate) interface{} {
		return &agg.relays[rand.Intn(len(agg.relays))]
	},
	"PublicKey": func(agg TwinsAggregate) interface{} {
		return &agg.publicKeys[rand.Intn(len(agg.publicKeys))]
	},
}

func TestTwins(t *testing.T) {
	t.Run("twins pagination test", func(t *testing.T) {
		f := proxytypes.TwinFilter{}
		l := proxytypes.Limit{
			Size:     5,
			Page:     1,
			RetCount: true,
		}
		for {
			want, wantCount, err := mockClient.Twins(context.Background(), f, l)
			require.NoError(t, err)

			got, gotCount, err := gridProxyClient.Twins(context.Background(), f, l)
			require.NoError(t, err)

			assert.Equal(t, wantCount, gotCount)

			require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))

			if l.Page*l.Size >= uint64(wantCount) {
				break
			}
			l.Page++
		}
	})

	t.Run("twins stress test", func(t *testing.T) {
		agg := calcTwinsAggregates(&data)
		for i := 0; i < TWINS_TESTS; i++ {
			l := proxytypes.Limit{
				Size:     999999999999,
				Page:     1,
				RetCount: true,
			}
			f, err := randomTwinsFilter(&agg)
			require.NoError(t, err)

			want, wantCount, err := mockClient.Twins(context.Background(), f, l)
			require.NoError(t, err)

			got, gotCount, err := gridProxyClient.Twins(context.Background(), f, l)
			require.NoError(t, err)

			assert.Equal(t, wantCount, gotCount, fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)))

			require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))
		}
	})
}

// TestTwinFilter iterates over all TwinFilter fields, and for each one generates a random value, then runs a test between the mock client and the gridproxy client
func TestTwinFilter(t *testing.T) {
	f := proxytypes.TwinFilter{}
	fp := &f
	v := reflect.ValueOf(fp).Elem()
	l := proxytypes.Limit{
		Size:     9999999,
		Page:     1,
		RetCount: true,
	}

	agg := calcTwinsAggregates(&data)

	for i := 0; i < v.NumField(); i++ {
		generator, ok := twinFilterRandomValueGenerator[v.Type().Field(i).Name]
		require.True(t, ok, "Filter field %s has no random value generator", v.Type().Field(i).Name)

		randomFieldValue := generator(agg)

		if v.Field(i).Type().Kind() != reflect.Slice {
			v.Field(i).Set(reflect.New(v.Field(i).Type().Elem()))
		}
		v.Field(i).Set(reflect.ValueOf(randomFieldValue))

		want, wantCount, err := mockClient.Twins(context.Background(), f, l)
		require.NoError(t, err)

		got, gotCount, err := gridProxyClient.Twins(context.Background(), f, l)
		require.NoError(t, err)

		assert.Equal(t, wantCount, gotCount)

		require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))

		v.Field(i).Set(reflect.Zero(v.Field(i).Type()))
	}
}

func randomTwinsFilter(agg *TwinsAggregate) (proxytypes.TwinFilter, error) {
	f := proxytypes.TwinFilter{}
	fp := &f
	v := reflect.ValueOf(fp).Elem()

	for i := 0; i < v.NumField(); i++ {
		if rand.Float32() > .5 {
			_, ok := twinFilterRandomValueGenerator[v.Type().Field(i).Name]
			if !ok {
				return proxytypes.TwinFilter{}, fmt.Errorf("Filter field %s has no random value generator", v.Type().Field(i).Name)
			}

			randomFieldValue := twinFilterRandomValueGenerator[v.Type().Field(i).Name](*agg)
			if v.Field(i).Type().Kind() != reflect.Slice {
				v.Field(i).Set(reflect.New(v.Field(i).Type().Elem()))
			}
			v.Field(i).Set(reflect.ValueOf(randomFieldValue))
		}
	}

	return f, nil
}

func calcTwinsAggregates(data *mock.DBData) (res TwinsAggregate) {
	for _, twin := range data.Twins {
		res.twinIDs = append(res.twinIDs, twin.TwinID)
		res.accountIDs = append(res.accountIDs, twin.AccountID)
		res.relays = append(res.relays, twin.Relay)
		res.publicKeys = append(res.publicKeys, twin.PublicKey)
	}
	res.twins = data.Twins
	sort.Slice(res.twinIDs, func(i, j int) bool {
		return res.twinIDs[i] < res.twinIDs[j]
	})
	sort.Slice(res.accountIDs, func(i, j int) bool {
		return res.accountIDs[i] < res.accountIDs[j]
	})
	sort.Slice(res.relays, func(i, j int) bool {
		return res.relays[i] < res.relays[j]
	})
	sort.Slice(res.publicKeys, func(i, j int) bool {
		return res.publicKeys[i] < res.publicKeys[j]
	})
	return
}
