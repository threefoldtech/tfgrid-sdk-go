package test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	db "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

func TestDBManyOpenConnections(t *testing.T) {
	p, err := db.NewPostgresDatabase(POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSSWORD, POSTGRES_DB, 80)
	require.NoError(t, err)

	gotQueriesCnt := atomic.Int32{}
	wg := sync.WaitGroup{}
	wantQueriesCnt := 5000
	wg.Add(wantQueriesCnt)
	for i := 0; i < wantQueriesCnt; i++ {
		go func() {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
			defer cancel()

			_, _, err := p.GetTwins(ctx, types.TwinFilter{}, types.Limit{Size: 100})
			if err != nil {
				log.Err(err).Msg("twin query failed")
				return
			}

			gotQueriesCnt.Add(1)
		}()
	}

	wg.Wait()

	assert.Equal(t, wantQueriesCnt, int(gotQueriesCnt.Load()), "some queries failed")
}
