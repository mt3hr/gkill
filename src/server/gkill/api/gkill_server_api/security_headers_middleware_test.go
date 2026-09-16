package gkill_server_api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// securityHeadersMiddleware は Serve() のルータにしか掛かっておらず、ハンドラのテストハーネス
// （setupTestRouter）は registerAPIRoutes だけを登録するので、これまでどのテストも通っていなかった。
// clickjacking / MIME スニッフィング / リファラ漏れの defense-in-depth なので、
// 「3つ付く」「経路側が付け直した値をミドルウェアの既定が潰さない」を直接固定する。
func TestSecurityHeadersMiddleware(t *testing.T) {
	t.Run("何も付いていない応答には3つとも付く", func(t *testing.T) {
		handler := securityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/get_kyous", nil))

		want := map[string]string{
			"X-Content-Type-Options": "nosniff",
			"X-Frame-Options":        "SAMEORIGIN",
			"Referrer-Policy":        "strict-origin-when-cross-origin",
		}
		for name, value := range want {
			if got := recorder.Header().Get(name); got != value {
				t.Errorf("%s = %q, want %q", name, got, value)
			}
		}
	})

	// serve.go は router.Use(securityHeadersMiddleware) で**ミドルウェアが先**に既定値を置き、
	// 利用者ファイル配信（withUserContentSecurityHeaders）はルート側で**後から** Set する。
	// ミドルウェアが next の後で上書きする実装に変わると、経路側の判断（サムネイルの
	// CSP sandbox 等）が黙って消えるので、「経路側の Set が最終値になる」ことを固定する。
	t.Run("経路側で付け直した値が最終値になる", func(t *testing.T) {
		handler := securityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if got := w.Header().Get("X-Frame-Options"); got != "SAMEORIGIN" {
				t.Errorf("ハンドラ到達時の X-Frame-Options = %q, want SAMEORIGIN（ミドルウェアが先に既定を置く）", got)
			}
			w.Header().Set("X-Frame-Options", "DENY")
			w.WriteHeader(http.StatusOK)
		}))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/files/a.png", nil))

		if got := recorder.Header().Get("X-Frame-Options"); got != "DENY" {
			t.Errorf("X-Frame-Options = %q, want DENY（経路側の値がミドルウェアに潰されている）", got)
		}
		// 経路側が触らなかった残り2つは既定のまま
		if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
		}
		if got := recorder.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
			t.Errorf("Referrer-Policy = %q, want strict-origin-when-cross-origin", got)
		}
	})

	// 外側のミドルウェアや逆プロキシ設定が先に付けた値は既定で潰さない（`if h.Get(...) == ""` の意味）
	t.Run("先に付いている値は既定で上書きしない", func(t *testing.T) {
		handler := securityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		recorder := httptest.NewRecorder()
		recorder.Header().Set("Referrer-Policy", "no-referrer")
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/get_kyous", nil))

		if got := recorder.Header().Get("Referrer-Policy"); got != "no-referrer" {
			t.Errorf("Referrer-Policy = %q, want no-referrer", got)
		}
	})
}
