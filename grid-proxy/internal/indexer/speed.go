package indexer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"

	zosPerfPkg "github.com/threefoldtech/zos/pkg/perf"
	zosIPerfPkg "github.com/threefoldtech/zos/pkg/perf/iperf"
)

const (
	perfTestCallCmd = "zos.perf.get"
	testName        = "iperf"
)

type SpeedWork struct {
	findersInterval map[string]time.Duration
}

func NewSpeedWork(interval uint) *SpeedWork {
	return &SpeedWork{
		findersInterval: map[string]time.Duration{
			"up": time.Duration(interval) * time.Minute,
		},
	}
}

func (w *SpeedWork) Finders() map[string]time.Duration {
	return w.findersInterval
}

func (w *SpeedWork) Get(ctx context.Context, rmb *peer.RpcClient, twinId uint32) ([]types.Speed, error) {
	payload := struct {
		Name string
	}{
		Name: testName,
	}
	var response zosPerfPkg.TaskResult
	if err := callNode(ctx, rmb, perfTestCallCmd, payload, twinId, &response); err != nil {
		return []types.Speed{}, err
	}

	speedReport, err := parseSpeed(response, twinId)
	if err != nil {
		return []types.Speed{}, err
	}

	return []types.Speed{speedReport}, nil
}

func (w *SpeedWork) Upsert(ctx context.Context, db db.Database, batch []types.Speed) error {
	return db.UpsertNetworkSpeed(ctx, batch)
}

func parseSpeed(res zosPerfPkg.TaskResult, twinId uint32) (types.Speed, error) {
	speed := types.Speed{
		NodeTwinId: twinId,
	}

	iperfResultBytes, err := json.Marshal(res.Result)
	if err != nil {
		return speed, err
	}

	var iperfResults []zosIPerfPkg.IperfResult
	if err := json.Unmarshal(iperfResultBytes, &iperfResults); err != nil {
		return speed, err
	}

	// TODO: better parsing
	// we have four speeds tcp/udp for ipv4/ipv6.
	// now, we just pick the first non-zero
	for _, report := range iperfResults {
		if report.DownloadSpeed != 0 {
			speed.Download = report.DownloadSpeed
			speed.Upload = report.UploadSpeed
			return speed, nil
		}
	}

	return speed, nil
}
