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
	"github.com/mt3hr/gkill/src/server/gkill/dao/account"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// HandleSetNewPassword は、リセットトークンを検証して新しいパスワードを設定します。
//
// POST /api/set_new_password（wrapNoAuth）
// req_res.SetNewPasswordRequest / req_res.SetNewPasswordResponse
//
// セッション不要で叩けるエンドポイントなので、送信元IP単位のレート制限で
// リセットトークンの総当たりを抑えます。
// NewPasswordSha256 は64桁hexでなければ受け付けません。
// トークンの照合はconstant-timeで行い、期限切れも弾きます。
// 設定に成功したらそのユーザの既存ログインセッションを全て失効させますが、
// 失効に失敗してもパスワード自体は変わっているので、ログだけ残して成功として返します。
func (g *GkillServerAPI) HandleSetNewPassword(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.SetNewPasswordRequest{}
	response := &req_res.SetNewPasswordResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse set new password response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidSetNewPasswordResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SET_NEW_PASSWORD_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse login response to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidSetNewPasswordResponseDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SET_NEW_PASSWORD_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// このエンドポイントは未認証で叩ける。リセットトークンを総当たりされないように
	// ログインと同じくIP単位で試行回数を絞る
	if !g.passwordResetRateLimiter.allow(extractIP(r.RemoteAddr)) {
		gkillError := &message.GkillError{
			ErrorCode:    message.LoginRateLimitError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "LOGIN_RATE_LIMITED_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// クライアントはパスワードのSHA-256を64桁hexにして送ってくる。
	// その形式でないものは受け付けない (空文字や巨大な文字列がそのまま資格情報になるのを防ぐ)
	if !account.IsValidCredentialFormat(request.NewPasswordSha256) {
		err = fmt.Errorf("error at invalid new password format user id = %s", request.UserID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidSetNewPasswordRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SET_NEW_PASSWORD_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象のアカウント情報を取得してパスワード設定
	targetAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), request.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", request.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SET_NEW_PASSWORD_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if targetAccount == nil {
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SET_NEW_PASSWORD_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// リセットトークンがあっているか確認する。
	// トークン自体が秘密なので、照合はconstant-timeで行い、期限も見る
	if !targetAccount.IsPasswordResetTokenValid(request.ResetToken, time.Now()) {
		err = fmt.Errorf("error at reset token is not match or expired user id = %s", request.UserID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidPasswordResetTokenError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SET_NEW_PASSWORD_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	passwordHash, err := account.HashPassword(request.NewPasswordSha256)
	if err != nil {
		err = fmt.Errorf("error at hash new password user id = %s: %w", request.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInfoUpdateError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SET_NEW_PASSWORD_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	updateTargetAccount := &account.Account{
		UserID:                       targetAccount.UserID,
		IsAdmin:                      targetAccount.IsAdmin,
		IsEnable:                     targetAccount.IsEnable,
		PasswordHash:                 &passwordHash,
		PasswordResetToken:           nil,
		PasswordResetTokenExpiration: nil,
	}
	ok, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.UpdateAccount(r.Context(), updateTargetAccount)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at update account user id = %s: %w", request.UserID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInfoUpdateError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SET_NEW_PASSWORD_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// パスワードを設定しなおしたので、それまでのセッションは失効させる。
	// 失敗してもパスワード自体は変わっているので、ログだけ残して続行する
	_, err = g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.DeleteLoginSessionsByUserID(r.Context(), targetAccount.UserID)
	if err != nil {
		err = fmt.Errorf("error at delete login sessions user id = %s: %w", targetAccount.UserID, err)
		slog.Log(r.Context(), gkill_log.Warn, "error", "error", fmt.Sprintf("%q", err))
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.SetNewPasswordSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_SET_NEW_PASSWORD_MESSAGE"}),
	})
}
