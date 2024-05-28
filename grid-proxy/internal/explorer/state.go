package explorer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
)

const (
	OkState = "ok"
)

func createReport(db DBClient, peer rmb.Client, idxIntervals map[string]uint) types.Healthiness {
	var report types.Healthiness

	// db connection
	report.DBConn = OkState
	if err := db.DB.Ping(); err != nil {
		report.DBConn = err.Error()
	}
	if err := db.DB.Initialized(); err != nil {
		report.DBConn = err.Error()
	}

	// rmb connection
	report.RMBConn = OkState
	if err := pingRandomTwins(db, peer); err != nil {
		report.RMBConn = err.Error()
	}

	// indexers
	indexers, err := db.DB.GetLastUpsertsTimestamp()
	if err != nil {
		log.Error().Err(err).Msg("failed to get last upsert timestamp")
	}
	report.Indexers = indexers

	// total
	report.TotalStateOk = true
	if report.DBConn != OkState ||
		report.RMBConn != OkState {
		report.TotalStateOk = false
	}

	if isIndexerStale(indexers.Dmi.UpdatedAt, idxIntervals["dmi"]) ||
		isIndexerStale(indexers.Gpu.UpdatedAt, idxIntervals["gpu"]) ||
		isIndexerStale(indexers.Health.UpdatedAt, idxIntervals["health"]) ||
		isIndexerStale(indexers.Ipv6.UpdatedAt, idxIntervals["ipv6"]) ||
		isIndexerStale(indexers.Speed.UpdatedAt, idxIntervals["speed"]) ||
		isIndexerStale(indexers.Workloads.UpdatedAt, idxIntervals["workloads"]) {
		report.TotalStateOk = false
	}

	return report
}

func isIndexerStale(updatedAt int64, interval uint) bool {
	updatedAtInTime := time.Unix(updatedAt, 0)
	return time.Now().Sub(updatedAtInTime) > time.Duration(interval)*time.Minute
}

func pingRandomTwins(db DBClient, peer rmb.Client) error {
	twinIds, err := db.DB.GetRandomHealthyTwinIds(10)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	successCh := make(chan bool)

	for _, twinId := range twinIds {
		wg.Add(1)
		go func(twinId uint32) {
			defer wg.Done()

			callCtx, callCancel := context.WithTimeout(ctx, 10*time.Second)
			defer callCancel()

			var res interface{}
			if err := peer.Call(callCtx, twinId, "zos.system.version", nil, &res); err == nil {
				select {
				case successCh <- true:
				case <-ctx.Done():
				}
			}
		}(twinId)
	}

	go func() {
		wg.Wait()
		close(successCh)
	}()

	select {
	case <-successCh:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("failed to call twins: %+v", twinIds)
	}
}
