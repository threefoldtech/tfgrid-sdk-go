package ratelimiter

import (
	"net/http"
	"testing"
	"time"
)

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name: "X-Forwarded-For header",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1, 10.0.0.1",
			},
			remoteAddr: "127.0.0.1:8080",
			expected:   "192.168.1.1",
		},
		{
			name: "X-Real-IP header",
			headers: map[string]string{
				"X-Real-IP": "203.0.113.1",
			},
			remoteAddr: "127.0.0.1:8080",
			expected:   "203.0.113.1",
		},
		{
			name:       "RemoteAddr fallback",
			headers:    map[string]string{},
			remoteAddr: "198.51.100.1:8080",
			expected:   "198.51.100.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Header:     make(http.Header),
				RemoteAddr: tt.remoteAddr,
			}

			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			result := GetClientIP(req)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestSlidingWindowRateLimiter(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(2)

	if !limiter.Allow("192.168.1.1") {
		t.Error("First request should be allowed")
	}
	if !limiter.Allow("192.168.1.1") {
		t.Error("Second request should be allowed")
	}

	if limiter.Allow("192.168.1.1") {
		t.Error("Third request should be blocked")
	}

	// Different IP should be allowed
	if !limiter.Allow("192.168.1.2") {
		t.Error("Request from different IP should be allowed")
	}

	// After waiting, requests should be allowed again
	time.Sleep(1100 * time.Millisecond)
	if !limiter.Allow("192.168.1.1") {
		t.Error("Request should be allowed after window slide")
	}
}

func TestSlidingWindowRateLimiterStats(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(5)

	// Make some requests
	limiter.Allow("192.168.1.1")
	limiter.Allow("192.168.1.1")
	limiter.Allow("192.168.1.2")

	stats := limiter.GetStats()

	if stats["rate_limit_per_second"] != 5 {
		t.Errorf("Expected rate limit 5, got %v", stats["rate_limit_per_second"])
	}

	if stats["total_tracked_ips"] != 2 {
		t.Errorf("Expected 2 tracked IPs, got %v", stats["total_tracked_ips"])
	}
}

func TestGetCurrentRequestCountAndRateLimit(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(5)

	// Test rate limit getter
	if limiter.GetRateLimit() != 5 {
		t.Errorf("Expected rate limit 5, got %d", limiter.GetRateLimit())
	}

	// Test initial request count
	if count := limiter.GetCurrentRequestCount("192.168.1.1"); count != 0 {
		t.Errorf("Expected 0 requests initially, got %d", count)
	}

	// Make some requests
	limiter.Allow("192.168.1.1")
	limiter.Allow("192.168.1.1")
	limiter.Allow("192.168.1.1")

	// Test current request count
	if count := limiter.GetCurrentRequestCount("192.168.1.1"); count != 3 {
		t.Errorf("Expected 3 requests, got %d", count)
	}

	// Different IP should have 0
	if count := limiter.GetCurrentRequestCount("192.168.1.2"); count != 0 {
		t.Errorf("Expected 0 requests for different IP, got %d", count)
	}
}
