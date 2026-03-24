package api

import (
	"strings"
	"sync"
	"time"
)

type loginRateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	limit    int
	window   time.Duration
}

func newLoginRateLimiter() *loginRateLimiter {
	return &loginRateLimiter{
		attempts: make(map[string][]time.Time),
		limit:    10,
		window:   15 * time.Minute,
	}
}

func (rl *loginRateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-rl.window)
	existing := rl.attempts[ip]
	var recent []time.Time
	for _, t := range existing {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	if len(recent) >= rl.limit {
		rl.attempts[ip] = recent
		return false
	}
	rl.attempts[ip] = append(recent, now)
	return true
}

func extractIP(remoteAddr string) string {
	spl := strings.Split(remoteAddr, ":")
	if len(spl) > 1 {
		return strings.Join(spl[:len(spl)-1], ":")
	}
	return remoteAddr
}
