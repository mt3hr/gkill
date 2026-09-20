package mcp

// GkillClient（読み書きサーバの client）の検査。
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
func TestGkillClientConstructor(t *testing.T) {
	t.Run("sets defaults when env vars are absent", func(t *testing.T) {
		clearGkillEnv(t)
		client := NewGkillClientFromEnv()
		expectEqual(t, client.BaseURL, "http://127.0.0.1:9999")
		expectEqual(t, client.UserID(), "")
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
		transport, ok := client.HTTPClient.Transport.(*http.Transport)
		expectTrue(t, ok, "transport is %T", client.HTTPClient.Transport)
		expectTrue(t, transport.TLSClientConfig != nil && transport.TLSClientConfig.InsecureSkipVerify, "TLS verification is still on")
	})
}

// ---------------------------------------------------------------------------
// resolvePasswordSha256
// ---------------------------------------------------------------------------
func TestReadWriteResolvePasswordSha256(t *testing.T) {
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
	})

	t.Run("returns empty string when no credentials", func(t *testing.T) {
		clearGkillEnv(t)
		client := NewGkillClientFromEnv()
		expectEqual(t, client.ResolvePasswordSha256(), "")
	})
}

// ---------------------------------------------------------------------------
// login
// ---------------------------------------------------------------------------
func TestReadWriteLogin(t *testing.T) {
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
	})

	t.Run("throws when credentials are missing", func(t *testing.T) {
		clearGkillEnv(t)
		client := NewGkillClientFromEnv()
		_, err := client.Login(context.Background())
		expectErrorContains(t, err, "Missing login credentials")
	})
}

// ---------------------------------------------------------------------------
// callApi
// ---------------------------------------------------------------------------
func TestReadWriteCallApi(t *testing.T) {
	t.Run("posts to the correct endpoint with auth", func(t *testing.T) {
		clearGkillEnv(t)
		t.Setenv("GKILL_USER", "admin")
		t.Setenv("GKILL_PASSWORD_SHA256", "hash")
		client := NewGkillClientFromEnv()
		fetch := installFetchMock(client)
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
				return httpJSON(200, `{"session_id":"old","errors":[]}`), nil
			case 2:
				return httpJSON(200, `{"errors":[{"error_code":"ERR000013"}]}`), nil
			case 3:
				return httpJSON(200, `{"session_id":"new","errors":[]}`), nil
			}
			return httpJSON(200, `{"kyous":[],"errors":[]}`), nil
		}

		_, err := client.CallApi(context.Background(), "/api/get_kyous_mcp", obj(), true, "")
		expectNoError(t, err)
		expectTrue(t, len(fetch.calls) == 4, "fetch called %d times", len(fetch.calls))
	})
}

// ---------------------------------------------------------------------------
// fetchFile
// ---------------------------------------------------------------------------
func TestReadWriteFetchFile(t *testing.T) {
	t.Run("fetches file with session cookie", func(t *testing.T) {
		clearGkillEnv(t)
		client := NewGkillClientFromEnv()
		fetch := installFetchMock(client)
		fetch.impl = func(_ int, _ fetchCall) (*http.Response, error) {
			return httpRaw(200, "image/png", make([]byte, 4)), nil
		}

		result, err := client.FetchFile(context.Background(), "/files/rep/test.png", "sess-123")
		expectNoError(t, err)
		expectEqual(t, result.ContentType, "image/png")
		expectTrue(t, len(result.Buffer) == 4, "buffer has %d bytes", len(result.Buffer))

		mustContain(t, fetch.calls[0].Header.Get("Cookie"), "sess-123")
	})
}
