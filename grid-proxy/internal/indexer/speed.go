package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
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

func updateWithFirstNonZero(current, newValue float64) float64 {
	if current == 0 {
		return newValue
	}
	return current
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

	speed.UpdatedAt = time.Now().Unix()
	for _, report := range iperfResults {
		ip := net.ParseIP(report.NodeIpv4)
		if ip == nil {
			return speed, fmt.Errorf("invalid IP address: %s", report.NodeIpv4)
		}
		isIpv4 := ip.To4() != nil
		isIpv6 := ip.To4() == nil && ip.To16() != nil
		if report.TestType == "tcp" && isIpv4 {
			speed.Upload = updateWithFirstNonZero(speed.Upload, report.UploadSpeed)
			speed.Download = updateWithFirstNonZero(speed.Download, report.DownloadSpeed)
		} else if report.TestType == "udp" && isIpv4 {
			speed.UDPUploadIPv4 = updateWithFirstNonZero(speed.Upload, report.UploadSpeed)
			speed.UDPDownloadIPv4 = updateWithFirstNonZero(speed.Download, report.DownloadSpeed)
		} else if report.TestType == "tcp" && isIpv6 {
			speed.TCPUploadIPv6 = updateWithFirstNonZero(speed.Upload, report.UploadSpeed)
			speed.TCPDownloadIPv6 = updateWithFirstNonZero(speed.Download, report.DownloadSpeed)
		} else if report.TestType == "udp" && isIpv6 {
			speed.UDPUploadIPv6 = updateWithFirstNonZero(speed.Upload, report.UploadSpeed)
			speed.UDPDownloadIPv6 = updateWithFirstNonZero(speed.Download, report.DownloadSpeed)
		}
		if speed.Upload != 0 && speed.Download != 0 &&
			speed.UDPUploadIPv4 != 0 && speed.UDPDownloadIPv4 != 0 &&
			speed.TCPUploadIPv6 != 0 && speed.TCPDownloadIPv6 != 0 &&
			speed.UDPUploadIPv6 != 0 && speed.UDPDownloadIPv6 != 0 {
			return speed, nil
		}
	}

	return speed, nil
}
