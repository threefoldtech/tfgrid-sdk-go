package ratelimiter

import (
	"net"
	"net/http"
	"strings"
)

// GetClientIP extracts the real client IP from the HTTP request
// It checks X-Forwarded-For, X-Real-IP headers, and falls back to RemoteAddr
func GetClientIP(r *http.Request) string {
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
