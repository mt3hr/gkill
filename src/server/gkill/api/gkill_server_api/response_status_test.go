package gkill_server_api

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// gkillのJSON APIは2026-08まで、異常時も必ずHTTP 200を返していた。
// エラーはレスポンスボディの errors 配列にだけ入るので、ステータスしか見ない層
// (監視・プロキシ・アクセスログ・素朴なHTTPクライアント)からは全部成功に見えていた。
//
// ここはエラーコードごとのステータス割り当て(message.HTTPStatusOf)が
// 実際のHTTP応答に出ていることを、エンドツーエンドで確認する。
// 表が正しくてもハンドラが writeErrorStatus を呼んでいなければ意味がないので、
// ソース走査(response_status_guard_test.go)とは別に実物を叩いておく。

const testPasswordSha256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// postRawJSON は生のJSON文字列をそのままPOSTする。
// 壊れたJSONを送りたいので、postJSON(構造体をMarshalする)とは別に用意している。
func postRawJSON(t *testing.T, url string, raw string) *http.Response {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(raw))
	if err != nil {
		t.Fatalf("POST %s failed: %v", url, err)
	}
	return resp
}

// decodeErrors はレスポンスボディから errors 配列だけを取り出す。
func decodeErrors(t *testing.T, resp *http.Response) []*message.GkillError {
	t.Helper()
	var body struct {
		Errors []*message.GkillError `json:"errors"`
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode body %q: %v", string(raw), err)
	}
	return body.Errors
}

// リポジトリ取得に失敗した認証済みAPIは、500とERR000018を返すだけでなく
// Errorログを残すこと。Debugへ戻ると通常の運用設定では原因が記録されない。
func TestResponseStatus_AuthRepositoryFailureReturns500AndLogsError(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, ts.URL, gkillAPI, "admin", testPasswordSha256)
	if err := gkillAPI.GkillDAOManager.ConfigDAOs.RepositoryDAO.Close(context.Background()); err != nil {
		t.Fatalf("RepositoryDAO.Close failed: %v", err)
	}

	var logBuffer bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logBuffer, &slog.HandlerOptions{Level: gkill_log.TraceSQL})))
	t.Cleanup(func() { slog.SetDefault(originalLogger) })

	resp := postJSON(t, ts.URL+"/api/get_kyous", &req_res.GetKyousRequest{
		SessionID:  sessionID,
		LocaleName: "en",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}
	errs := decodeErrors(t, resp)
	if len(errs) != 1 || errs[0].ErrorCode != message.RepositoriesGetError {
		t.Fatalf("errors = %v, want [%s]", formatErrorCodes(errs), message.RepositoriesGetError)
	}

	logged := logBuffer.String()
	if !strings.Contains(logged, `"level":"ERROR"`) ||
		!strings.Contains(logged, "in auth middleware") {
		t.Errorf("認証ミドルウェアのリポジトリ取得失敗がErrorログに無い: %s", logged)
	}
}

// formatErrorCodes はエラーコードだけを並べる(失敗時のメッセージ用)。
func formatErrorCodes(errs []*message.GkillError) []string {
	codes := []string{}
	for _, e := range errs {
		if e != nil {
			codes = append(codes, e.ErrorCode)
		}
	}
	return codes
}

// assertStatusAndCode はステータスとエラーコードの両方を確認する。
//
// **両方を見るのが大事。** ステータスだけ見ると「別の理由で失敗していた」のを取り違えるし、
// エラーコードだけ見ると元の「全部200」に戻ったことに気付けない。
func assertStatusAndCode(t *testing.T, resp *http.Response, wantStatus int, wantCode string) {
	t.Helper()
	errs := decodeErrors(t, resp)
	if resp.StatusCode != wantStatus {
		t.Errorf("status = %d, want %d (errors=%v)", resp.StatusCode, wantStatus, formatErrorCodes(errs))
	}
	if wantCode == "" {
		return
	}
	for _, e := range errs {
		if e != nil && e.ErrorCode == wantCode {
			return
		}
	}
	t.Errorf("errors に %s が無い: %v", wantCode, formatErrorCodes(errs))
}

// TestResponseStatus_Unauthorized はセッションが無効なら401になることを確認する。
//
// 一番効くのがここ。以前はセッション切れも200だったので、
// 期限切れのまま動き続けているのか正常なのかを外から区別できなかった。
func TestResponseStatus_Unauthorized(t *testing.T) {
	ts, _, cleanup := setupTestRouter(t)
	defer cleanup()

	cases := []struct {
		name string
		body string
	}{
		{"session_idが空", `{"session_id":"","locale_name":"en"}`},
		{"session_idが不正", `{"session_id":"not_a_real_session","locale_name":"en"}`},
		{"ボディが壊れている", `{`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp := postRawJSON(t, ts.URL+"/api/get_kyous", c.body)
			defer resp.Body.Close()
			assertStatusAndCode(t, resp, http.StatusUnauthorized, message.AccountSessionNotFoundError)
		})
	}
}

// TestResponseStatus_LoginFailureIsUnauthorized はログイン失敗が401になることを確認する。
//
// 存在しないユーザとパスワード誤りは、利用者列挙を防ぐために同じコード・同じ文言に
// 統一されている(指摘 S3-login)。ステータスも当然同じでなければならない。
func TestResponseStatus_LoginFailureIsUnauthorized(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	prepareLoginReadyAccount(t, gkillAPI, "admin", testPasswordSha256)

	cases := []struct {
		name string
		req  *req_res.LoginRequest
	}{
		{"パスワード誤り", &req_res.LoginRequest{UserID: "admin", PasswordSha256: strings.Repeat("0", 64), LocaleName: "en"}},
		{"存在しないユーザ", &req_res.LoginRequest{UserID: "no_such_user", PasswordSha256: testPasswordSha256, LocaleName: "en"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp := postJSON(t, ts.URL+"/api/login", c.req)
			defer resp.Body.Close()
			assertStatusAndCode(t, resp, http.StatusUnauthorized, message.AccountInvalidPasswordError)
		})
	}
}

// TestResponseStatus_Forbidden は管理者権限が要る操作を非管理者が叩くと403になることを確認する。
func TestResponseStatus_Forbidden(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	adminSession := loginAndGetSession(t, ts.URL, gkillAPI, "admin", testPasswordSha256)

	addResp := postJSON(t, ts.URL+"/api/add_user", &req_res.AddAccountRequest{
		SessionID:    adminSession,
		AccountInfo:  &req_res.Account{UserID: "testuser_status", IsAdmin: false, IsEnable: true},
		DoInitialize: false,
		LocaleName:   "en",
	})
	addResp.Body.Close()
	prepareLoginReadyAccount(t, gkillAPI, "testuser_status", testPasswordSha256)

	normalSession := loginAndGetSession(t, ts.URL, gkillAPI, "testuser_status", testPasswordSha256)

	resp := postJSON(t, ts.URL+"/api/add_user", &req_res.AddAccountRequest{
		SessionID:    normalSession,
		AccountInfo:  &req_res.Account{UserID: "testuser_status2", IsAdmin: false, IsEnable: true},
		DoInitialize: false,
		LocaleName:   "en",
	})
	defer resp.Body.Close()
	assertStatusAndCode(t, resp, http.StatusForbidden, message.AccountNotHasAdminError)
}

// TestResponseStatus_BadRequest はリクエストのパースに失敗したら400になることを確認する。
//
// 認証が要るエンドポイントは、ミドルウェアがボディを先読みするので壊れたJSONは401になる
// (TestResponseStatus_Unauthorized で確認済み)。400になるのは、
// セッションは読めるが型が合わない場合と、認証が要らないエンドポイントの場合。
func TestResponseStatus_BadRequest(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	t.Run("認証不要のエンドポイントで壊れたJSON", func(t *testing.T) {
		resp := postRawJSON(t, ts.URL+"/api/login", `{"user_id":`)
		defer resp.Body.Close()
		assertStatusAndCode(t, resp, http.StatusBadRequest, message.AccountInvalidLoginRequestDataError)
	})

	t.Run("セッションは読めるが型が合わない", func(t *testing.T) {
		sessionID := loginAndGetSession(t, ts.URL, gkillAPI, "admin", testPasswordSha256)
		// kmemo は構造体なのに数値を送る。session_id は読めるので認証は通り、
		// ハンドラ側の Decode で落ちる。
		raw := `{"session_id":"` + sessionID + `","locale_name":"en","kmemo":123}`
		resp := postRawJSON(t, ts.URL+"/api/add_kmemo", raw)
		defer resp.Body.Close()
		assertStatusAndCode(t, resp, http.StatusBadRequest, message.AccountInvalidAddKmemoRequestDataError)
	})
}

// TestResponseStatus_Conflict は既にあるものを二重に作ろうとしたら409になることを確認する。
//
// 衛星リポジトリ(gkill_autocomplete)は「既存タグの追加はエラーではなく成功」と扱うために
// ERR000056 を見ている。ステータスが変わってもエラーコードは変えていないので、
// 本文を読める限りそちらは今までどおり動く。
func TestResponseStatus_Conflict(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	adminSession := loginAndGetSession(t, ts.URL, gkillAPI, "admin", testPasswordSha256)

	// admin は既に居るので、同じ user_id で作ろうとすると AlreadyExistAccountError になる。
	resp := postJSON(t, ts.URL+"/api/add_user", &req_res.AddAccountRequest{
		SessionID:    adminSession,
		AccountInfo:  &req_res.Account{UserID: "admin", IsAdmin: false, IsEnable: true},
		DoInitialize: false,
		LocaleName:   "en",
	})
	defer resp.Body.Close()
	assertStatusAndCode(t, resp, http.StatusConflict, message.AlreadyExistAccountError)
}

// TestResponseStatus_SuccessIsStill200 は成功時が今までどおり 200 で、
// errors / messages が **空配列 []** で返ることを確認する。
//
// 2026-08 のステータス導入時は「ボディを1バイトも変えない」を守り null のままだったが、
// 2026-09-15（ADR-0710）に境界で [] に揃えた。消費者側の `res.errors ?? []` /
// `&& length` のガードはそのまま通る（[] は truthy で長さ 0）。
func TestResponseStatus_SuccessIsStill200(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, ts.URL, gkillAPI, "admin", testPasswordSha256)

	resp := postJSON(t, ts.URL+"/api/get_application_config", &req_res.GetApplicationConfigRequest{
		SessionID: sessionID, LocaleName: "en",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	// 成功時の errors は null ではなく []（message.GkillErrors の MarshalJSON）。
	// null に戻ると、全消費者が「サーバの偶然の実装詳細」を防御する状態へ戻る。
	if got := string(body["errors"]); got != "[]" {
		t.Errorf("成功時の errors = %s, want []", got)
	}
	// messages は成功メッセージが1件以上入る（null ではない）
	if got := string(body["messages"]); got == "null" || got == "" {
		t.Errorf("成功時の messages = %s, want 配列", got)
	}
	if _, ok := body["application_config"]; !ok {
		t.Error("application_config がレスポンスに無い")
	}
}

// TestResponseStatus_PanicReturns500WithGzip は panic 時に
// 500 と、gzipとして復号できるJSON本文が返ることを確認する。
//
// **これは実際に踏んでいた。** gzipMiddleware の defer gzipWriter.Close() が
// panic の巻き戻しで先に走って空のgzipストリームを書き、暗黙200を確定させるので、
// 外側の recoverMiddleware が書く500は捨てられていた。
// 再現テストで「status=200 / 復号すると空の本文」を確認したうえで、
// recoverMiddleware を最内層にも登録して直してある(serve.go)。
func TestResponseStatus_PanicReturns500WithGzip(t *testing.T) {
	g := &GkillServerAPI{}
	router := mux.NewRouter()
	// serve.go と同じ並び(最外層 recover → gzip → 最内層 recover)。
	router.Use(g.recoverMiddleware)
	router.Use(gzipMiddleware())
	router.Use(g.recoverMiddleware)
	router.HandleFunc("/api/panic_for_test", func(w http.ResponseWriter, r *http.Request) {
		panic("panic for TestResponseStatus_PanicReturns500WithGzip")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/panic_for_test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 (gzipのCloseが先に走って200が確定していないか確認)", rec.Code)
	}

	body := rec.Body.Bytes()
	if rec.Header().Get("Content-Encoding") == "gzip" {
		zr, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			t.Fatalf("gzipとして読めない: %v", err)
		}
		decoded, err := io.ReadAll(zr)
		if err != nil {
			t.Fatalf("gzip展開に失敗: %v", err)
		}
		body = decoded
	}

	var parsed struct {
		Errors []*message.GkillError `json:"errors"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("本文がJSONとして読めない %q: %v", string(body), err)
	}
	if len(parsed.Errors) == 0 || parsed.Errors[0].ErrorCode != message.InternalServerPanicError {
		t.Errorf("errors = %v, want %s", formatErrorCodes(parsed.Errors), message.InternalServerPanicError)
	}
}

// decodeRecordedErrors は httptest.NewRecorder の本文から errors 配列を取り出す。
// 本文がJSONとして読めること自体も検査対象(素の WriteHeader だと本文が空になる)。
func decodeRecordedErrors(t *testing.T, rec *httptest.ResponseRecorder) []*message.GkillError {
	t.Helper()
	var parsed struct {
		Errors []*message.GkillError `json:"errors"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("本文がJSONとして読めない %q: %v (素の WriteHeader に戻っていないか確認)", rec.Body.String(), err)
	}
	return parsed.Errors
}

// TestResponseStatus_AuthBodyTooLargeReturns413WithJSONBody は、認証系ミドルウェアの
// ボディ先読みが上限(maxAuthBodyBytes)を超えたとき、413 と JSON の errors 本文の
// **両方**が返ることを確認する。
//
// **以前は w.WriteHeader(413) だけで本文が空だった。** クライアント(gkill-api.ts)は
// ステータスを見ずに res.json() するので、本文が空だとそこで例外になる
// (writeGkillErrorResponse の doc コメント)。auth_middleware.go は handle_*.go では
// ないので response_status_guard_test.go のエンコード行走査には掛からず、
// ここで実物のミドルウェアを叩いて固定する。
func TestResponseStatus_AuthBodyTooLargeReturns413WithJSONBody(t *testing.T) {
	g := &GkillServerAPI{}

	middlewares := []struct {
		name string
		wrap func(http.Handler) http.Handler
	}{
		{"authMiddleware", g.authMiddleware},
		{"authWithReposMiddleware", g.authWithReposMiddleware},
	}

	for _, m := range middlewares {
		t.Run(m.name, func(t *testing.T) {
			handlerCalled := false
			handler := m.wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
			}))

			body := bytes.NewReader(make([]byte, maxAuthBodyBytes+1))
			req := httptest.NewRequest(http.MethodPost, "/api/get_kyous", body)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if handlerCalled {
				t.Error("上限超過なのにハンドラ本体が呼ばれた")
			}
			if rec.Code != http.StatusRequestEntityTooLarge {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
			}
			errs := decodeRecordedErrors(t, rec)
			if len(errs) == 0 || errs[0].ErrorCode != message.RequestBodyTooLargeError {
				t.Errorf("errors = %v, want %s", formatErrorCodes(errs), message.RequestBodyTooLargeError)
			}
			if len(errs) > 0 && errs[0].ErrorMessage == "" {
				t.Error("ErrorMessage が空。利用者向けの文言(REQUEST_BODY_TOO_LARGE_MESSAGE)を載せること")
			}
		})
	}
}

// errorReadCloser は Read が常に失敗するボディ。認証系ミドルウェアの
// 「上限超過以外の読み取り失敗」経路を再現するために使う。
type errorReadCloser struct{}

func (errorReadCloser) Read([]byte) (int, error) { return 0, errors.New("read error for test") }
func (errorReadCloser) Close() error             { return nil }

// TestResponseStatus_AuthBodyReadFailureReturns500WithJSONBody は、認証系ミドルウェアの
// ボディ読み取りが上限超過以外の理由で失敗したとき、500 と JSON の errors 本文が
// 返ることを確認する。こちらも以前は w.WriteHeader(500) だけで本文が空だった。
func TestResponseStatus_AuthBodyReadFailureReturns500WithJSONBody(t *testing.T) {
	g := &GkillServerAPI{}
	handler := g.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("読み取り失敗なのにハンドラ本体が呼ばれた")
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/get_kyous", errorReadCloser{})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	errs := decodeRecordedErrors(t, rec)
	if len(errs) == 0 || errs[0].ErrorCode != message.ReadRequestBodyError {
		t.Errorf("errors = %v, want %s", formatErrorCodes(errs), message.ReadRequestBodyError)
	}
}
