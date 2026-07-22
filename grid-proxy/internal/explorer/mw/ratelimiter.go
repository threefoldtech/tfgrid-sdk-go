package mw

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/tools/ratelimiter"
)

// RateLimiterMiddleware wraps the rate limiter to work with the existing middleware pattern
type RateLimiterMiddleware struct {
	limiter *ratelimiter.SlidingWindowRateLimiter
}

// NewRateLimiterMiddleware creates a new rate limiter middleware
func NewRateLimiterMiddleware(ratePerSecond int) *RateLimiterMiddleware {
	return &RateLimiterMiddleware{
		limiter: ratelimiter.NewSlidingWindowRateLimiter(ratePerSecond),
	}
}

// RateLimitAction wraps an Action with rate limiting
func (rlm *RateLimiterMiddleware) RateLimitAction(action Action) Action {
	return func(r *http.Request) (interface{}, Response) {
		clientIP := ratelimiter.GetClientIP(r)

		if !rlm.limiter.Allow(clientIP) {
			log.Warn().
				Str("ip", clientIP).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Msg("Rate limit exceeded")

			return nil, rlm.TooManyRequests(fmt.Errorf("rate limit exceeded for IP: %s", clientIP), clientIP)
		}

		return action(r)
	}
}

// RateLimitProxyAction wraps a ProxyAction with rate limiting
func (rlm *RateLimiterMiddleware) RateLimitProxyAction(action ProxyAction) ProxyAction {
	return func(r *http.Request) (*http.Response, Response) {
		clientIP := ratelimiter.GetClientIP(r)
		if !rlm.limiter.Allow(clientIP) {
			log.Warn().
				Str("ip", clientIP).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Msg("Rate limit exceeded")

			return nil, rlm.TooManyRequests(fmt.Errorf("rate limit exceeded for IP: %s", clientIP), clientIP)
		}

		return action(r)
	}
}

// AsRateLimitedHandlerFunc wraps AsHandlerFunc with rate limiting
func (rlm *RateLimiterMiddleware) AsRateLimitedHandlerFunc(action Action) http.HandlerFunc {
	rateLimitedAction := rlm.RateLimitAction(action)
	return AsHandlerFunc(rateLimitedAction)
}

// AsRateLimitedProxyHandlerFunc wraps AsProxyHandlerFunc with rate limiting
func (rlm *RateLimiterMiddleware) AsRateLimitedProxyHandlerFunc(action ProxyAction) http.HandlerFunc {
	rateLimitedAction := rlm.RateLimitProxyAction(action)
	return AsProxyHandlerFunc(rateLimitedAction)
}

// GetStats returns rate limiter statistics
func (rlm *RateLimiterMiddleware) GetStats() map[string]interface{} {
	return rlm.limiter.GetStats()
}

// TooManyRequests returns a 429 Too Many Requests response with accurate rate limit headers
func (rlm *RateLimiterMiddleware) TooManyRequests(err error, clientIP string) Response {
	rateLimit := rlm.limiter.GetRateLimit()
	currentRequests := rlm.limiter.GetCurrentRequestCount(clientIP)
	remaining := max(0, rateLimit-currentRequests)
	resetTime := time.Now().Add(time.Second)

	return Error(err, http.StatusTooManyRequests).
		WithHeader("Retry-After", "1").
		WithHeader("X-RateLimit-Limit", strconv.Itoa(rateLimit)).
		WithHeader("X-RateLimit-Remaining", strconv.Itoa(remaining)).
		WithHeader("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10)).
		WithHeader("X-Client-IP", clientIP)
}
