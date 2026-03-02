package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	window  time.Duration
	max     int
	clients sync.Map // ip -> *clientWindow
}

type clientWindow struct {
	mu         sync.Mutex
	timestamps []time.Time
}

func NewRateLimiter(window time.Duration, max int) *RateLimiter {
	rl := &RateLimiter{window: window, max: max}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		if ip == "" {
			ip = r.RemoteAddr
		}

		val, _ := rl.clients.LoadOrStore(ip, &clientWindow{})
		cw := val.(*clientWindow)

		cw.mu.Lock()
		now := time.Now()
		cutoff := now.Add(-rl.window)

		// Remove expired timestamps
		valid := cw.timestamps[:0]
		for _, ts := range cw.timestamps {
			if ts.After(cutoff) {
				valid = append(valid, ts)
			}
		}
		cw.timestamps = valid

		if len(cw.timestamps) >= rl.max {
			oldest := cw.timestamps[0]
			retryAfter := oldest.Add(rl.window).Sub(now).Seconds()
			cw.mu.Unlock()
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		cw.timestamps = append(cw.timestamps, now)
		cw.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-rl.window)
		rl.clients.Range(func(key, value any) bool {
			cw := value.(*clientWindow)
			cw.mu.Lock()
			allExpired := true
			for _, ts := range cw.timestamps {
				if ts.After(cutoff) {
					allExpired = false
					break
				}
			}
			cw.mu.Unlock()
			if allExpired {
				rl.clients.Delete(key)
			}
			return true
		})
	}
}
