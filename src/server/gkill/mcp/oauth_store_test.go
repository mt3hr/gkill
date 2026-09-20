package mcp

// oauth_store.go の検査。
//
// vitest の vi.useFakeTimers / advanceTimersByTime は、パッケージ変数 timeNow の差し替え（fakeClock）で写す。
// 定期掃除（StartCleanup）だけは実時間の ticker なので、短い間隔で回して待つ。

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// fakeClock は timeNow を固定し、advance で進める。
type fakeClock struct {
	now time.Time
}

func useFakeTimers(t *testing.T) *fakeClock {
	t.Helper()
	clock := &fakeClock{now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}
	previous := timeNow
	timeNow = func() time.Time { return clock.now }
	t.Cleanup(func() { timeNow = previous })
	return clock
}

func (c *fakeClock) advance(d time.Duration) { c.now = c.now.Add(d) }

func statsObj(s OAuthStats) *jsonobj.Object {
	return obj("codes", s.Codes, "accessTokens", s.AccessTokens, "refreshTokens", s.RefreshTokens, "clients", s.Clients)
}

// mustGet(t)(store.Get(...)) の形で使う（多値の呼び出しをそのまま渡すため）。
func mustGet(t *testing.T) func(*jsonobj.Object, bool) *jsonobj.Object {
	return func(value *jsonobj.Object, ok bool) *jsonobj.Object {
		t.Helper()
		expectTrue(t, ok, "expected a value")
		return value
	}
}

// mustMiss(t)(store.Get(...)) の形で使う。
func mustMiss(t *testing.T) func(*jsonobj.Object, bool) {
	return func(_ *jsonobj.Object, ok bool) {
		t.Helper()
		expectTrue(t, !ok, "expected null")
	}
}

// ---------------------------------------------------------------------------
// generateToken
// ---------------------------------------------------------------------------
func TestGenerateToken(t *testing.T) {
	t.Run("returns a 64-character hex string", func(t *testing.T) {
		mustMatch(t, GenerateToken(), `^[0-9a-f]{64}$`)
	})

	t.Run("generates unique tokens", func(t *testing.T) {
		tokens := NewStringSet()
		for range 50 {
			tokens.Add(GenerateToken())
		}
		expectEqual(t, tokens.Len(), 50)
	})
}

// ---------------------------------------------------------------------------
// TTL constants
// ---------------------------------------------------------------------------
func TestTTLConstants(t *testing.T) {
	t.Run("AUTHORIZATION_CODE is 5 minutes", func(t *testing.T) {
		expectEqual(t, TTLAuthorizationCode.Milliseconds(), 5*60*1000)
	})

	t.Run("ACCESS_TOKEN is 1 hour", func(t *testing.T) {
		expectEqual(t, TTLAccessToken.Milliseconds(), 60*60*1000)
	})

	t.Run("REFRESH_TOKEN is 30 days", func(t *testing.T) {
		expectEqual(t, TTLRefreshToken.Milliseconds(), 30*24*60*60*1000)
	})
}

// ---------------------------------------------------------------------------
// Authorization Codes
// ---------------------------------------------------------------------------
func TestOAuthStoreAuthorizationCodes(t *testing.T) {
	t.Run("putCode + getAndDeleteCode returns the stored data", func(t *testing.T) {
		store := NewOAuthStore("", nil)
		data := obj("clientId", "c1", "redirectUri", "http://localhost/cb")
		store.PutCode("code1", data)
		expectEqual(t, mustGet(t)(store.GetAndDeleteCode("code1")), data)
	})

	t.Run("getAndDeleteCode returns null for unknown code", func(t *testing.T) {
		store := NewOAuthStore("", nil)
		mustMiss(t)(store.GetAndDeleteCode("nonexistent"))
	})

	t.Run("getAndDeleteCode deletes the code (one-time use)", func(t *testing.T) {
		store := NewOAuthStore("", nil)
		store.PutCode("code1", obj("clientId", "c1"))
		_, _ = store.GetAndDeleteCode("code1")
		mustMiss(t)(store.GetAndDeleteCode("code1"))
	})

	t.Run("getAndDeleteCode returns null for expired code", func(t *testing.T) {
		clock := useFakeTimers(t)
		store := NewOAuthStore("", nil)
		store.PutCodeWithTTL("code1", obj("clientId", "c1"), 1000*time.Millisecond) // 1 second TTL
		clock.advance(1001 * time.Millisecond)
		mustMiss(t)(store.GetAndDeleteCode("code1"))
	})

	t.Run("getAndDeleteCode returns data before TTL expires", func(t *testing.T) {
		clock := useFakeTimers(t)
		store := NewOAuthStore("", nil)
		store.PutCodeWithTTL("code1", obj("clientId", "c1"), 5000*time.Millisecond)
		clock.advance(4999 * time.Millisecond)
		expectEqual(t, mustGet(t)(store.GetAndDeleteCode("code1")), obj("clientId", "c1"))
	})
}

// ---------------------------------------------------------------------------
// Access Tokens
// ---------------------------------------------------------------------------
func TestOAuthStoreAccessTokens(t *testing.T) {
	t.Run("putAccessToken + getAccessToken returns data", func(t *testing.T) {
		store := NewOAuthStore("", nil)
		data := obj("clientId", "c1", "userId", "testuser")
		store.PutAccessToken("tok1", data)
		expectEqual(t, mustGet(t)(store.GetAccessToken("tok1")), data)
	})

	t.Run("getAccessToken returns null for unknown token", func(t *testing.T) {
		store := NewOAuthStore("", nil)
		mustMiss(t)(store.GetAccessToken("unknown"))
	})

	t.Run("getAccessToken returns null for expired token and cleans up", func(t *testing.T) {
		clock := useFakeTimers(t)
		store := NewOAuthStore("", nil)
		store.PutAccessTokenWithTTL("tok1", obj("clientId", "c1"), 1000*time.Millisecond)
		clock.advance(1001 * time.Millisecond)
		mustMiss(t)(store.GetAccessToken("tok1"))
		// Entry should be deleted
		expectTrue(t, !store.HasAccessToken("tok1"), "expired token was kept")
	})

	t.Run("getAccessToken returns data before TTL", func(t *testing.T) {
		clock := useFakeTimers(t)
		store := NewOAuthStore("", nil)
		store.PutAccessTokenWithTTL("tok1", obj("clientId", "c1"), 5000*time.Millisecond)
		clock.advance(4999 * time.Millisecond)
		expectEqual(t, mustGet(t)(store.GetAccessToken("tok1")), obj("clientId", "c1"))
	})

	t.Run("deleteAccessToken removes the token", func(t *testing.T) {
		store := NewOAuthStore("", nil)
		store.PutAccessToken("tok1", obj("clientId", "c1"))
		store.DeleteAccessToken("tok1")
		mustMiss(t)(store.GetAccessToken("tok1"))
	})

	t.Run("can store multiple tokens", func(t *testing.T) {
		store := NewOAuthStore("", nil)
		store.PutAccessToken("tok1", obj("userId", "a"))
		store.PutAccessToken("tok2", obj("userId", "b"))
		expectEqual(t, mustGet(t)(store.GetAccessToken("tok1")), obj("userId", "a"))
		expectEqual(t, mustGet(t)(store.GetAccessToken("tok2")), obj("userId", "b"))
	})
}

// ---------------------------------------------------------------------------
// Refresh Tokens
// ---------------------------------------------------------------------------
func TestOAuthStoreRefreshTokens(t *testing.T) {
	t.Run("putRefreshToken + getRefreshToken returns data", func(t *testing.T) {
		store := NewOAuthStore("", nil)
		data := obj("clientId", "c1", "userId", "testuser")
		store.PutRefreshToken("rt1", data)
		expectEqual(t, mustGet(t)(store.GetRefreshToken("rt1")), data)
	})

	t.Run("getRefreshToken returns null for unknown token", func(t *testing.T) {
		store := NewOAuthStore("", nil)
		mustMiss(t)(store.GetRefreshToken("unknown"))
	})

	t.Run("getRefreshToken returns null for expired token", func(t *testing.T) {
		clock := useFakeTimers(t)
		store := NewOAuthStore("", nil)
		store.PutRefreshTokenWithTTL("rt1", obj("clientId", "c1"), 1000*time.Millisecond)
		clock.advance(1001 * time.Millisecond)
		mustMiss(t)(store.GetRefreshToken("rt1"))
	})

	t.Run("deleteRefreshToken removes the token", func(t *testing.T) {
		store := NewOAuthStore("", nil)
		store.PutRefreshToken("rt1", obj("clientId", "c1"))
		store.DeleteRefreshToken("rt1")
		mustMiss(t)(store.GetRefreshToken("rt1"))
	})
}

// ---------------------------------------------------------------------------
// Client Registrations
// ---------------------------------------------------------------------------
func TestOAuthStoreClients(t *testing.T) {
	t.Run("putClient + getClient returns metadata", func(t *testing.T) {
		store := NewOAuthStore("", nil)
		meta := obj("client_name", "Test App", "redirect_uris", strs("http://localhost/cb"))
		store.PutClient("client1", meta)
		expectEqual(t, mustGet(t)(store.GetClient("client1")), meta)
	})

	t.Run("getClient returns null for unknown client", func(t *testing.T) {
		store := NewOAuthStore("", nil)
		mustMiss(t)(store.GetClient("unknown"))
	})

	t.Run("putClient overwrites existing registration", func(t *testing.T) {
		store := NewOAuthStore("", nil)
		store.PutClient("c1", obj("client_name", "Old"))
		store.PutClient("c1", obj("client_name", "New"))
		expectEqual(t, mustGet(t)(store.GetClient("c1")), obj("client_name", "New"))
	})
}

// ---------------------------------------------------------------------------
// sweep
// ---------------------------------------------------------------------------
func TestOAuthStoreSweep(t *testing.T) {
	t.Run("removes expired entries from all maps", func(t *testing.T) {
		clock := useFakeTimers(t)
		store := NewOAuthStore("", nil)
		store.PutCodeWithTTL("c1", obj("x", 1), 1000*time.Millisecond)
		store.PutAccessTokenWithTTL("t1", obj("x", 2), 2000*time.Millisecond)
		store.PutRefreshTokenWithTTL("r1", obj("x", 3), 3000*time.Millisecond)

		// Before any expiry
		store.Sweep()
		expectEqual(t, statsObj(store.Stats()), obj("codes", 1, "accessTokens", 1, "refreshTokens", 1, "clients", 0))

		// After code expires
		clock.advance(1001 * time.Millisecond)
		store.Sweep()
		expectEqual(t, statsObj(store.Stats()), obj("codes", 0, "accessTokens", 1, "refreshTokens", 1, "clients", 0))

		// After access token expires
		clock.advance(1000 * time.Millisecond)
		store.Sweep()
		expectEqual(t, statsObj(store.Stats()), obj("codes", 0, "accessTokens", 0, "refreshTokens", 1, "clients", 0))

		// After refresh token expires
		clock.advance(1000 * time.Millisecond)
		store.Sweep()
		expectEqual(t, statsObj(store.Stats()), obj("codes", 0, "accessTokens", 0, "refreshTokens", 0, "clients", 0))
	})

	t.Run("does not remove unexpired entries", func(t *testing.T) {
		clock := useFakeTimers(t)
		store := NewOAuthStore("", nil)
		store.PutCodeWithTTL("c1", obj("x", 1), 10000*time.Millisecond)
		store.PutAccessTokenWithTTL("t1", obj("x", 2), 10000*time.Millisecond)

		clock.advance(5000 * time.Millisecond)
		store.Sweep()
		expectEqual(t, store.Stats().Codes, 1)
		expectEqual(t, store.Stats().AccessTokens, 1)
	})

	t.Run("does not remove client registrations (no TTL)", func(t *testing.T) {
		store := NewOAuthStore("", nil)
		store.PutClient("c1", obj("client_name", "App"))
		store.Sweep()
		expectEqual(t, mustGet(t)(store.GetClient("c1")), obj("client_name", "App"))
	})
}

// ---------------------------------------------------------------------------
// stats
// ---------------------------------------------------------------------------
func TestOAuthStoreStats(t *testing.T) {
	t.Run("returns zero counts for empty store", func(t *testing.T) {
		store := NewOAuthStore("", nil)
		expectEqual(t, statsObj(store.Stats()), obj("codes", 0, "accessTokens", 0, "refreshTokens", 0, "clients", 0))
	})

	t.Run("returns correct counts", func(t *testing.T) {
		store := NewOAuthStore("", nil)
		store.PutCode("c1", obj())
		store.PutCode("c2", obj())
		store.PutAccessToken("t1", obj())
		store.PutClient("cl1", obj())
		expectEqual(t, statsObj(store.Stats()), obj("codes", 2, "accessTokens", 1, "refreshTokens", 0, "clients", 1))
	})
}

// ---------------------------------------------------------------------------
// startCleanup / stopCleanup
// ---------------------------------------------------------------------------
func TestOAuthStoreCleanupInterval(t *testing.T) {
	waitFor := func(t *testing.T, cond func() bool) bool {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if cond() {
				return true
			}
			time.Sleep(5 * time.Millisecond)
		}
		return cond()
	}

	t.Run("startCleanup triggers periodic sweep", func(t *testing.T) {
		store := NewOAuthStore("", nil)
		store.PutCodeWithTTL("c1", obj(), time.Millisecond)
		store.StartCleanup(10 * time.Millisecond)
		defer store.StopCleanup()

		// Code should have expired (TTL 1ms) and been swept
		expectTrue(t, waitFor(t, func() bool { return store.CodesLen() == 0 }), "periodic sweep did not run")
	})

	t.Run("stopCleanup stops the interval", func(t *testing.T) {
		store := NewOAuthStore("", nil)
		store.StartCleanup(10 * time.Millisecond)
		store.StopCleanup()
		store.PutCodeWithTTL("c1", obj(), time.Millisecond)

		time.Sleep(50 * time.Millisecond)
		// Sweep should NOT have run since cleanup was stopped
		expectEqual(t, store.CodesLen(), 1)
	})

	t.Run("startCleanup replaces previous interval", func(t *testing.T) {
		store := NewOAuthStore("", nil)
		store.StartCleanup(10 * time.Millisecond)
		store.StartCleanup(20 * time.Millisecond) // should replace
		store.StopCleanup()
	})
}

// ---------------------------------------------------------------------------
// Persistence (JSON file)
// ---------------------------------------------------------------------------
func TestOAuthStorePersistence(t *testing.T) {
	newPersistPath := func(t *testing.T) string {
		t.Helper()
		return filepath.Join(t.TempDir(), "mcp_oauth_read_state.json")
	}
	exists := func(path string) bool {
		_, err := os.Stat(path)
		return err == nil
	}
	readState := func(t *testing.T, path string) *jsonobj.Object {
		t.Helper()
		raw, err := os.ReadFile(path)
		expectNoError(t, err)
		return parseObj(t, string(raw))
	}

	t.Run("save() writes refresh tokens and clients to JSON file", func(t *testing.T) {
		persistPath := newPersistPath(t)
		store := NewOAuthStore(persistPath, nil)
		store.PutRefreshTokenWithTTL("rt1", obj("userId", "testuser"), 60000*time.Millisecond)
		store.PutClient("c1", obj("client_name", "Test"))

		expectTrue(t, exists(persistPath), "state file missing")
		data := readState(t, persistPath)
		expectEqual(t, objAt(t, data, "refreshTokens", "rt1", "value").Value("userId"), "testuser")
		expectEqual(t, objAt(t, data, "clients", "c1").Value("client_name"), "Test")
	})

	// M-08: temp+rename の原子的書き込み。中間の .tmp を残さない。
	t.Run("save() leaves no .tmp file behind (atomic temp+rename)", func(t *testing.T) {
		persistPath := newPersistPath(t)
		store := NewOAuthStore(persistPath, nil)
		store.PutRefreshTokenWithTTL("rt1", obj("userId", "testuser"), 60000*time.Millisecond)
		expectTrue(t, !exists(persistPath+".tmp"), ".tmp left behind")
		// 完全な JSON として再 load できる。
		store2 := NewOAuthStore(persistPath, nil)
		store2.Load()
		expectEqual(t, mustGet(t)(store2.GetRefreshToken("rt1")), obj("userId", "testuser"))
	})

	// M-08: POSIX では 0600 で書く（他ローカルユーザから refresh token を読めなくする）。
	// Windows では mode は実質 no-op なので検査しない。
	t.Run("save() writes the state file with 0600 permissions on POSIX", func(t *testing.T) {
		if isWindows() {
			t.Skip("mode bits are a no-op on Windows")
		}
		persistPath := newPersistPath(t)
		store := NewOAuthStore(persistPath, nil)
		store.PutRefreshTokenWithTTL("rt1", obj("userId", "testuser"), 60000*time.Millisecond)
		info, err := os.Stat(persistPath)
		expectNoError(t, err)
		expectEqual(t, int(info.Mode().Perm()), 0o600)
	})

	t.Run("load() restores refresh tokens and clients from file", func(t *testing.T) {
		persistPath := newPersistPath(t)
		// Write with store1
		store1 := NewOAuthStore(persistPath, nil)
		store1.PutRefreshTokenWithTTL("rt1", obj("userId", "testuser"), 60000*time.Millisecond)
		store1.PutClient("c1", obj("client_name", "App"))

		// Load with store2
		store2 := NewOAuthStore(persistPath, nil)
		store2.Load()
		expectEqual(t, mustGet(t)(store2.GetRefreshToken("rt1")), obj("userId", "testuser"))
		expectEqual(t, mustGet(t)(store2.GetClient("c1")), obj("client_name", "App"))
	})

	t.Run("load() skips expired refresh tokens", func(t *testing.T) {
		clock := useFakeTimers(t)
		persistPath := newPersistPath(t)
		store1 := NewOAuthStore(persistPath, nil)
		store1.PutRefreshTokenWithTTL("rt-expired", obj("userId", "x"), 100*time.Millisecond)
		store1.PutRefreshTokenWithTTL("rt-valid", obj("userId", "y"), 60000*time.Millisecond)

		clock.advance(200 * time.Millisecond) // rt-expired is now past its expiresAt

		store2 := NewOAuthStore(persistPath, nil)
		store2.Load()
		mustMiss(t)(store2.GetRefreshToken("rt-expired"))
		expectEqual(t, mustGet(t)(store2.GetRefreshToken("rt-valid")), obj("userId", "y"))
	})

	t.Run("load() does not error when file does not exist", func(t *testing.T) {
		store := NewOAuthStore(filepath.Join(t.TempDir(), "nonexistent.json"), nil)
		store.Load()
		expectEqual(t, store.Stats().RefreshTokens, 0)
	})

	t.Run("load() does not error on invalid JSON", func(t *testing.T) {
		badPath := filepath.Join(t.TempDir(), "bad.json")
		expectNoError(t, os.WriteFile(badPath, []byte("not json!"), 0o600))
		store := NewOAuthStore(badPath, nil)
		store.Load()
		expectEqual(t, store.Stats().RefreshTokens, 0)
	})

	t.Run("putRefreshToken() auto-saves", func(t *testing.T) {
		persistPath := newPersistPath(t)
		store := NewOAuthStore(persistPath, nil)
		store.PutRefreshTokenWithTTL("rt1", obj("x", 1), 60000*time.Millisecond)
		expectTrue(t, exists(persistPath), "state file missing")
	})

	t.Run("putClient() auto-saves", func(t *testing.T) {
		persistPath := newPersistPath(t)
		store := NewOAuthStore(persistPath, nil)
		store.PutClient("c1", obj("y", 2))
		expectTrue(t, exists(persistPath), "state file missing")
	})

	t.Run("deleteRefreshToken() auto-saves", func(t *testing.T) {
		persistPath := newPersistPath(t)
		store := NewOAuthStore(persistPath, nil)
		store.PutRefreshTokenWithTTL("rt1", obj("x", 1), 60000*time.Millisecond)
		store.DeleteRefreshToken("rt1")

		data := readState(t, persistPath)
		expectTrue(t, !objAt(t, data, "refreshTokens").Has("rt1"), "rt1 still persisted")
	})

	t.Run("no-op when persistPath is null", func(t *testing.T) {
		store := NewOAuthStore("", nil)
		store.PutRefreshToken("rt1", obj("x", 1))
		store.PutClient("c1", obj("y", 2))
		store.Load()
		// No file should be created anywhere
	})

	// Node 版が書いた状態ファイル（同じ書式）をそのまま読めること。
	// これが読めないと、稼働中のコネクタ（DCR クライアント・refresh token）が全部再認可になる。
	t.Run("load() reads a state file written by the Node implementation", func(t *testing.T) {
		clock := useFakeTimers(t)
		persistPath := newPersistPath(t)
		future := clock.now.Add(24 * time.Hour).UnixMilli()
		nodeState := `{
  "refreshTokens": {
    "rt-node": {
      "value": {
        "clientId": "client-node",
        "scope": "gkill:read",
        "gkillSessionId": "sess-node",
        "userId": "testuser"
      },
      "expiresAt": ` + jsonobj.MarshalString(future) + `
    }
  },
  "clients": {
    "client-node": {
      "client_id": "client-node",
      "client_name": "Example Connector",
      "redirect_uris": [
        "https://example.test/callback"
      ],
      "token_endpoint_auth_method": "none"
    }
  }
}`
		expectNoError(t, os.WriteFile(persistPath, []byte(nodeState), 0o600))
		store := NewOAuthStore(persistPath, nil)
		store.Load()
		expectEqual(t, mustGet(t)(store.GetRefreshToken("rt-node")), obj("clientId", "client-node", "scope", "gkill:read", "gkillSessionId", "sess-node", "userId", "testuser"))
		expectEqual(t, objAt(t, mustGet(t)(store.GetClient("client-node"))).Value("client_name"), "Example Connector")
		// 書き戻しても同じ形（キーの順序と expiresAt のミリ秒）になる
		store.PutClient("client-node", mustGet(t)(store.GetClient("client-node")))
		expectEqual(t, readState(t, persistPath), parseObj(t, nodeState))
	})
}
