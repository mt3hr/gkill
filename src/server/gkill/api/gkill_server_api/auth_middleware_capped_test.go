package gkill_server_api

// wrapNoAuthCapped の境界動作（上限±1バイト・413 の JSON 本文・読み取り期限）と、
// ルート表（apiRoutes）のボディ付き無認証経路がすべて上限つきで登録されていることの検査。
//
// 2026-08-30 監査 F-002: wrapNoAuth の経路は認証ミドルウェアを通らないため
// readAuthBody の 32MB 上限が効かず、未認証の無制限ボディがそのままヒープへ載っていた。

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
)

func TestWrapNoAuthCapped_OverLimitReturns413JSON(t *testing.T) {
	ts, gkillAPI, optCleanup := setupTestRouter(t)
	defer func() { ts.Close(); optCleanup() }()

	called := false
	h := gkillAPI.wrapNoAuthCapped(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}, 128, time.Minute)
	capped := httptest.NewServer(h)
	defer capped.Close()

	res, err := http.Post(capped.URL, "application/json", bytes.NewReader(make([]byte, 129)))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", res.StatusCode)
	}
	// 本文は JSON の errors で返る（ステータスを見ずに res.json() するクライアントのため）
	var body struct {
		Errors []*message.GkillError `json:"errors"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("413 の本文が JSON で読めない: %v", err)
	}
	if len(body.Errors) == 0 || body.Errors[0].ErrorCode != message.RequestBodyTooLargeError {
		t.Errorf("errors = %+v, want error_code %s", body.Errors, message.RequestBodyTooLargeError)
	}
	if called {
		t.Error("上限超過なのにハンドラが呼ばれた")
	}
}

func TestWrapNoAuthCapped_AtLimitPassesBodyIntact(t *testing.T) {
	ts, gkillAPI, optCleanup := setupTestRouter(t)
	defer func() { ts.Close(); optCleanup() }()

	want := bytes.Repeat([]byte("a"), 128)
	var got []byte
	h := gkillAPI.wrapNoAuthCapped(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("ハンドラ側でボディが読めない: %v", err)
		}
		got = b
		w.WriteHeader(http.StatusOK)
	}, 128, time.Minute)
	capped := httptest.NewServer(h)
	defer capped.Close()

	res, err := http.Post(capped.URL, "application/json", bytes.NewReader(want))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200（上限ちょうどは通る）", res.StatusCode)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("ハンドラへ渡ったボディが元と違う: len=%d want %d", len(got), len(want))
	}
}

func TestWrapNoAuthCapped_SlowBodyIsCutByReadDeadline(t *testing.T) {
	ts, gkillAPI, optCleanup := setupTestRouter(t)
	defer func() { ts.Close(); optCleanup() }()

	called := false
	h := gkillAPI.wrapNoAuthCapped(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}, 1<<20, 300*time.Millisecond)
	capped := httptest.NewServer(h)
	defer capped.Close()

	conn, err := net.Dial("tcp", capped.Listener.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// ヘッダだけ送り、本文を出し惜しみして読み取り期限を越えさせる
	fmt.Fprintf(conn, "POST / HTTP/1.1\r\nHost: gkill-test\r\nContent-Type: application/json\r\nContent-Length: 1000\r\n\r\n")
	if _, err := conn.Write([]byte("{")); err != nil {
		t.Fatalf("write first byte: %v", err)
	}
	time.Sleep(900 * time.Millisecond)
	// 期限後に残りを送っても、読み取りは既に打ち切られている。
	// サーバが接続ごと閉じていれば write / read はエラーになり、それも合格。
	_, _ = conn.Write(bytes.Repeat([]byte(" "), 999))
	res, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err == nil {
		defer func() { _ = res.Body.Close() }()
		if res.StatusCode < 400 {
			t.Errorf("status = %d, want エラー応答（読み取り期限で打ち切られるはず）", res.StatusCode)
		}
	}
	if called {
		t.Error("期限切れなのにハンドラが呼ばれた")
	}
}

// 本番の応答経路は accessLog の responseRecorder と gzip の gzipResponseWriter が
// ResponseWriter を包む。どちらかが Unwrap を欠くと http.ResponseController が
// SetReadDeadline を底の *http.response まで辿れず、wrapNoAuthCapped の
// スローボディ打ち切りが本番でだけ静かに無効になる（素の httptest では検出できない）。
func TestResponseWriterWrappersUnwrapForResponseController(t *testing.T) {
	deadlineErr := make(chan error, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := newResponseRecorder(w)
		wrapped := &gzipResponseWriter{ResponseWriter: rec}
		rc := http.NewResponseController(wrapped)
		deadlineErr <- rc.SetReadDeadline(time.Now().Add(time.Minute))
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	_ = res.Body.Close()
	if err := <-deadlineErr; err != nil {
		t.Errorf("SetReadDeadline がラッパ越しに効かない（Unwrap の欠落）: %v", err)
	}
}

// ルート表（apiRoutes）の無認証経路がすべてボディ上限つきで登録されていることを固定する。
// 素の wrapNoAuth（bodyNone）に残ってよいのはボディを読まない経路だけ。
// ボディ付きの経路を bodyNone で足すと、未認証の無制限ボディがそのままヒープへ載る
// （目の前ではエラーにならない）。validateAPIRoutes が起動時にも同じ検査をするが、
// アップロード2本が大容量側の上限であること・それ以外が 32MB 側であることは表の意図なので
// ここで名指しで固定する。
func TestAPIRoutesNoAuthBodyRoutesAreCapped(t *testing.T) {
	gkillAPI := &GkillServerAPI{}
	routes := gkillAPI.apiRoutes()

	// GET でボディを読まない経路だけが bodyNone を許される
	allowedBareNoAuth := map[string]bool{
		"/api/urlog_bookmarklet_page": true,
	}
	// アップロード2経路は大容量側の上限・期限で登録する
	uploadPaths := map[string]bool{
		"/api/upload_files":        true,
		"/api/upload_gpslog_files": true,
	}

	seen := map[string]bool{}
	for _, rt := range routes {
		if rt.Auth != authNone {
			continue
		}
		seen[rt.Path] = true
		switch {
		case allowedBareNoAuth[rt.Path]:
			if rt.Body != bodyNone || rt.Method != http.MethodGet {
				t.Errorf("%s はボディを読まない GET 配信なので bodyNone のまま: body=%d method=%s", rt.Path, rt.Body, rt.Method)
			}
		case uploadPaths[rt.Path]:
			if rt.Body != bodyUpload {
				t.Errorf("%s はアップロード経路なので bodyUpload（maxUploadBodyBytes / uploadBodyReadTimeout）で登録すること", rt.Path)
			}
		default:
			if rt.Body != bodyAuth {
				t.Errorf("%s が素の wrapNoAuth 相当で登録されている。ボディを読む無認証経路は bodyAuth を使うこと（2026-08-30 監査 F-002）", rt.Path)
			}
		}
	}
	for path := range allowedBareNoAuth {
		if !seen[path] {
			t.Errorf("免除リストの %s が表に無い。リストから消すこと", path)
		}
	}
	for path := range uploadPaths {
		if !seen[path] {
			t.Errorf("アップロード経路 %s が表に無い。リストから消すこと", path)
		}
	}
}
