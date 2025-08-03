package peer

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCooldownRelaySet_FairFailoverAndCooldown(t *testing.T) {
	relay1 := &InnerConnection{twinID: 1}
	relay2 := &InnerConnection{twinID: 2}
	relay3 := &InnerConnection{twinID: 3}

	set := &CooldownRelaySet{
		Relays: []RelayPenalty{
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
	require.Equal(t, relay1.twinID, items[2].Relay.twinID)
	ids := []uint32{items[0].Relay.twinID, items[1].Relay.twinID}
	require.Contains(t, ids, uint32(2))
	require.Contains(t, ids, uint32(3))

	// After cooldown expires, relay1 should be healthy again
	time.Sleep(210 * time.Millisecond)
	items = set.Sorted(time.Now())
	// All relays should be shuffled fairly
	ids = []uint32{items[0].Relay.twinID, items[1].Relay.twinID, items[2].Relay.twinID}
	require.Contains(t, ids, uint32(1))
	require.Contains(t, ids, uint32(2))
	require.Contains(t, ids, uint32(3))
}

func TestCooldownRelaySet_FailingRelaysOrderedByErrorTime(t *testing.T) {
	relayA := &InnerConnection{twinID: 1}
	relayB := &InnerConnection{twinID: 2}
	relayC := &InnerConnection{twinID: 3}
	set := &CooldownRelaySet{
		Relays: []RelayPenalty{
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
	ids := []uint32{items[0].Relay.twinID, items[1].Relay.twinID, items[2].Relay.twinID}
	require.Equal(t, []uint32{2, 3, 1}, ids, "Relays should be ordered by error time (oldest first)")
}

func TestCooldownRelaySet_ThreadSafePenalty(t *testing.T) {
	relay := &InnerConnection{twinID: 1}
	set := &CooldownRelaySet{
		Relays: []RelayPenalty{
			{Relay: relay},
		},
		Cooldown: 50 * time.Millisecond,
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for range 1000 {
			set.MarkFailure(relay, time.Now())
		}
	}()
	go func() {
		defer wg.Done()
		for range 1000 {
			set.MarkSuccess(relay)
		}
	}()
	wg.Wait()

	// Should not panic or race
}
