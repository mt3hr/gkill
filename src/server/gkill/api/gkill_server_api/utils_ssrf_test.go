package gkill_server_api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// SSRF ロジック本体（IsDisallowedFetchIP / スキーム検査 / サイズ上限）は
// api/safefetch/safefetch_test.go へ移設した。ここは httpGetBase64Data が
// そのラッパとして従来どおり loopback とスキームを弾くことだけを確認する。

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
