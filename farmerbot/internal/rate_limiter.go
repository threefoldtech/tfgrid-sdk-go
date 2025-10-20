package internal

import (
	"sync"
	"time"
)

// RateLimiter implements a simple rate limiter
type RateLimiter struct {
	mu                        sync.Mutex
	lastCallTime              time.Time
	callsIntervalBetweenCalls time.Duration
}

// NewRateLimiter creates a new rate limiter with the specified interval between calls
func NewRateLimiter(interval time.Duration) *RateLimiter {
	return &RateLimiter{
		lastCallTime:              time.Time{}, // Zero time
		callsIntervalBetweenCalls: interval,
	}
}

func (r *RateLimiter) Allowed() (bool, time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	if r.lastCallTime.IsZero() {
		r.lastCallTime = now
		return true, 0
	}

	elapsed := now.Sub(r.lastCallTime)
	if elapsed < r.callsIntervalBetweenCalls {
		return false, r.callsIntervalBetweenCalls - elapsed
	}

	r.lastCallTime = now
	return true, 0
}

// getRateLimitDuration returns the rate limit duration based on the configured value
func getRateLimitDuration(configuredSeconds int) time.Duration {
	if configuredSeconds <= 0 {
		// Default to a second if not configured or invalid
		return time.Second
	}
	return time.Duration(configuredSeconds) * time.Second
}
