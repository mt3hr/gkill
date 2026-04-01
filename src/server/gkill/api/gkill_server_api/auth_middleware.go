package gkill_server_api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// sessionPeek はリクエストボディからSessionIDとLocaleNameだけを読み取るための構造体
type sessionPeek struct {
	SessionID  string `json:"session_id"`
	LocaleName string `json:"locale_name"`
}

// wrapNoAuth wraps handler with filterLocalOnly only
func (g *GkillServerAPI) wrapNoAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !g.filterLocalOnly(w, r) {
			return
		}
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

		// ボディを読み取り
		rawBody, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Log(ctx, gkill_log.Debug, "error at read request body in auth middleware", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.Body.Close()

		// ボディを復元（ハンドラが再度読めるように）
		r.Body = io.NopCloser(bytes.NewReader(rawBody))

		// SessionIDとLocaleNameを抽出
		var peek sessionPeek
		if err := json.Unmarshal(rawBody, &peek); err != nil || peek.SessionID == "" {
			// SessionIDが取得できない場合はエラーレスポンス
			w.Header().Set("Content-Type", "application/json")
			errResp := struct {
				Errors []*message.GkillError `json:"errors"`
			}{
				Errors: []*message.GkillError{
					{
						ErrorCode:    message.AccountSessionNotFoundError,
						ErrorMessage: "session_id is required",
					},
				},
			}
			json.NewEncoder(w).Encode(errResp)
			return
		}

		// アカウント認証
		account, gkillError, err := g.getAccountFromSessionID(ctx, peek.SessionID, peek.LocaleName)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			errResp := struct {
				Errors []*message.GkillError `json:"errors"`
			}{
				Errors: []*message.GkillError{gkillError},
			}
			json.NewEncoder(w).Encode(errResp)
			return
		}

		// デバイス取得
		device, err := g.GetDevice()
		if err != nil {
			err = fmt.Errorf("error at get device name in auth middleware: %w", err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			w.Header().Set("Content-Type", "application/json")
			errResp := struct {
				Errors []*message.GkillError `json:"errors"`
			}{
				Errors: []*message.GkillError{
					{
						ErrorCode:    message.GetDeviceError,
						ErrorMessage: api.GetLocalizer(peek.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
					},
				},
			}
			json.NewEncoder(w).Encode(errResp)
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

		// ボディを読み取り
		rawBody, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Log(ctx, gkill_log.Debug, "error at read request body in auth middleware", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.Body.Close()

		// ボディを復元
		r.Body = io.NopCloser(bytes.NewReader(rawBody))

		// SessionIDとLocaleNameを抽出
		var peek sessionPeek
		if err := json.Unmarshal(rawBody, &peek); err != nil || peek.SessionID == "" {
			w.Header().Set("Content-Type", "application/json")
			errResp := struct {
				Errors []*message.GkillError `json:"errors"`
			}{
				Errors: []*message.GkillError{
					{
						ErrorCode:    message.AccountSessionNotFoundError,
						ErrorMessage: "session_id is required",
					},
				},
			}
			json.NewEncoder(w).Encode(errResp)
			return
		}

		// アカウント認証
		account, gkillError, err := g.getAccountFromSessionID(ctx, peek.SessionID, peek.LocaleName)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			errResp := struct {
				Errors []*message.GkillError `json:"errors"`
			}{
				Errors: []*message.GkillError{gkillError},
			}
			json.NewEncoder(w).Encode(errResp)
			return
		}

		// デバイス取得
		device, err := g.GetDevice()
		if err != nil {
			err = fmt.Errorf("error at get device name in auth middleware: %w", err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			w.Header().Set("Content-Type", "application/json")
			errResp := struct {
				Errors []*message.GkillError `json:"errors"`
			}{
				Errors: []*message.GkillError{
					{
						ErrorCode:    message.GetDeviceError,
						ErrorMessage: api.GetLocalizer(peek.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
					},
				},
			}
			json.NewEncoder(w).Encode(errResp)
			return
		}

		// リポジトリ取得
		repositories, err := g.GkillDAOManager.GetRepositories(account.UserID, device)
		if err != nil {
			err = fmt.Errorf("error at get repositories user id = %s device = %s in auth middleware: %w", account.UserID, device, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			w.Header().Set("Content-Type", "application/json")
			errResp := struct {
				Errors []*message.GkillError `json:"errors"`
			}{
				Errors: []*message.GkillError{
					{
						ErrorCode:    message.RepositoriesGetError,
						ErrorMessage: api.GetLocalizer(peek.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
					},
				},
			}
			json.NewEncoder(w).Encode(errResp)
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
