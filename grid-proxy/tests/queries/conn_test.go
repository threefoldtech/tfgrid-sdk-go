package test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

func TestDBManyOpenConnections(t *testing.T) {

	gotQueriesCnt := atomic.Int32{}
	wg := sync.WaitGroup{}
	wantQueriesCnt := 5000
	wg.Add(wantQueriesCnt)
	for i := 0; i < wantQueriesCnt; i++ {
		go func() {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
			defer cancel()

			_, _, err := DBClient.GetTwins(ctx, types.TwinFilter{}, types.Limit{Size: 100})
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
