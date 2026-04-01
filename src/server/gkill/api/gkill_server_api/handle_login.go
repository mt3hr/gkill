package gkill_server_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/account_state"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func (g *GkillServerAPI) HandleLogin(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.LoginRequest{}
	response := &req_res.LoginResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Warn, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse login response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Warn, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidLoginResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_LOGIN_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse login request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Warn, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidLoginRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_LOGIN_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	ip := extractIP(r.RemoteAddr)
	if !g.loginRateLimiter.allow(ip) {
		gkillError := &message.GkillError{
			ErrorCode:    message.LoginRateLimitError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "LOGIN_RATE_LIMITED_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 存在するアカウントを取得
	account, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), request.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", request.UserID, err)
		slog.Log(r.Context(), gkill_log.Warn, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_LOGIN_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if account == nil {
		err = fmt.Errorf("error at get account user id = %s: account not found", request.UserID)
		slog.Log(r.Context(), gkill_log.Warn, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INVALID_USER_ID_OR_PASSWORD"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウント有効確認
	if !account.IsEnable {
		err = fmt.Errorf("error at account is not enable = %s", request.UserID)
		slog.Log(r.Context(), gkill_log.Warn, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountIsNotEnableError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "ACCOUNT_DISABLED_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// パスワードリセット処理実施中のアカウントはログインから弾く
	if account.PasswordResetToken != nil {
		err = fmt.Errorf("error at password reset token is not nil = %s", request.UserID)
		slog.Log(r.Context(), gkill_log.Warn, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountPasswordResetTokenIsNotNilError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "REQUESTED_PASSWORD_RESET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// パスワード不一致を弾く
	if account.PasswordSha256 != nil && *account.PasswordSha256 != request.PasswordSha256 {
		err = fmt.Errorf("error at account invalid password = %s", request.UserID)
		slog.Log(r.Context(), gkill_log.Warn, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidPasswordError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_LOGIN_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ログインセッション追加
	isLocalAppUser := false
	spl := strings.Split(r.RemoteAddr, ":")
	remoteHost := strings.Join(spl[:len(spl)-1], ":")
	switch remoteHost {
	case "localhost":
		fallthrough
	case "127.0.0.1":
		fallthrough
	case "[::1]":
		fallthrough
	case "::1":
		isLocalAppUser = true
	}

	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Warn, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	loginSession := &account_state.LoginSession{
		ID:              GenerateNewID(),
		UserID:          request.UserID,
		Device:          device,
		ApplicationName: "gkill",
		SessionID:       GenerateNewID(),
		ClientIPAddress: remoteHost,
		LoginTime:       time.Now(),
		ExpirationTime:  time.Now().Add(time.Hour * 24 * 30), // 1ヶ月
		IsLocalAppUser:  isLocalAppUser,
	}
	ok, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.AddLoginSession(r.Context(), loginSession)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error add login session user_id = %s: %w", request.UserID, err)
			slog.Log(r.Context(), gkill_log.Warn, "error", "error", err)
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountLoginInternalServerError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_LOGIN_INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// URLogブックマークレット用のセッションがもしなければ作成する
	loginSessions, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.GetAllLoginSessions(r.Context())
	if err != nil {
		if err != nil {
			err = fmt.Errorf("error get login sessions = %s: %w", request.UserID, err)
			slog.Log(r.Context(), gkill_log.Warn, "error", "error", err)
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.GetAccountSessionsError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_LOGIN_INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	var urlogBookmarkletSession *account_state.LoginSession
	for _, loginSession := range loginSessions {
		if loginSession.ApplicationName == "urlog_bookmarklet" && loginSession.UserID == request.UserID {
			urlogBookmarkletSession = loginSession
			break
		}
	}
	if urlogBookmarkletSession == nil {
		loginSession := &account_state.LoginSession{
			ID:              GenerateNewID(),
			UserID:          request.UserID,
			Device:          device,
			ApplicationName: "urlog_bookmarklet",
			SessionID:       GenerateNewID(),
			ClientIPAddress: remoteHost,
			LoginTime:       time.Now(),
			ExpirationTime:  time.Now().Add(time.Hour * 24 * 30), // 1ヶ月
			IsLocalAppUser:  isLocalAppUser,
		}
		ok, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.AddLoginSession(r.Context(), loginSession)
		if !ok || err != nil {
			if err != nil {
				err = fmt.Errorf("error add login session = %s: %w", request.UserID, err)
				slog.Log(r.Context(), gkill_log.Warn, "error", "error", err)
			}
			gkillError := &message.GkillError{
				ErrorCode:    message.AddURLogLoginSessionError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_LOGIN_INTERNAL_SERVER_ERROR_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	response.SessionID = loginSession.SessionID
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.LoginSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_LOGIN_MESSAGE"}),
	})
}
