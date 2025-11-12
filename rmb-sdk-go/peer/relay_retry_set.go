package peer

import (
	"math/rand"
	"sort"
	"sync/atomic"
	"time"
)

// RelayPenalty tracks relay connection and its penalty (last error timestamp).
// Relay should be a pointer type for atomic safety.
type RelayPenalty struct {
	Relay       *InnerConnection
	LastErrorAt int64 // UnixNano timestamp of last error, 0 means healthy (must be accessed atomically)
}

// CooldownRelaySet manages a set of relays with cooldown-based penalty logic.
//
// This structure is used as the main relay manager in Peer for fair failover and retry.
// It is thread-safe for penalty updates using atomic operations, but not for concurrent mutation of the relay set itself.
type CooldownRelaySet struct {
	Relays   []RelayPenalty // slice of relay+penalty state, must not be mutated concurrently
	Cooldown time.Duration  // cooldown period for penalized relays
}

// Sorted returns relays sorted by penalty (lowest/oldest error first), shuffling among equals.
func (s *CooldownRelaySet) Sorted(now time.Time) []RelayPenalty {
	type sortableRelay struct {
		RelayPenalty
		effectiveError int64
	}

	items := make([]sortableRelay, len(s.Relays))
	for i := range s.Relays {
		// Must atomically load LastErrorAt to avoid racing with MarkSuccess/MarkFailure
		lastErr := atomic.LoadInt64(&s.Relays[i].LastErrorAt)
		r := RelayPenalty{
			Relay:       s.Relays[i].Relay,
			LastErrorAt: lastErr,
		}
		items[i] = sortableRelay{
			RelayPenalty:   r,
			effectiveError: s.effectiveError(r, now),
		}
	}

	// Sort by effective error time (ascending). Stable sort preserves original
	// order for relays with the same penalty, which is fair.
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].effectiveError < items[j].effectiveError
	})

	// Shuffle among equals
	start := 0
	for start < len(items) {
		end := start + 1
		for end < len(items) && items[end].effectiveError == items[start].effectiveError {
			end++
		}
		rand.Shuffle(end-start, func(i, j int) {
			items[start+i], items[start+j] = items[start+j], items[start+i]
		})
		start = end
	}

	// Unwrap the sorted relays
	result := make([]RelayPenalty, len(items))
	for i, item := range items {
		result[i] = item.RelayPenalty
	}

	return result
}

// MarkFailure updates the penalty for the given relay.
func (s *CooldownRelaySet) MarkFailure(relay *InnerConnection, now time.Time) {
	for i := range s.Relays {
		if s.Relays[i].Relay == relay {
			atomic.StoreInt64(&s.Relays[i].LastErrorAt, now.UnixNano())
			return
		}
	}
}

// MarkSuccess resets the penalty for the given relay.
func (s *CooldownRelaySet) MarkSuccess(relay *InnerConnection) {
	for i := range s.Relays {
		if s.Relays[i].Relay == relay {
			atomic.StoreInt64(&s.Relays[i].LastErrorAt, 0)
			return
		}
	}
}

// effectiveError returns zero if cooldown expired, otherwise returns LastErrorAt.
func (s *CooldownRelaySet) effectiveError(r RelayPenalty, now time.Time) int64 {
	lastErr := atomic.LoadInt64(&r.LastErrorAt)
	if lastErr == 0 {
		return 0
	}
	if now.Sub(time.Unix(0, lastErr)) > s.Cooldown {
		return 0
	}
	return lastErr
}
