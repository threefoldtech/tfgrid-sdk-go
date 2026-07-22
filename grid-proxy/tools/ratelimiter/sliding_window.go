package ratelimiter

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// SlidingWindow represents a sliding window rate limiter for a single IP
type SlidingWindow struct {
	requests    []time.Time
	lastCleanup time.Time
	mu          sync.RWMutex
}

// SlidingWindowRateLimiter implements IP-based rate limiting using sliding window algorithm
type SlidingWindowRateLimiter struct {
	rate              int
	windowSize        time.Duration
	clients           map[string]*SlidingWindow
	mu                sync.RWMutex
	cleanupInterval   time.Duration
	lastGlobalCleanup time.Time
}

// NewSlidingWindowRateLimiter creates a new sliding window rate limiter
func NewSlidingWindowRateLimiter(ratePerSecond int) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		rate:              ratePerSecond,
		windowSize:        time.Second,
		clients:           make(map[string]*SlidingWindow),
		cleanupInterval:   time.Minute * 5,
		lastGlobalCleanup: time.Now(),
	}
}

// Allow checks if a request from the given IP should be allowed
func (rl *SlidingWindowRateLimiter) Allow(ip string) bool {
	now := time.Now()
	rl.performGlobalCleanupIfNeeded(now)
	window := rl.getOrCreateWindow(ip)

	window.mu.Lock()
	defer window.mu.Unlock()

	cutoff := now.Add(-rl.windowSize)
	window.requests = rl.filterRequests(window.requests, cutoff)
	window.lastCleanup = now

	if len(window.requests) >= rl.rate {
		log.Debug().
			Str("ip", ip).
			Int("current_requests", len(window.requests)).
			Int("rate_limit", rl.rate).
			Msg("Rate limit exceeded")
		return false
	}

	window.requests = append(window.requests, now)
	log.Debug().
		Str("ip", ip).
		Int("current_requests", len(window.requests)).
		Int("rate_limit", rl.rate).
		Msg("Request allowed")

	return true
}

// getOrCreateWindow retrieves or creates a sliding window for an IP address
func (rl *SlidingWindowRateLimiter) getOrCreateWindow(ip string) *SlidingWindow {
	rl.mu.RLock()
	window, exists := rl.clients[ip]
	rl.mu.RUnlock()
	if exists {
		return window
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	if window, exists := rl.clients[ip]; exists {
		return window
	}

	window = &SlidingWindow{
		requests:    make([]time.Time, 0, rl.rate),
		lastCleanup: time.Now(),
	}
	rl.clients[ip] = window

	log.Debug().
		Str("ip", ip).
		Msg("Created new sliding window for IP")

	return window
}

// filterRequests removes requests older than the cutoff time
func (rl *SlidingWindowRateLimiter) filterRequests(requests []time.Time, cutoff time.Time) []time.Time {
	validIdx := 0
	for i, req := range requests {
		if req.After(cutoff) {
			validIdx = i
			break
		}
		validIdx = len(requests)
	}

	if validIdx >= len(requests) {
		return requests[:0]
	}

	return requests[validIdx:]
}

// performGlobalCleanupIfNeeded removes old IP entries that haven't been used recently
func (rl *SlidingWindowRateLimiter) performGlobalCleanupIfNeeded(now time.Time) {
	if now.Sub(rl.lastGlobalCleanup) < rl.cleanupInterval {
		return
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := now.Add(-rl.cleanupInterval)
	toDelete := make([]string, 0)

	for ip, window := range rl.clients {
		window.mu.RLock()
		shouldDelete := window.lastCleanup.Before(cutoff) && len(window.requests) == 0
		window.mu.RUnlock()

		if shouldDelete {
			toDelete = append(toDelete, ip)
		}
	}

	for _, ip := range toDelete {
		delete(rl.clients, ip)
	}

	rl.lastGlobalCleanup = now

	if len(toDelete) > 0 {
		log.Debug().
			Int("cleaned_ips", len(toDelete)).
			Int("remaining_ips", len(rl.clients)).
			Msg("Performed global cleanup of rate limiter")
	}
}

// GetStats returns current statistics about the rate limiter
func (rl *SlidingWindowRateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	totalRequests := 0
	activeIPs := 0

	for _, window := range rl.clients {
		window.mu.RLock()
		totalRequests += len(window.requests)
		if len(window.requests) > 0 {
			activeIPs++
		}
		window.mu.RUnlock()
	}

	return map[string]interface{}{
		"rate_limit_per_second": rl.rate,
		"total_tracked_ips":     len(rl.clients),
		"active_ips":            activeIPs,
		"total_active_requests": totalRequests,
		"window_size_seconds":   rl.windowSize.Seconds(),
	}
}

// String returns a string representation of the rate limiter
func (rl *SlidingWindowRateLimiter) String() string {
	return fmt.Sprintf("SlidingWindowRateLimiter(rate=%d/sec, window=%v)", rl.rate, rl.windowSize)
}

// GetCurrentRequestCount returns the current number of requests for a specific IP
func (rl *SlidingWindowRateLimiter) GetCurrentRequestCount(ip string) int {
	rl.mu.RLock()
	window, exists := rl.clients[ip]
	rl.mu.RUnlock()

	if !exists {
		return 0
	}

	window.mu.RLock()
	defer window.mu.RUnlock()

	now := time.Now()
	cutoff := now.Add(-rl.windowSize)

	count := 0
	for _, req := range window.requests {
		if req.After(cutoff) {
			count++
		}
	}

	return count
}

// GetRateLimit returns the configured rate limit
func (rl *SlidingWindowRateLimiter) GetRateLimit() int {
	return rl.rate
}
