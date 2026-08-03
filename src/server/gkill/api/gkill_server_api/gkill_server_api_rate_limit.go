package gkill_server_api

import (
	"net"
	"net/http"
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

// isLoopbackRemoteAddr は接続元がループバックかを返す。
// 同一マシンからのアクセスにだけ許す処理 (初回セットアップのリセットトークン受け渡しなど) の判定に使う。
func isLoopbackRemoteAddr(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	host = strings.Trim(host, "[]")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// forwardingHeaders はリバースプロキシが付ける転送ヘッダ。
// これらが付いている時点で接続元はプロキシであり、r.RemoteAddr がループバックでも
// 実際の要求元は別のどこかなので、ループバック限定の処理を通してはいけない。
var forwardingHeaders = []string{
	"X-Forwarded-For",
	"X-Real-Ip",
	"Forwarded",
	"X-Forwarded-Host",
}

// hasForwardingHeader はリクエストにリバースプロキシ由来の転送ヘッダが付いているかを返す。
func hasForwardingHeader(r *http.Request) bool {
	for _, headerName := range forwardingHeaders {
		if r.Header.Get(headerName) != "" {
			return true
		}
	}
	return false
}

// isTrustedLocalRequest は「同一マシンから直接来た」と言い切れるリクエストかを返す。
func isTrustedLocalRequest(r *http.Request) bool {
	return isLoopbackRemoteAddr(r.RemoteAddr) && !hasForwardingHeader(r)
}
