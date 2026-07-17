package gkill_server_api

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsDisallowedFetchIP(t *testing.T) {
	disallowed := []string{
		"127.0.0.1",
		"10.0.0.1",
		"172.16.0.1",
		"192.168.1.1",
		"169.254.169.254",
		"0.0.0.0",
		"::1",
		"fe80::1",
		"fc00::1",
	}
	for _, s := range disallowed {
		if !isDisallowedFetchIP(net.ParseIP(s)) {
			t.Errorf("isDisallowedFetchIP(%s) = false, want true", s)
		}
	}

	allowed := []string{
		"93.184.216.34",
		"8.8.8.8",
		"2001:4860:4860::8888",
	}
	for _, s := range allowed {
		if isDisallowedFetchIP(net.ParseIP(s)) {
			t.Errorf("isDisallowedFetchIP(%s) = true, want false", s)
		}
	}

	if !isDisallowedFetchIP(nil) {
		t.Error("isDisallowedFetchIP(nil) = false, want true")
	}
}

func TestHttpGetBase64Data_BlocksLoopback(t *testing.T) {
	// loopbackで実際にサーバを立てても、接続段階で拒否されることを確認する
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("request to loopback server should have been blocked")
	}))
	defer server.Close()

	_, err := httpGetBase64Data(server.URL)
	if err == nil {
		t.Fatal("httpGetBase64Data to loopback should fail")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("error should mention blocked address, got: %v", err)
	}
}

func TestHttpGetBase64Data_RejectsScheme(t *testing.T) {
	for _, u := range []string{"file:///etc/passwd", "ftp://example.com/a", "gopher://example.com"} {
		_, err := httpGetBase64Data(u)
		if err == nil || !strings.Contains(err.Error(), "unsupported url scheme") {
			t.Errorf("httpGetBase64Data(%s) should fail with scheme error, got: %v", u, err)
		}
	}
}
