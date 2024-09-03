package indexer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

const (
	perfTestCallCmd = "zos.perf.get"
	testName        = "iperf"
)

type SpeedWork struct {
	findersInterval map[string]time.Duration
}

type TaskResult struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Timestamp   uint64      `json:"timestamp"`
	Result      interface{} `json:"result"`
}

type IperfResult struct {
	UploadSpeed   float64               `json:"upload_speed"`   // in bit/sec
	DownloadSpeed float64               `json:"download_speed"` // in bit/sec
	NodeID        uint32                `json:"node_id"`
	NodeIpv4      string                `json:"node_ip"`
	TestType      string                `json:"test_type"`
	Error         string                `json:"error"`
	CpuReport     CPUUtilizationPercent `json:"cpu_report"`
}

type CPUUtilizationPercent struct {
	HostTotal    float64 `json:"host_total"`
	HostUser     float64 `json:"host_user"`
	HostSystem   float64 `json:"host_system"`
	RemoteTotal  float64 `json:"remote_total"`
	RemoteUser   float64 `json:"remote_user"`
	RemoteSystem float64 `json:"remote_system"`
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
	var response TaskResult
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

func parseSpeed(res TaskResult, twinId uint32) (types.Speed, error) {
	speed := types.Speed{
		NodeTwinId: twinId,
	}

	iperfResultBytes, err := json.Marshal(res.Result)
	if err != nil {
		return speed, err
	}

	var iperfResults []IperfResult
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

	speed.UpdatedAt = time.Now().Unix()

	return speed, nil
}
