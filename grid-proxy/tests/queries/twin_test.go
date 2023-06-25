package test

import (
	"math/rand"
	"reflect"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	proxytypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	mock "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/tests/mock_client"
)

type TwinsAggregate struct {
	twinIDs    []uint32
	accountIDs []string
	relays     []string
	publicKeys []string
	twins      map[uint32]mock.DBTwin
}

const (
	TWINS_TESTS = 200
)

func TestTwins(t *testing.T) {
	t.Run("twins pagination test", func(t *testing.T) {
		t.Parallel()

		f := proxytypes.TwinFilter{}
		l := proxytypes.Limit{
			Size:     5,
			Page:     1,
			RetCount: true,
		}
		for {
			localTwins, localCount, err := mockClient.Twins(f, l)
			assert.NoError(t, err)

			remoteTwins, remoteCount, err := proxyClient.Twins(f, l)
			assert.NoError(t, err)

			require.Equal(t, localCount, remoteCount, serializeFilter(f))
			require.Equal(t, len(localTwins), len(remoteTwins), serializeFilter(f))

			require.True(t, reflect.DeepEqual(localTwins, remoteTwins), serializeFilter(f), cmp.Diff(localTwins, remoteTwins))

			if l.Page*l.Size >= uint64(localCount) {
				break
			}
			l.Page++
		}
	})

	t.Run("twins stress test", func(t *testing.T) {
		t.Parallel()

		agg := calcTwinsAggregates(&dbData)
		for i := 0; i < TWINS_TESTS; i++ {
			l := proxytypes.Limit{
				Size:     999999999999,
				Page:     1,
				RetCount: false,
			}
			f := randomTwinsFilter(&agg)

			localTwins, _, err := mockClient.Twins(f, l)
			assert.NoError(t, err)

			remoteTwins, _, err := proxyClient.Twins(f, l)
			assert.NoError(t, err)

			require.Equal(t, len(localTwins), len(remoteTwins), serializeFilter(f))

			require.True(t, reflect.DeepEqual(localTwins, remoteTwins), serializeFilter(f), cmp.Diff(localTwins, remoteTwins))
		}
	})
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

func randomTwinsFilter(agg *TwinsAggregate) proxytypes.TwinFilter {
	var f proxytypes.TwinFilter

	twinID := twinRandTwinID(agg)

	f.TwinID = twinID
	f.AccountID = twinRandAccountID(agg, twinID)
	f.Relay = twinRandRelay(agg, twinID)
	f.PublicKey = twinRandPublicKey(agg, twinID)

	return f
}

func twinRandTwinID(agg *TwinsAggregate) *uint32 {
	if flip(.5) {
		c := agg.twinIDs[rand.Intn(len(agg.twinIDs))]
		return &c
	}

	return nil
}

func twinRandAccountID(agg *TwinsAggregate, twinID *uint32) *string {
	if flip(.5) {
		if twinID != nil && flip(.4) {
			accountID := agg.twins[*twinID].AccountID
			return &accountID
		} else {
			c := agg.accountIDs[rand.Intn(len(agg.accountIDs))]
			return &c
		}
	}

	return nil
}

func twinRandRelay(agg *TwinsAggregate, twinID *uint32) *string {
	if flip(.5) {
		if twinID != nil && flip(.4) {
			relay := agg.twins[*twinID].AccountID
			return &relay
		} else {
			c := agg.relays[rand.Intn(len(agg.relays))]
			return &c
		}
	}

	return nil
}

func twinRandPublicKey(agg *TwinsAggregate, twinID *uint32) *string {
	if flip(.5) {
		if twinID != nil && flip(.4) {
			publicKey := agg.twins[*twinID].AccountID
			return &publicKey
		} else {
			c := agg.publicKeys[rand.Intn(len(agg.publicKeys))]
			return &c
		}
	}

	return nil
}
