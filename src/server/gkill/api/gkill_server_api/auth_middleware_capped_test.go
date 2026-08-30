package gkill_server_api

// wrapNoAuthCapped の境界動作（上限±1バイト・413 の JSON 本文・読み取り期限）と、
// serve.go のボディ付き wrapNoAuth 経路がすべて capped 版で登録されていることのソース走査。
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
	"os"
	"regexp"
	"strings"
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

// serve.go の登録をソース走査で固定する。素の wrapNoAuth に残ってよいのは
// ボディを読まない経路だけ。ボディ付きの経路を wrapNoAuth で足すと、
// 未認証の無制限ボディがそのままヒープへ載る（目の前ではエラーにならない）。
func TestServeSourceNoAuthBodyRoutesAreCapped(t *testing.T) {
	src, err := os.ReadFile("serve.go")
	if err != nil {
		t.Fatalf("read serve.go: %v", err)
	}
	text := string(src)

	allowedBareNoAuth := map[string]bool{
		// GET でボディを読まない配信系だけ
		"HandleFileServe":            true,
		"HandleZipCacheFileServe":    true,
		"HandleURLogBookmarkletPage": true,
	}
	re := regexp.MustCompile(`g\.wrapNoAuth\(g\.(\w+)\)`)
	for _, m := range re.FindAllStringSubmatch(text, -1) {
		if !allowedBareNoAuth[m[1]] {
			t.Errorf("%s が素の wrapNoAuth で登録されている。ボディを読む経路は wrapNoAuthCapped を使うこと（2026-08-30 監査 F-002）", m[1])
		}
	}

	// アップロード2経路は大容量側の上限・期限で登録する
	for _, want := range []string{
		"g.wrapNoAuthCapped(g.HandleUploadFiles, maxUploadBodyBytes, uploadBodyReadTimeout)",
		"g.wrapNoAuthCapped(g.HandleUploadGPSLogFiles, maxUploadBodyBytes, uploadBodyReadTimeout)",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("serve.go に %q が見当たらない", want)
		}
	}

	// 残りのボディ付き無認証経路は 32MB の上限で登録する
	for _, handlerName := range []string{
		"HandleLogin", "HandleLogout", "HandleResetPassword", "HandleSetNewPassword",
		"HandleGetSharedKyous", "HandleURLogBookmarkletAddress",
		"HandleGetKyousMCP", "HandleGetRepInfosMCP",
		"HandleBrowseZipContents", "HandleGetIDFKyouByRelativePath",
	} {
		want := fmt.Sprintf("g.wrapNoAuthCapped(g.%s, maxAuthBodyBytes, noAuthBodyReadTimeout)", handlerName)
		if !strings.Contains(text, want) {
			t.Errorf("serve.go に %q が見当たらない", want)
		}
	}
}
