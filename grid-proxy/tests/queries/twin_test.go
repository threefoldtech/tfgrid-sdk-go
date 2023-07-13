package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	proxyclient "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
	proxytypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

type TwinsAggregate struct {
	twinIDs    []uint64
	accountIDs []string
	relays     []string
	publicKeys []string
	twins      map[uint64]twin
}

const (
	TWINS_TESTS = 200
)

func TestTwins(t *testing.T) {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSSWORD, POSTGRES_DB)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(errors.Wrap(err, "failed to open db"))
	}
	defer db.Close()

	data, err := load(db)
	if err != nil {
		panic(err)
	}
	proxyClient := proxyclient.NewClient(ENDPOINT)
	localClient := NewGridProxyClient(data)

	t.Run("twins pagination test", func(t *testing.T) {
		f := proxytypes.TwinFilter{}
		l := proxytypes.Limit{
			Size:     5,
			Page:     1,
			RetCount: true,
		}
		for {
			localTwins, localCount, err := localClient.Twins(f, l)
			assert.NoError(t, err)
			remoteTwins, remoteCount, err := proxyClient.Twins(f, l)
			assert.NoError(t, err)
			assert.Equal(t, localCount, remoteCount)
			require.True(t, reflect.DeepEqual(localTwins, remoteTwins), serializeFilter(f), cmp.Diff(localTwins, remoteTwins))

			if l.Page*l.Size >= uint64(localCount) {
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
				RetCount: false,
			}
			f := randomTwinsFilter(&agg)
			localTwins, _, err := localClient.Twins(f, l)
			assert.NoError(t, err)
			remoteTwins, _, err := proxyClient.Twins(f, l)
			assert.NoError(t, err)
			require.True(t, reflect.DeepEqual(localTwins, remoteTwins), serializeFilter(f), cmp.Diff(localTwins, remoteTwins))

		}
	})
}

func randomTwinsFilter(agg *TwinsAggregate) proxytypes.TwinFilter {
	var f proxytypes.TwinFilter
	if flip(.2) {
		c := agg.twinIDs[rand.Intn(len(agg.twinIDs))]
		f.TwinID = &c
	}
	if flip(.2) {
		if f.TwinID != nil && flip(.4) {
			accountID := agg.twins[*f.TwinID].account_id
			f.AccountID = &accountID
		} else {
			c := agg.accountIDs[rand.Intn(len(agg.accountIDs))]
			f.AccountID = &c
		}
	}
	if flip(.2) {
		if f.TwinID != nil && flip(.4) {
			relay := agg.twins[*f.TwinID].account_id
			f.Relay = &relay
		} else {
			c := agg.relays[rand.Intn(len(agg.relays))]
			f.Relay = &c
		}
	}
	if flip(.2) {
		if f.TwinID != nil && flip(.4) {
			publicKey := agg.twins[*f.TwinID].account_id
			f.PublicKey = &publicKey
		} else {
			c := agg.publicKeys[rand.Intn(len(agg.publicKeys))]
			f.PublicKey = &c
		}
	}

	return f
}

func calcTwinsAggregates(data *DBData) (res TwinsAggregate) {
	for _, twin := range data.twins {
		res.twinIDs = append(res.twinIDs, twin.twin_id)
		res.accountIDs = append(res.accountIDs, twin.account_id)
		res.relays = append(res.relays, twin.relay)
		res.publicKeys = append(res.publicKeys, twin.public_key)
	}
	res.twins = data.twins
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
