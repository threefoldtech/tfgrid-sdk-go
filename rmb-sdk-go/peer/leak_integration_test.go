//go:build goleak
// +build goleak

package peer

import (
	"context"
	"os"
	"testing"
	"time"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer/types"
	"go.uber.org/goleak"
)

// TestMain runs goleak after all tests to detect leaked goroutines.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// TestPeerShutdownNoLeak is an integration test that requires real endpoints.
// It will be skipped unless the following env vars are provided:
// - RMB_RELAY_URL   (e.g. wss://relay.grid.tf or ngrok relay)
// - TFCHAIN_WS_URL  (e.g. wss://tfchain.dev.grid.tf/ws)
// - RMB_MNEMONICS   (development mnemonics string)
// run with RMB_RELAY_URL="" TFCHAIN_WS_URL="" RMB_MNEMONICS="" go test ./rmb-sdk-go/peer -tags goleak -race -run TestPeerShutdownNoLeak -v
func TestPeerShutdownNoLeak(t *testing.T) {
	relayURL := os.Getenv("RMB_RELAY_URL")
	substrateURL := os.Getenv("TFCHAIN_WS_URL")
	mnemonics := os.Getenv("RMB_MNEMONICS")
	if relayURL == "" || substrateURL == "" || mnemonics == "" {
		t.Skip("integration test skipped: set RMB_RELAY_URL, TFCHAIN_WS_URL, RMB_MNEMONICS to run")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Build substrate manager with the provided TFChain URL
	mgr := substrate.NewManager(substrateURL)

	h := func(ctx context.Context, p *Peer, env *types.Envelope, err error) {
		// no-op handler for test
	}

	peer, err := NewPeer(ctx, mnemonics, mgr, h, WithRelay(relayURL))
	if err != nil {
		t.Fatalf("NewPeer failed: %v", err)
	}

	time.Sleep(5 * time.Second)

	cancel()
	peer.Wait()
}
