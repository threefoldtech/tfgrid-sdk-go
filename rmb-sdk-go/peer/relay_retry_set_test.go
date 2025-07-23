package peer

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type dummyRelay struct {
	id   int
	fail atomic.Bool
}

func (d *dummyRelay) send(ctx context.Context, data []byte) error {
	if d.fail.Load() {
		return context.DeadlineExceeded
	}
	return nil
}

func TestCooldownRelaySet_FairFailoverAndCooldown(t *testing.T) {
	relay1 := &dummyRelay{id: 1}
	relay2 := &dummyRelay{id: 2}
	relay3 := &dummyRelay{id: 3}

	set := &CooldownRelaySet[*dummyRelay]{
		Relays: []RelayPenalty[*dummyRelay]{
			{Relay: relay1},
			{Relay: relay2},
			{Relay: relay3},
		},
		Cooldown: 200 * time.Millisecond,
	}

	// Initially, all relays are healthy
	items := set.Sorted(time.Now())
	require.Len(t, items, 3)

	// Mark relay1 as failed
	set.MarkFailure(relay1, time.Now())
	items = set.Sorted(time.Now())
	// relay1 should be deprioritized
	require.Equal(t, relay1.id, items[2].Relay.id)
	ids := []int{items[0].Relay.id, items[1].Relay.id}
	require.Contains(t, ids, 2)
	require.Contains(t, ids, 3)

	// After cooldown expires, relay1 should be healthy again
	time.Sleep(210 * time.Millisecond)
	items = set.Sorted(time.Now())
	// All relays should be shuffled fairly
	ids = []int{items[0].Relay.id, items[1].Relay.id, items[2].Relay.id}
	require.Contains(t, ids, 1)
	require.Contains(t, ids, 2)
	require.Contains(t, ids, 3)
}

func TestCooldownRelaySet_FailingRelaysOrderedByErrorTime(t *testing.T) {
	relayA := &dummyRelay{id: 1}
	relayB := &dummyRelay{id: 2}
	relayC := &dummyRelay{id: 3}
	set := &CooldownRelaySet[*dummyRelay]{
		Relays: []RelayPenalty[*dummyRelay]{
			{Relay: relayA},
			{Relay: relayB},
			{Relay: relayC},
		},
		Cooldown: 10 * time.Second,
	}

	now := time.Now()
	set.MarkFailure(relayB, now.Add(-3*time.Second)) // oldest error
	set.MarkFailure(relayC, now.Add(-2*time.Second)) // middle error
	set.MarkFailure(relayA, now.Add(-1*time.Second)) // newest error
	items := set.Sorted(now)

	// Only error time matters for ordering, so we expect relayB, relayC, relayA (oldest to newest)
	ids := []int{items[0].Relay.id, items[1].Relay.id, items[2].Relay.id}
	require.Equal(t, []int{2, 3, 1}, ids, "Relays should be ordered by error time (oldest first)")
}

func TestCooldownRelaySet_ThreadSafePenalty(t *testing.T) {
	relay := &dummyRelay{id: 1}
	set := &CooldownRelaySet[*dummyRelay]{
		Relays: []RelayPenalty[*dummyRelay]{
			{Relay: relay},
		},
		Cooldown: 50 * time.Millisecond,
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			set.MarkFailure(relay, time.Now())
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			set.MarkSuccess(relay)
		}
	}()
	wg.Wait()

	// Should not panic or race
}
