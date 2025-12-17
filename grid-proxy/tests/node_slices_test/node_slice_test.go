package nodeslicestest

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"testing"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	vegeta "github.com/tsenart/vegeta/lib"
)

func TestNodeSlices(t *testing.T) {
	loadTest := &LoadTest{
		Endpoints: []string{"/nodes_test"},
		Freq:      10,
		Duration:  5,
		BaseUrl:   "http://localhost:8080",
		// Example fuzzing ranges; adjust as needed
		MinSRU: 1 * 1024 * 1024 * 1024, MaxSRU: 64 * 1024 * 1024 * 1024, // 1–64 GiB
		MinHRU: 10 * 1024 * 1024 * 1024, MaxHRU: 500 * 1024 * 1024 * 1024, // 10–500 GiB
		MinMRU: 1 * 1024 * 1024 * 1024, MaxMRU: 64 * 1024 * 1024 * 1024, // 1–64 GiB
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	attacker := vegeta.NewAttacker()

	rate := vegeta.Rate{Freq: loadTest.Freq, Per: time.Second}
	duration := time.Duration(loadTest.Duration) * time.Second

	// Custom targeter that fuzzes the filter values for each request
	targeter := func(tgt *vegeta.Target) error {
		endpoint := loadTest.Endpoints[rng.Intn(len(loadTest.Endpoints))]

		u, err := url.Parse(loadTest.BaseUrl + endpoint)
		if err != nil {
			return err
		}

		q := u.Query()
		// Helper to pick a random int in [min, max] using the local RNG
		randInRange := func(min, max int) int {
			if max <= min {
				return min
			}
			return rng.Intn(max-min+1) + min
		}

		sru := randInRange(loadTest.MinSRU, loadTest.MaxSRU)
		hru := randInRange(loadTest.MinHRU, loadTest.MaxHRU)
		mru := randInRange(loadTest.MinMRU, loadTest.MaxMRU)

		q.Set("free_sru", fmt.Sprintf("%d", sru))
		q.Set("free_hru", fmt.Sprintf("%d", hru))
		q.Set("free_mru", fmt.Sprintf("%d", mru))

		u.RawQuery = q.Encode()

		tgt.Method = "GET"
		tgt.URL = u.String()
		return nil
	}

	var metrics vegeta.Metrics
	reqIdx := 0
	for res := range attacker.Attack(targeter, rate, duration, "load testing") {
		reqIdx++
		var nodes []types.NodeWithSlicesResult
		nodeCount := 0
		if res.Code == 200 {
			if err := json.Unmarshal(res.Body, &nodes); err == nil {
				nodeCount = len(nodes)
			}
		}

		if res.Error != "" || res.Code >= 400 {
			t.Logf("Req #%d | Code=%d | Latency=%s | Error=%s",
				reqIdx, res.Code, res.Latency, res.Error)
		} else {
			t.Logf("Req #%d | Code=%d | Latency=%s | Nodes=%d",
				reqIdx, res.Code, res.Latency, nodeCount)
		}

		metrics.Add(res)
	}
	metrics.Close()

	t.Logf("=== Summary ===")
	t.Logf("Total requests: %d", metrics.Requests)
	t.Logf("Total errors: %d", len(metrics.Errors))
	t.Logf("Total duration: %s", metrics.Duration)
	t.Logf("Overall rate: %.2f req/s", metrics.Rate)
}

type LoadTest struct {
	Endpoints []string
	Freq      int
	Duration  int
	BaseUrl   string
	MinSRU    int
	MaxSRU    int
	MinHRU    int
	MaxHRU    int
	MinMRU    int
	MaxMRU    int
}
