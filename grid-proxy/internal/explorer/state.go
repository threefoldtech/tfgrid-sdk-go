package explorer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
)

const (
	OkState = "ok"
)

func createReport(db DBClient, peer rmb.Client) types.Healthiness {
	var report types.Healthiness

	// db connection
	report.DBConn = OkState
	err := db.DB.Ping()
	if err != nil {
		report.DBConn = err.Error()
	}
	err = db.DB.Initialized()
	if err != nil {
		report.DBConn = err.Error()
	}

	// rmb connection
	report.RMBConn = OkState
	if err := pingRandomTwins(db, peer); err != nil {
		report.RMBConn = err.Error()
	}

	// total
	report.TotalStateOk = true
	if report.DBConn != OkState ||
		report.RMBConn != OkState {
		report.TotalStateOk = false
	}
	return report
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
