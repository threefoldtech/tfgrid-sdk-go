package mw

import "net/http"

// WithRateLimit wraps an Action with rate limiting if a rate limiter is provided.
// If rateLimiter is nil, it falls back to the standard AsHandlerFunc wrapper.
// This provides a clean way to conditionally apply rate limiting to endpoints.
func WithRateLimit(rateLimiter *RateLimiterMiddleware, action Action) http.HandlerFunc {
	if rateLimiter != nil {
		return rateLimiter.AsRateLimitedHandlerFunc(action)
	}
	return AsHandlerFunc(action)
}

// WithRateLimitProxy wraps a ProxyAction with rate limiting if a rate limiter is provided.
// If rateLimiter is nil, it falls back to the standard AsProxyHandlerFunc wrapper.
// This provides a clean way to conditionally apply rate limiting to proxy endpoints.
func WithRateLimitProxy(rateLimiter *RateLimiterMiddleware, action ProxyAction) http.HandlerFunc {
	if rateLimiter != nil {
		return rateLimiter.AsRateLimitedProxyHandlerFunc(action)
	}
	return AsProxyHandlerFunc(action)
}
