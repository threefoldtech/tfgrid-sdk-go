package test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

func TestManyOpenConnections(t *testing.T) {
	gotQueriesCnt := atomic.Int32{}
	wg := sync.WaitGroup{}
	wantQueriesCnt := 5000
	for i := 0; i < wantQueriesCnt; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, _, err := gridProxyClient.Twins(types.TwinFilter{}, types.Limit{Size: 100})
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
