package api

import (
	"net/http"
	"sync"
	"time"
)

// FixedWindowRateLimiter limits requests per window using a fixed-window algorithm.
type FixedWindowRateLimiter struct {
	mu      sync.Mutex
	windows map[string]*rateWindow
	limit   int
	window  time.Duration
}

type rateWindow struct {
	count    int
	startsAt time.Time
}

func NewRateLimiter(limit int, window time.Duration) *FixedWindowRateLimiter {
	return &FixedWindowRateLimiter{
		windows: make(map[string]*rateWindow),
		limit:   limit,
		window:  window,
	}
}

func (rl *FixedWindowRateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	w, ok := rl.windows[key]
	if !ok || now.Sub(w.startsAt) >= rl.window {
		rl.windows[key] = &rateWindow{count: 1, startsAt: now}
		return true
	}
	if w.count >= rl.limit {
		return false
	}
	w.count++
	return true
}

// RateLimit returns a middleware that applies per-IP rate limiting.
func RateLimit(limiter *FixedWindowRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.Header.Get("X-Forwarded-For")
			if ip == "" {
				ip = r.Header.Get("X-Real-IP")
			}
			if ip == "" {
				ip = r.RemoteAddr
			}
			if !limiter.Allow(ip) {
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
