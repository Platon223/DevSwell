package httpapi

import (
	"net"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/time/rate"
)

// ipRateLimiter tracks a token-bucket rate limiter per client IP. It's a
// simple in-memory limiter suited to a single-instance deployment; the map
// grows unbounded over time, which is fine at this project's current scale.
type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	burst    int
}

func newIPRateLimiter(r rate.Limit, burst int) *ipRateLimiter {
	return &ipRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		r:        r,
		burst:    burst,
	}
}

func (l *ipRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	limiter, exists := l.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(l.r, l.burst)
		l.limiters[ip] = limiter
	}
	l.mu.Unlock()
	return limiter.Allow()
}

func rateLimited(next http.HandlerFunc, limiter *ipRateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !limiter.allow(clientIP(r)) {
			http.Error(w, "too many requests, please try again later", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

// clientIP prefers X-Forwarded-For (set by Railway's proxy in front of the
// app) and falls back to the direct connection's address for local dev.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
