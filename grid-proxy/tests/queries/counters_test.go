package main

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	proxyclient "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
	proxytypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

func TestCounters(t *testing.T) {
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

	t.Run("counters up test", func(t *testing.T) {
		f := proxytypes.StatsFilter{
			Status: &STATUS_UP,
		}
		counters, err := localClient.Counters(f)
		assert.NoError(t, err)
		remote, err := proxyClient.Counters(f)
		assert.NoError(t, err)
		require.True(t, reflect.DeepEqual(counters, remote), cmp.Diff(counters, remote))

	})

	t.Run("counters all test", func(t *testing.T) {
		f := proxytypes.StatsFilter{}
		counters, err := localClient.Counters(f)
		assert.NoError(t, err)
		remote, err := proxyClient.Counters(f)
		assert.NoError(t, err)
		require.True(t, reflect.DeepEqual(counters, remote), cmp.Diff(counters, remote))

	})
}

// func validateCountersResults(local, remote proxytypes.Counters) error {
// 	if !reflect.DeepEqual(local, remote) {
// 		return fmt.Errorf("counters mismatch: local: %+v, remote: %+v", local, remote)
// 	}
// 	return nil
// }
