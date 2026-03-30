package api

import (
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := newLoginRateLimiter()
	ip := "192.168.1.1"

	// First 10 attempts should be allowed
	for i := range 10 {
		if !rl.allow(ip) {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
	}

	// 11th attempt should be denied
	if rl.allow(ip) {
		t.Fatal("attempt 11 should be denied")
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	rl := newLoginRateLimiter()

	// Exhaust limit for IP1
	for range 10 {
		rl.allow("10.0.0.1")
	}

	// IP2 should still be allowed
	if !rl.allow("10.0.0.2") {
		t.Fatal("different IP should be allowed independently")
	}
}

func TestRateLimiter_WindowExpiry(t *testing.T) {
	rl := &loginRateLimiter{
		attempts: make(map[string][]time.Time),
		limit:    10,
		window:   15 * time.Minute,
	}
	ip := "172.16.0.1"

	// Add 10 attempts that are 16 minutes old (expired)
	old := time.Now().Add(-16 * time.Minute)
	for range 10 {
		rl.attempts[ip] = append(rl.attempts[ip], old)
	}

	// Should be allowed because old attempts are outside the window
	if !rl.allow(ip) {
		t.Fatal("should be allowed after window expiry")
	}
}

func TestExtractIP_IPv4WithPort(t *testing.T) {
	got := extractIP("192.168.1.1:8080")
	if got != "192.168.1.1" {
		t.Errorf("expected '192.168.1.1', got %q", got)
	}
}

func TestExtractIP_IPv6WithPort(t *testing.T) {
	got := extractIP("[::1]:8080")
	if got != "[::1]" {
		t.Errorf("expected '[::1]', got %q", got)
	}
}

func TestExtractIP_NoPort(t *testing.T) {
	got := extractIP("192.168.1.1")
	if got != "192.168.1.1" {
		t.Errorf("expected '192.168.1.1', got %q", got)
	}
}
