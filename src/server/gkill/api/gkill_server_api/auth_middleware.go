package gkill_server_api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// maxAuthBodyBytes は認証系ミドルウェアが認証前に読むボディの上限。
// 未認証の攻撃者に無制限のメモリを確保させないための上限。大容量が正規に必要な
// アップロード系（/api/upload_files 等）は wrapNoAuth 登録でこの経路を通らないため、
// 別枠の maxUploadBodyBytes を wrapNoAuthCapped で掛ける（2026-08-30 監査 F-002）。
const maxAuthBodyBytes = 32 * 1024 * 1024 // 32MB

// maxUploadBodyBytes はアップロード2経路（/api/upload_files, /api/upload_gps_log_files）の
// ボディ上限。ファイル本体が base64 のデータURIとしてJSONへ入るため、実ファイル合計の
// 約1.33倍を見込んでも実容量で700MB級まで1リクエストで送れる値にしてある。
// これを超えるファイルは rep ディレクトリへの直置き + UpdateCache の取り込みで入れる。
const maxUploadBodyBytes = 1 << 30 // 1GB

// noAuthBodyReadTimeout / uploadBodyReadTimeout は wrapNoAuthCapped が掛ける
// そのリクエストの読み取り期限。サーバ全体の ReadTimeout は大容量応答を切らないよう
// 0（無制限）のままにしてある（serve.go）ので、ヘッダだけ送って本文を極小レートで
// 送り続ける型のスローボディはここで打ち切る。
const noAuthBodyReadTimeout = 5 * time.Minute
const uploadBodyReadTimeout = 30 * time.Minute

// sessionPeek はリクエストボディからSessionIDとLocaleNameだけを読み取るための構造体
type sessionPeek struct {
	SessionID  string `json:"session_id"`
	LocaleName string `json:"locale_name"`
}

// readAuthBody は認証前のボディ読み取りを上限付きで行う。
// 上限超過なら 413、その他の読み取り失敗なら 500 を返し、読めたかどうかを ok で返す。
func readAuthBody(w http.ResponseWriter, r *http.Request, ctx context.Context) ([]byte, bool) {
	return readBodyCapped(w, r, ctx, maxAuthBodyBytes)
}

// readBodyCapped は認証前のボディ読み取りを指定の上限付きで行う本体。
// 上限超過なら 413、その他の読み取り失敗なら 500 を返し、読めたかどうかを ok で返す。
//
// どちらの失敗も writeGkillErrorResponse で JSON の errors 本文ごと返す。素の
// WriteHeader だけだと本文が空になり、ステータスを見ずに res.json() する
// クライアント(gkill-api.ts)側で例外になる(writeGkillErrorResponse の doc コメント)。
// ボディが読めていないので locale_name も分からず、文言は既定言語で返す。
func readBodyCapped(w http.ResponseWriter, r *http.Request, ctx context.Context, limitBytes int64) ([]byte, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, limitBytes)
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			slog.Log(ctx, gkill_log.Debug, "request body too large in auth middleware", "error", fmt.Sprintf("%q", err))
			writeGkillErrorResponse(ctx, w, &message.GkillError{
				ErrorCode:    message.RequestBodyTooLargeError,
				ErrorMessage: localeUnawareLocalizer().MustLocalizeMessage(&i18n.Message{ID: "REQUEST_BODY_TOO_LARGE_MESSAGE"}),
			})
			return nil, false
		}
		slog.Log(ctx, gkill_log.Debug, "error at read request body in auth middleware", "error", fmt.Sprintf("%q", err))
		writeGkillErrorResponse(ctx, w, &message.GkillError{
			ErrorCode:    message.ReadRequestBodyError,
			ErrorMessage: localeUnawareLocalizer().MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return nil, false
	}
	_ = r.Body.Close()
	return rawBody, true
}

// wrapNoAuth wraps handler with filterLocalOnly only
//
// ボディを読む経路には使わないこと。認証ミドルウェアを通らないため readAuthBody の
// 32MB上限が効かず、未認証の無制限ボディがそのままヒープへ載る。ボディ付きの
// 無認証経路は wrapNoAuthCapped を使う（auth_middleware_capped_test.go が機械検査する）。
func (g *GkillServerAPI) wrapNoAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !g.filterLocalOnly(w, r) {
			return
		}
		h(w, r)
	}
}

// wrapNoAuthCapped は wrapNoAuth にボディ上限とリクエスト単位の読み取り期限を付けた版。
// wrapNoAuth のボディ付き経路は認証ミドルウェアを通らず 32MB 上限が効かないため、
// ここで経路別の上限を掛ける（2026-08-30 監査 F-002）。上限超過は 413 の JSON、
// 期限超過は読み取りエラーとして打ち切られる。期限はこのリクエストの読み取りに
// だけ効き、次のリクエストでは net/http がヘッダ読み取り時に張り直す。
func (g *GkillServerAPI) wrapNoAuthCapped(h http.HandlerFunc, limitBytes int64, readTimeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !g.filterLocalOnly(w, r) {
			return
		}
		rc := http.NewResponseController(w)
		if err := rc.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
			// 期限を張れないとスローボディ対策が1段消える(上限だけで守る状態になる)ので Warn。
			// 本番経路のラッパ(responseRecorder / gzipResponseWriter)は Unwrap を実装しており
			// ここへは来ない。来るのは httptest.ResponseRecorder 等、期限を持てない実装だけ。
			slog.Log(r.Context(), gkill_log.Warn, "error at set read deadline for no-auth capped route", "error", fmt.Sprintf("%q", err))
		}
		rawBody, ok := readBodyCapped(w, r, r.Context(), limitBytes)
		if !ok {
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(rawBody))
		h(w, r)
	}
}

// wrapAuth wraps with filterLocalOnly + auth (no repos)
func (g *GkillServerAPI) wrapAuth(h http.HandlerFunc) http.HandlerFunc {
	wrapped := g.authMiddleware(http.HandlerFunc(h))
	return func(w http.ResponseWriter, r *http.Request) {
		if !g.filterLocalOnly(w, r) {
			return
		}
		wrapped.ServeHTTP(w, r)
	}
}

// wrapAuthRepos wraps with filterLocalOnly + auth + repos
func (g *GkillServerAPI) wrapAuthRepos(h http.HandlerFunc) http.HandlerFunc {
	wrapped := g.authWithReposMiddleware(http.HandlerFunc(h))
	return func(w http.ResponseWriter, r *http.Request) {
		if !g.filterLocalOnly(w, r) {
			return
		}
		wrapped.ServeHTTP(w, r)
	}
}

// authMiddleware は認証のみ（リポジトリ取得なし）のミドルウェア
func (g *GkillServerAPI) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// ボディを読み取り（認証前なので上限つき）
		rawBody, ok := readAuthBody(w, r, ctx)
		if !ok {
			return
		}

		// ボディを復元（ハンドラが再度読めるように）
		r.Body = io.NopCloser(bytes.NewReader(rawBody))

		// SessionIDとLocaleNameを抽出
		var peek sessionPeek
		if err := json.Unmarshal(rawBody, &peek); err != nil || peek.SessionID == "" {
			// SessionIDが取得できない場合はエラーレスポンス
			writeGkillErrorResponse(ctx, w, &message.GkillError{
				ErrorCode:    message.AccountSessionNotFoundError,
				ErrorMessage: "session_id is required",
			})
			return
		}

		// アカウント認証
		account, gkillError, err := g.getAccountFromSessionID(ctx, peek.SessionID, peek.LocaleName)
		if err != nil {
			writeGkillErrorResponse(ctx, w, gkillError)
			return
		}

		// デバイス取得
		device, err := g.GetDevice()
		if err != nil {
			err = fmt.Errorf("error at get device name in auth middleware: %w", err)
			slog.Log(ctx, gkill_log.Debug, "error at get device name in auth middleware", "error", fmt.Sprintf("%q", err))
			writeGkillErrorResponse(ctx, w, &message.GkillError{
				ErrorCode:    message.GetDeviceError,
				ErrorMessage: api.GetLocalizer(peek.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
			})
			return
		}

		auth := &AuthContext{
			Account: account,
			UserID:  account.UserID,
			Device:  device,
		}

		r = r.WithContext(contextWithAuth(ctx, auth))
		next.ServeHTTP(w, r)
	})
}

// authWithReposMiddleware は認証＋リポジトリ取得のミドルウェア
func (g *GkillServerAPI) authWithReposMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// ボディを読み取り（認証前なので上限つき）
		rawBody, ok := readAuthBody(w, r, ctx)
		if !ok {
			return
		}

		// ボディを復元
		r.Body = io.NopCloser(bytes.NewReader(rawBody))

		// SessionIDとLocaleNameを抽出
		var peek sessionPeek
		if err := json.Unmarshal(rawBody, &peek); err != nil || peek.SessionID == "" {
			writeGkillErrorResponse(ctx, w, &message.GkillError{
				ErrorCode:    message.AccountSessionNotFoundError,
				ErrorMessage: "session_id is required",
			})
			return
		}

		// アカウント認証
		account, gkillError, err := g.getAccountFromSessionID(ctx, peek.SessionID, peek.LocaleName)
		if err != nil {
			writeGkillErrorResponse(ctx, w, gkillError)
			return
		}

		// デバイス取得
		device, err := g.GetDevice()
		if err != nil {
			err = fmt.Errorf("error at get device name in auth middleware: %w", err)
			slog.Log(ctx, gkill_log.Debug, "error at get device name in auth middleware", "error", fmt.Sprintf("%q", err))
			writeGkillErrorResponse(ctx, w, &message.GkillError{
				ErrorCode:    message.GetDeviceError,
				ErrorMessage: api.GetLocalizer(peek.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
			})
			return
		}

		// リポジトリ取得
		repositories, err := g.GkillDAOManager.GetRepositories(account.UserID, device)
		if err != nil {
			err = fmt.Errorf("error at get repositories user id = %s device = %s in auth middleware: %w", account.UserID, device, err)
			// ここは Debug ではなく Error（gkill_error.log へ振り分けられる）。
			// この失敗はそのユーザの**全API**を500にするのに、Debugだと通常のログレベルでは
			// 1行も残らず、「全部エラーになるが理由がどこにも出ない」状態になる。
			// 実際 2026-08-30 の障害では、--log debug で動いていた回のログが偶然残っていた
			// おかげでしか原因に辿り着けなかった。
			slog.Log(ctx, gkill_log.Error, "error at get repositories in auth middleware", "error", fmt.Sprintf("%q", err))
			writeGkillErrorResponse(ctx, w, &message.GkillError{
				ErrorCode:    message.RepositoriesGetError,
				ErrorMessage: api.GetLocalizer(peek.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
			})
			return
		}

		auth := &AuthContext{
			Account:      account,
			UserID:       account.UserID,
			Device:       device,
			Repositories: repositories,
		}

		r = r.WithContext(contextWithAuth(ctx, auth))
		next.ServeHTTP(w, r)
	})
}
