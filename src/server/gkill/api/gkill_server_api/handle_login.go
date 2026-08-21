package gkill_server_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	accountdao "github.com/mt3hr/gkill/src/server/gkill/dao/account"
	"github.com/mt3hr/gkill/src/server/gkill/dao/account_state"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// dummyPasswordHash は、存在しないユーザでも Argon2id 検証を1回実行して、
// 応答時間を存在するユーザと近づけるための固定ハッシュ（ユーザ列挙のタイミング差を消す）。
// 固定入力なので実運用で生成は失敗しない。失敗しても空文字なら検証が即 false で返るだけ。
var dummyPasswordHash = func() string {
	h, err := accountdao.HashPassword("gkill-dummy-password-for-login-timing-equalization")
	if err != nil {
		return ""
	}
	return h
}()

// performDummyPasswordVerification は存在しないユーザのときに呼ぶ。
// account パッケージ名がハンドラ内のローカル変数 account と衝突するのでここに分ける。
func performDummyPasswordVerification(credential string) {
	_, _ = accountdao.VerifyPassword(dummyPasswordHash, credential)
}

// HandleLogin は、user_idとパスワードのSHA-256を検証してログインセッションを発行します。
//
// POST /api/login（wrapNoAuth）
// req_res.LoginRequest / req_res.LoginResponse
//
// 送信元IP単位のレート制限があり、超過した場合は資格情報を検証せずに弾きます。
// 無効化済みのアカウントと、パスワードリセット中 (PasswordResetToken が非nil) の
// アカウントもログインできません。パスワード未設定のアカウントは不一致として扱います。
// 成功時はSessionIDを返すほか、そのユーザにURLogブックマークレット用
// (ApplicationName = "urlog_bookmarklet") のセッションが無ければ併せて作成します。
// 既にあって期限切れの場合はsession_idを変えずに有効期限だけ延ばすので、
// 発行済みのブックマークレットを貼り替えずに済みます。
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
			slog.Log(r.Context(), gkill_log.Warn, "error", "error", fmt.Sprintf("%q", err))
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
		slog.Log(r.Context(), gkill_log.Warn, "error", "error", fmt.Sprintf("%q", err))
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
		slog.Log(r.Context(), gkill_log.Warn, "error", "error", fmt.Sprintf("%q", err))
		// ユーザ列挙対策: 存在しない/引けないユーザと「パスワード誤り」を
		// 同じ error_code + 文言に統一し、Argon2id もダミーで1回実行して応答時間を近づける。
		performDummyPasswordVerification(request.PasswordSha256)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidPasswordError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INVALID_USER_ID_OR_PASSWORD"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if account == nil {
		err = fmt.Errorf("error at get account user id = %s: account not found", request.UserID)
		slog.Log(r.Context(), gkill_log.Warn, "error", "error", fmt.Sprintf("%q", err))
		performDummyPasswordVerification(request.PasswordSha256)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidPasswordError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INVALID_USER_ID_OR_PASSWORD"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウント有効確認
	if !account.IsEnable {
		err = fmt.Errorf("error at account is not enable = %s", request.UserID)
		slog.Log(r.Context(), gkill_log.Warn, "error", "error", fmt.Sprintf("%q", err))
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
		slog.Log(r.Context(), gkill_log.Warn, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountPasswordResetTokenIsNotNilError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "REQUESTED_PASSWORD_RESET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// パスワード不一致を弾く。
	// パスワードが設定されていないアカウントも不一致として扱う (fail-closed)。
	passwordMatched, err := account.VerifyPassword(request.PasswordSha256)
	if err != nil {
		err = fmt.Errorf("error at verify password user id = %s: %w", request.UserID, err)
		slog.Log(r.Context(), gkill_log.Warn, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidPasswordError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INVALID_USER_ID_OR_PASSWORD"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if !passwordMatched {
		err = fmt.Errorf("error at account invalid password = %s", request.UserID)
		slog.Log(r.Context(), gkill_log.Warn, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidPasswordError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INVALID_USER_ID_OR_PASSWORD"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ログインセッション追加
	remoteHost := extractIP(r.RemoteAddr)
	isLocalAppUser := isLoopbackRemoteAddr(r.RemoteAddr)

	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Warn, "error", "error", fmt.Sprintf("%q", err))
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
			slog.Log(r.Context(), gkill_log.Warn, "error", "error", fmt.Sprintf("%q", err))
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
			slog.Log(r.Context(), gkill_log.Warn, "error", "error", fmt.Sprintf("%q", err))
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
		newSession := &account_state.LoginSession{
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
		ok, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.AddLoginSession(r.Context(), newSession)
		if !ok || err != nil {
			if err != nil {
				err = fmt.Errorf("error add login session = %s: %w", request.UserID, err)
				slog.Log(r.Context(), gkill_log.Warn, "error", "error", fmt.Sprintf("%q", err))
			}
			gkillError := &message.GkillError{
				ErrorCode:    message.AddURLogLoginSessionError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_LOGIN_INTERNAL_SERVER_ERROR_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	} else if time.Now().After(urlogBookmarkletSession.ExpirationTime) {
		// 期限切れの場合は有効期限を更新する（session_idは維持するので既存ブックマークレットがそのまま使える）
		urlogBookmarkletSession.ExpirationTime = time.Now().Add(time.Hour * 24 * 30)
		urlogBookmarkletSession.LoginTime = time.Now()
		ok, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.UpdateLoginSession(r.Context(), urlogBookmarkletSession)
		if !ok || err != nil {
			if err != nil {
				err = fmt.Errorf("error update urlog bookmarklet session = %s: %w", request.UserID, err)
				slog.Log(r.Context(), gkill_log.Warn, "error", "error", fmt.Sprintf("%q", err))
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
