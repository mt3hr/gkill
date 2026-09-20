package mcp

// GkillClient（読み取りサーバの client）の検査。
//
// ネットワークは全部モック（http.Client の Transport 差し替え）なので実サーバは要らない。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"testing"
)

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------
func TestGkillReadClientConstructor(t *testing.T) {
	t.Run("sets defaults when env vars are absent", func(t *testing.T) {
		clearGkillEnv(t)
		client := NewGkillClientFromEnv()
		expectEqual(t, client.BaseURL, "http://127.0.0.1:9999")
		expectEqual(t, client.UserID(), "")
		expectEqual(t, client.PasswordSha256, "")
		expectEqual(t, client.Password, "")
		expectEqual(t, client.DefaultLocale(), "ja")
		expectEqual(t, client.SessionID(), "")
		expectTrue(t, !client.Insecure, "insecure transport without GKILL_INSECURE")
	})

	t.Run("reads env vars when set", func(t *testing.T) {
		clearGkillEnv(t)
		t.Setenv("GKILL_BASE_URL", "https://example.com:1234")
		t.Setenv("GKILL_USER", "testuser")
		t.Setenv("GKILL_PASSWORD_SHA256", "abc123")
		t.Setenv("GKILL_LOCALE", "en")
		t.Setenv("GKILL_SESSION_ID", "sess-42")

		client := NewGkillClientFromEnv()
		expectEqual(t, client.BaseURL, "https://example.com:1234")
		expectEqual(t, client.UserID(), "testuser")
		expectEqual(t, client.PasswordSha256, "abc123")
		expectEqual(t, client.DefaultLocale(), "en")
		expectEqual(t, client.SessionID(), "sess-42")
	})
}

// ---------------------------------------------------------------------------
// buildApiUrl
// ---------------------------------------------------------------------------
func TestBuildApiURL(t *testing.T) {
	t.Run("constructs correct URL", func(t *testing.T) {
		clearGkillEnv(t)
		client := NewGkillClientFromEnv()
		expectEqual(t, client.BuildApiURL("/api/login"), "http://127.0.0.1:9999/api/login")
	})

	t.Run("constructs URL with custom base", func(t *testing.T) {
		clearGkillEnv(t)
		t.Setenv("GKILL_BASE_URL", "https://gkill.example.com")
		client := NewGkillClientFromEnv()
		expectEqual(t, client.BuildApiURL("/api/get_kyous_mcp"), "https://gkill.example.com/api/get_kyous_mcp")
	})
}

// ---------------------------------------------------------------------------
// resolvePasswordSha256
// ---------------------------------------------------------------------------
func TestResolvePasswordSha256(t *testing.T) {
	t.Run("returns GKILL_PASSWORD_SHA256 directly when set", func(t *testing.T) {
		clearGkillEnv(t)
		t.Setenv("GKILL_PASSWORD_SHA256", "deadbeef")
		client := NewGkillClientFromEnv()
		expectEqual(t, client.ResolvePasswordSha256(), "deadbeef")
	})

	t.Run("computes SHA-256 from GKILL_PASSWORD when hash is not set", func(t *testing.T) {
		clearGkillEnv(t)
		t.Setenv("GKILL_PASSWORD", "secret")
		client := NewGkillClientFromEnv()
		hash := client.ResolvePasswordSha256()
		// SHA-256 of "secret"
		sum := sha256.Sum256([]byte("secret"))
		expectEqual(t, hash, hex.EncodeToString(sum[:]))
	})

	t.Run("returns empty string when neither env var is set", func(t *testing.T) {
		clearGkillEnv(t)
		client := NewGkillClientFromEnv()
		expectEqual(t, client.ResolvePasswordSha256(), "")
	})
}

// ---------------------------------------------------------------------------
// hasErrors
// ---------------------------------------------------------------------------
func TestHasErrors(t *testing.T) {
	clearGkillEnv(t)
	client := NewGkillClientFromEnv()

	t.Run("returns true when errors array is non-empty", func(t *testing.T) {
		expectTrue(t, client.HasErrors(obj("errors", arr(obj("error_code", "ERR")))), "expected true")
	})

	t.Run("returns false when errors array is empty", func(t *testing.T) {
		expectTrue(t, !client.HasErrors(obj("errors", arr())), "expected false")
	})

	t.Run("returns false when errors field is missing", func(t *testing.T) {
		expectTrue(t, !client.HasErrors(obj()), "expected false")
	})

	t.Run("returns false for null/undefined", func(t *testing.T) {
		expectTrue(t, !client.HasErrors(nil), "expected false for nil")
	})
}

// ---------------------------------------------------------------------------
// login
// ---------------------------------------------------------------------------
func TestLogin(t *testing.T) {
	t.Run("calls fetch with correct URL and body", func(t *testing.T) {
		clearGkillEnv(t)
		t.Setenv("GKILL_USER", "admin")
		t.Setenv("GKILL_PASSWORD_SHA256", "hash123")
		client := NewGkillClientFromEnv()
		fetch := installFetchMock(client)
		fetch.mockFetchOk(`{"session_id":"test-session","errors":[]}`)

		sessionID, err := client.Login(context.Background())
		expectNoError(t, err)
		expectEqual(t, sessionID, "test-session")
		expectTrue(t, len(fetch.calls) == 1, "fetch called %d times", len(fetch.calls))

		call := fetch.calls[0]
		expectEqual(t, call.URL, "http://127.0.0.1:9999/api/login")
		expectEqual(t, call.Method, "POST")
		expectEqual(t, parse(t, call.Body), obj(
			"user_id", "admin",
			"password_sha256", "hash123",
			"locale_name", "ja",
		))
	})

	t.Run("returns existing sessionId without calling fetch", func(t *testing.T) {
		clearGkillEnv(t)
		t.Setenv("GKILL_SESSION_ID", "existing-session")
		client := NewGkillClientFromEnv()
		fetch := installFetchMock(client)
		fetch.mockFetchOk(`{}`)

		sessionID, err := client.Login(context.Background())
		expectNoError(t, err)
		expectEqual(t, sessionID, "existing-session")
		expectTrue(t, len(fetch.calls) == 0, "fetch was called")
	})

	t.Run("throws when credentials are missing", func(t *testing.T) {
		clearGkillEnv(t)
		client := NewGkillClientFromEnv()
		_, err := client.Login(context.Background())
		expectErrorContains(t, err, "Missing login credentials")
	})
}

// ---------------------------------------------------------------------------
// post
// ---------------------------------------------------------------------------
func TestPost(t *testing.T) {
	t.Run("sends correct request and returns JSON body", func(t *testing.T) {
		clearGkillEnv(t)
		client := NewGkillClientFromEnv()
		fetch := installFetchMock(client)
		fetch.mockFetchOk(`{"data":"result","errors":[]}`)

		result, err := client.Post(context.Background(), "/api/test", obj("key", "value"))
		expectNoError(t, err)
		expectEqual(t, result, obj("data", "result", "errors", arr()))
		expectTrue(t, len(fetch.calls) == 1, "fetch called %d times", len(fetch.calls))

		call := fetch.calls[0]
		expectEqual(t, call.URL, "http://127.0.0.1:9999/api/test")
		expectEqual(t, call.Method, "POST")
		expectEqual(t, call.Header.Get("Content-Type"), "application/json")
		expectEqual(t, parse(t, call.Body), obj("key", "value"))
	})

	// gkill は 2026-08 から異常時に 4xx/5xx を返す。ステータスだけで throw すると
	// 本文の errors が呼び出し側へ届かず、CallApi の再ログインが不通になる。
	t.Run("returns the body on non-ok response when it is a gkill envelope", func(t *testing.T) {
		clearGkillEnv(t)
		client := NewGkillClientFromEnv()
		fetch := installFetchMock(client)
		fetch.mockFetchError(500, `{"errors":[{"error_code":"ERR000410"}],"messages":null}`)

		result, err := client.Post(context.Background(), "/api/fail", obj())
		expectNoError(t, err)
		expectEqual(t, objAt(t, arrAt(t, result, "errors")[0]).Value("error_code"), "ERR000410")
	})

	t.Run("throws GkillApiError on non-ok response that is not a gkill envelope", func(t *testing.T) {
		clearGkillEnv(t)
		client := NewGkillClientFromEnv()
		fetch := installFetchMock(client)
		fetch.mockFetchError(502, `{"message":"Bad Gateway"}`)

		_, err := client.Post(context.Background(), "/api/fail", obj())
		expectGkillApiError(t, err)
		expectErrorContains(t, err, "HTTP 502")
	})

	// 本文が JSON でない非2xx (リバースプロキシの HTML 502/504、本文なしの 413/500 等) を
	// 「Failed to parse JSON response」にすると、HTTP ステータスが呼び出し側へ一切届かない。
	t.Run("includes the HTTP status when a non-2xx response body is not JSON", func(t *testing.T) {
		clearGkillEnv(t)
		client := NewGkillClientFromEnv()
		fetch := installFetchMock(client)
		fetch.impl = func(_ int, _ fetchCall) (*http.Response, error) {
			return httpRaw(502, "text/html", []byte("<html>bad gateway</html>")), nil
		}

		_, err := client.Post(context.Background(), "/api/fail", obj())
		expectErrorContains(t, err, "HTTP 502")
	})

	t.Run("keeps the parse error when a 2xx response body is not JSON", func(t *testing.T) {
		clearGkillEnv(t)
		client := NewGkillClientFromEnv()
		fetch := installFetchMock(client)
		fetch.impl = func(_ int, _ fetchCall) (*http.Response, error) {
			return httpRaw(200, "application/json", []byte("")), nil
		}

		_, err := client.Post(context.Background(), "/api/broken", obj())
		expectErrorContains(t, err, "Failed to parse JSON response")
	})

	t.Run("throws GkillApiError on network failure", func(t *testing.T) {
		clearGkillEnv(t)
		client := NewGkillClientFromEnv()
		fetch := installFetchMock(client)
		fetch.mockFetchReject(errors.New("ECONNREFUSED"))

		_, err := client.Post(context.Background(), "/api/down", obj())
		expectGkillApiError(t, err)
		expectErrorContains(t, err, "Network error")
	})
}

// ---------------------------------------------------------------------------
// fetchFile
// ---------------------------------------------------------------------------
func TestFetchFile(t *testing.T) {
	// /files/ はエンベロープ(errors配列)を返さないので、セッション切れは HTTP 401 でしか
	// 分からない。CallApi と同じく、401 なら1回だけログインし直して再試行する。
	// MCP は長寿命プロセスなので、これが無いと期限切れ以降ファイル取得が復旧不能になる。
	t.Run("re-logins once and retries when the file request returns HTTP 401", func(t *testing.T) {
		clearGkillEnv(t)
		t.Setenv("GKILL_USER", "admin")
		t.Setenv("GKILL_PASSWORD_SHA256", "hash")
		client := NewGkillClientFromEnv()
		fetch := installFetchMock(client)
		fetch.impl = func(callCount int, _ fetchCall) (*http.Response, error) {
			switch callCount {
			case 1:
				// First file request with the expired session
				return httpRaw(401, "", nil), nil
			case 2:
				// Re-login
				return httpJSON(200, `{"session_id":"new-sess","errors":null}`), nil
			}
			// Retry succeeds
			return httpRaw(200, "image/png", make([]byte, 4)), nil
		}

		result, err := client.FetchFile(context.Background(), "/files/rep/photo.png", "expired-sess")
		expectNoError(t, err)
		expectEqual(t, result.ContentType, "image/png")
		expectTrue(t, len(result.Buffer) == 4, "buffer has %d bytes", len(result.Buffer))
		expectTrue(t, len(fetch.calls) == 3, "fetch called %d times", len(fetch.calls))

		// 再試行はログインし直したセッションの Cookie で行われること
		mustContain(t, fetch.calls[2].Header.Get("Cookie"), "new-sess")
	})

	t.Run("throws when the retry after re-login still returns HTTP 401", func(t *testing.T) {
		clearGkillEnv(t)
		t.Setenv("GKILL_USER", "admin")
		t.Setenv("GKILL_PASSWORD_SHA256", "hash")
		client := NewGkillClientFromEnv()
		fetch := installFetchMock(client)
		fetch.impl = func(callCount int, _ fetchCall) (*http.Response, error) {
			if callCount == 2 {
				// Re-login
				return httpJSON(200, `{"session_id":"new-sess","errors":null}`), nil
			}
			// First attempt and the retry both get 401
			return httpRaw(401, "", nil), nil
		}

		_, err := client.FetchFile(context.Background(), "/files/rep/photo.png", "expired-sess")
		expectErrorContains(t, err, "HTTP 401 fetching file")
		expectTrue(t, len(fetch.calls) == 3, "fetch called %d times", len(fetch.calls))
	})

	// 401 以外は再ログインしない。read_handlers の 404→原因名指しの変換は
	// 生の「HTTP 404 fetching file」がそのまま1回で返ることに依存している。
	t.Run("throws immediately without re-login on non-401 errors", func(t *testing.T) {
		clearGkillEnv(t)
		client := NewGkillClientFromEnv()
		fetch := installFetchMock(client)
		fetch.impl = func(_ int, _ fetchCall) (*http.Response, error) { return httpRaw(404, "", nil), nil }

		_, err := client.FetchFile(context.Background(), "/files/rep/missing.png", "sess")
		expectErrorContains(t, err, "HTTP 404 fetching file")
		expectTrue(t, len(fetch.calls) == 1, "fetch called %d times", len(fetch.calls))
	})
}
