package mcp

// GkillClient（書き込みサーバの client）の検査。
//
// ネットワークは全部モック（http.Client の Transport 差し替え）なので実サーバは要らない。

import (
	"context"
	"net/http"
	"testing"
)

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------
func TestGkillWriteClientConstructor(t *testing.T) {
	t.Run("sets defaults when env vars are absent", func(t *testing.T) {
		clearGkillEnv(t)
		client := NewGkillClientFromEnv()
		expectEqual(t, client.BaseURL, "http://127.0.0.1:9999")
		expectEqual(t, client.UserID(), "")
		expectEqual(t, client.PasswordSha256, "")
		expectEqual(t, client.Password, "")
		expectEqual(t, client.DefaultLocale(), "ja")
		expectEqual(t, client.SessionID(), "")
		expectTrue(t, !client.Insecure, "insecure without GKILL_INSECURE")
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

	t.Run("sets dispatcher when GKILL_INSECURE=true", func(t *testing.T) {
		clearGkillEnv(t)
		t.Setenv("GKILL_INSECURE", "true")
		client := NewGkillClientFromEnv()
		expectTrue(t, client.Insecure, "Insecure is false")
	})
}

// ---------------------------------------------------------------------------
// resolvePasswordSha256
// ---------------------------------------------------------------------------
func TestWriteResolvePasswordSha256(t *testing.T) {
	t.Run("returns passwordSha256 when set", func(t *testing.T) {
		clearGkillEnv(t)
		t.Setenv("GKILL_PASSWORD_SHA256", "abc123")
		client := NewGkillClientFromEnv()
		expectEqual(t, client.ResolvePasswordSha256(), "abc123")
	})

	t.Run("hashes password when only password is set", func(t *testing.T) {
		clearGkillEnv(t)
		t.Setenv("GKILL_PASSWORD", "secret")
		client := NewGkillClientFromEnv()
		hash := client.ResolvePasswordSha256()
		expectTrue(t, len(hash) == 64, "hash length %d", len(hash))
		expectTrue(t, hash != "secret", "hash equals the plain password")
	})

	t.Run("returns empty string when no credentials", func(t *testing.T) {
		clearGkillEnv(t)
		client := NewGkillClientFromEnv()
		expectEqual(t, client.ResolvePasswordSha256(), "")
	})
}

// ---------------------------------------------------------------------------
// buildApiUrl
// ---------------------------------------------------------------------------
func TestWriteBuildApiURL(t *testing.T) {
	t.Run("constructs correct URL", func(t *testing.T) {
		clearGkillEnv(t)
		client := NewGkillClientFromEnv()
		expectEqual(t, client.BuildApiURL("/api/add_kmemo"), "http://127.0.0.1:9999/api/add_kmemo")
	})
}

// ---------------------------------------------------------------------------
// hasErrors / hasAuthErrors / formatErrors
// ---------------------------------------------------------------------------
func TestErrorHelpers(t *testing.T) {
	clearGkillEnv(t)
	client := NewGkillClientFromEnv()

	t.Run("hasErrors returns false for empty errors", func(t *testing.T) {
		expectTrue(t, !client.HasErrors(obj("errors", arr())), "expected false")
	})

	t.Run("hasErrors returns true for non-empty errors", func(t *testing.T) {
		expectTrue(t, client.HasErrors(obj("errors", arr(obj("error_code", "E1")))), "expected true")
	})

	t.Run("hasAuthErrors detects auth error codes", func(t *testing.T) {
		expectTrue(t, client.HasAuthErrors(obj("errors", arr(obj("error_code", "ERR000013")))), "ERR000013 not detected")
		expectTrue(t, !client.HasAuthErrors(obj("errors", arr(obj("error_code", "ERR999999")))), "ERR999999 detected")
	})

	t.Run("formatErrors joins error messages", func(t *testing.T) {
		result := client.FormatErrors(obj("errors", arr(
			obj("error_code", "E1", "error_message", "msg1"),
			obj("error_code", "E2", "error_message", "msg2"),
		)))
		expectEqual(t, result, "E1: msg1; E2: msg2")
	})

	t.Run("formatErrors appends the server's error_kind / reason tokens when present", func(t *testing.T) {
		result := client.FormatErrors(obj("errors", arr(
			obj("error_code", "ERR000023", "error_message", "failed", "error_kind", "config", "reason", "write_rep_missing"),
			obj("error_code", "ERR000070", "error_message", "not found", "error_kind", "not_found"),
		)))
		expectEqual(t, result, "ERR000023: failed [kind=config, reason=write_rep_missing]; ERR000070: not found [kind=not_found]")
	})
}

// ---------------------------------------------------------------------------
// login
// ---------------------------------------------------------------------------
func TestWriteLogin(t *testing.T) {
	t.Run("returns existing session_id without calling API", func(t *testing.T) {
		clearGkillEnv(t)
		t.Setenv("GKILL_SESSION_ID", "existing-session")
		client := NewGkillClientFromEnv()
		result, err := client.Login(context.Background())
		expectNoError(t, err)
		expectEqual(t, result, "existing-session")
	})

	t.Run("calls /api/login and returns session_id", func(t *testing.T) {
		clearGkillEnv(t)
		t.Setenv("GKILL_USER", "admin")
		t.Setenv("GKILL_PASSWORD_SHA256", "hash")
		client := NewGkillClientFromEnv()
		fetch := installFetchMock(client)
		fetch.mockFetchOk(`{"session_id":"new-session","errors":[]}`)

		result, err := client.Login(context.Background())
		expectNoError(t, err)
		expectEqual(t, result, "new-session")
		expectEqual(t, client.SessionID(), "new-session")
	})

	t.Run("throws when credentials are missing", func(t *testing.T) {
		clearGkillEnv(t)
		client := NewGkillClientFromEnv()
		_, err := client.Login(context.Background())
		expectErrorContains(t, err, "Missing login credentials")
	})

	t.Run("throws when login returns errors", func(t *testing.T) {
		clearGkillEnv(t)
		t.Setenv("GKILL_USER", "admin")
		t.Setenv("GKILL_PASSWORD_SHA256", "hash")
		client := NewGkillClientFromEnv()
		fetch := installFetchMock(client)
		fetch.mockFetchOk(`{"errors":[{"error_code":"E1","error_message":"bad password"}]}`)

		_, err := client.Login(context.Background())
		expectErrorContains(t, err, "Login failed")
	})
}

// ---------------------------------------------------------------------------
// callApi
// ---------------------------------------------------------------------------
func TestWriteCallApi(t *testing.T) {
	t.Run("posts to the correct endpoint with auth", func(t *testing.T) {
		clearGkillEnv(t)
		t.Setenv("GKILL_USER", "admin")
		t.Setenv("GKILL_PASSWORD_SHA256", "hash")
		client := NewGkillClientFromEnv()
		fetch := installFetchMock(client)
		// First call = login, second = actual API call
		fetch.impl = func(callCount int, _ fetchCall) (*http.Response, error) {
			if callCount == 1 {
				return httpJSON(200, `{"session_id":"sess","errors":[]}`), nil
			}
			return httpJSON(200, `{"added_kmemo":{"id":"123"},"errors":[]}`), nil
		}

		result, err := client.CallApi(context.Background(), "/api/add_kmemo", obj("kmemo", obj()), true, "")
		expectNoError(t, err)
		expectEqual(t, objAt(t, result, "added_kmemo").Value("id"), "123")
	})

	t.Run("retries on auth error", func(t *testing.T) {
		clearGkillEnv(t)
		t.Setenv("GKILL_USER", "admin")
		t.Setenv("GKILL_PASSWORD_SHA256", "hash")
		client := NewGkillClientFromEnv()
		fetch := installFetchMock(client)
		fetch.impl = func(callCount int, _ fetchCall) (*http.Response, error) {
			switch callCount {
			case 1:
				// Initial login
				return httpJSON(200, `{"session_id":"old-sess","errors":[]}`), nil
			case 2:
				// First attempt returns auth error
				return httpJSON(200, `{"errors":[{"error_code":"ERR000013","error_message":"session expired"}]}`), nil
			case 3:
				// Re-login
				return httpJSON(200, `{"session_id":"new-sess","errors":[]}`), nil
			}
			// Retry succeeds
			return httpJSON(200, `{"added_kmemo":{"id":"456"},"errors":[]}`), nil
		}

		result, err := client.CallApi(context.Background(), "/api/add_kmemo", obj("kmemo", obj()), true, "")
		expectNoError(t, err)
		expectEqual(t, objAt(t, result, "added_kmemo").Value("id"), "456")
		expectTrue(t, len(fetch.calls) == 4, "fetch called %d times", len(fetch.calls))
	})

	// gkill がセッション切れに 401 を返すようになっても、再ログインが走ること。
	// ステータスで打ち切ると HasAuthErrors に到達せず、MCP は長寿命プロセスなので
	// 期限切れ以降ずっと復旧できなくなる。
	t.Run("re-logins once when the session expires with HTTP 401", func(t *testing.T) {
		clearGkillEnv(t)
		t.Setenv("GKILL_USER", "admin")
		t.Setenv("GKILL_PASSWORD_SHA256", "hash")
		client := NewGkillClientFromEnv()
		fetch := installFetchMock(client)
		fetch.impl = func(callCount int, _ fetchCall) (*http.Response, error) {
			switch callCount {
			case 1:
				return httpJSON(200, `{"session_id":"old-sess","errors":null}`), nil
			case 2:
				return httpJSON(401, `{"errors":[{"error_code":"ERR000373","error_message":"session expired"}],"messages":null}`), nil
			case 3:
				return httpJSON(200, `{"session_id":"new-sess","errors":null}`), nil
			}
			return httpJSON(200, `{"added_kmemo":{"id":"456"},"errors":null}`), nil
		}

		result, err := client.CallApi(context.Background(), "/api/add_kmemo", obj("kmemo", obj()), true, "")
		expectNoError(t, err)
		expectEqual(t, objAt(t, result, "added_kmemo").Value("id"), "456")
		expectTrue(t, len(fetch.calls) == 4, "fetch called %d times", len(fetch.calls))
	})

	t.Run("uses session override when provided", func(t *testing.T) {
		clearGkillEnv(t)
		client := NewGkillClientFromEnv()
		fetch := installFetchMock(client)
		fetch.mockFetchOk(`{"added_kmemo":{"id":"789"},"errors":[]}`)
		client.SetSessionID("default-sess")

		_, err := client.CallApi(context.Background(), "/api/add_kmemo", obj("kmemo", obj()), true, "override-sess")
		expectNoError(t, err)
		callBody := parseObj(t, fetch.calls[0].Body)
		expectEqual(t, callBody.Value("session_id"), "override-sess")
	})
}
