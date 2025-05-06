package indexer

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
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

func isValidIpv4(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}

	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			return false
		}

		if num < 0 || num > 255 {
			return false
		}
	}
	return true
}

func isValidIpv6(ip string) bool {
	if strings.Contains(ip, "::") {
		if strings.Count(ip, "::") > 1 {
			return false
		}

		parts := strings.Split(ip, "::")
		if len(parts) > 2 {
			return false
		}

		// Check the parts before and after ::
		if len(parts[0]) > 0 {
			beforeParts := strings.Split(parts[0], ":")
			for _, part := range beforeParts {
				if !isValidIpv6Hextet(part) {
					return false
				}
			}
		}

		if len(parts) > 1 && len(parts[1]) > 0 {
			afterParts := strings.Split(parts[1], ":")
			for _, part := range afterParts {
				if !isValidIpv6Hextet(part) {
					return false
				}
			}
		}

		return true
	}

	// Handle regular (uncompressed) IPv6
	parts := strings.Split(ip, ":")
	if len(parts) != 8 {
		return false
	}

	for _, part := range parts {
		if !isValidIpv6Hextet(part) {
			return false
		}
	}

	return true
}

// Helper function to validate an IPv6 hexadecimal segment
func isValidIpv6Hextet(hextet string) bool {
	// Each IPv6 segment must be a valid hexadecimal value between 0 and FFFF
	if len(hextet) == 0 || len(hextet) > 4 {
		return false
	}

	for _, c := range hextet {
		isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
		if !isHex {
			return false
		}
	}

	return true
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

	// Parse the results into the appropriate fields based on TestType and IpVersion
	for _, report := range iperfResults {
		isIpv4 := isValidIpv4(report.NodeIpv4)
		isIpv6 := isValidIpv6(report.NodeIpv4)
		if report.TestType == "tcp" && isIpv4 {
			speed.Upload = report.UploadSpeed
			speed.Download = report.DownloadSpeed
		} else if report.TestType == "udp" && isIpv4 {
			speed.UDPUploadIPv4 = report.UploadSpeed
			speed.UDPDownloadIPv4 = report.DownloadSpeed
		} else if report.TestType == "tcp" && isIpv6 {
			speed.TCPUploadIPv6 = report.UploadSpeed
			speed.TCPDownloadIPv6 = report.DownloadSpeed
		} else if report.TestType == "udp" && isIpv6 {
			speed.UDPUploadIPv6 = report.UploadSpeed
			speed.UDPDownloadIPv6 = report.DownloadSpeed
		}
	}

	// For backward compatibility, if no TCP/IPv4 values were found but others were,
	// set default Upload/Download to the first valid result
	if speed.Upload == 0 && speed.Download == 0 {
		for _, report := range iperfResults {
			if report.DownloadSpeed != 0 || report.UploadSpeed != 0 {
				speed.Upload = report.UploadSpeed
				speed.Download = report.DownloadSpeed
				break
			}
		}
	}

	speed.UpdatedAt = time.Now().Unix()

	return speed, nil
}
