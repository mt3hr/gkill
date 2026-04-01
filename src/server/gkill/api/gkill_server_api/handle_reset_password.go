package gkill_server_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/account"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func (g *GkillServerAPI) HandleResetPassword(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.ResetPasswordRequest{}
	response := &req_res.ResetPasswordResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse reset password to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidResetPasswordResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_PASSWORD_RESET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse reset password request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidResetPasswordRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_PASSWORD_RESET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// パスワードリセット操作をしたユーザを特定
	requesterSession, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.GetLoginSession(r.Context(), request.SessionID)
	if requesterSession == nil || err != nil {
		err = fmt.Errorf("error at get login session session id = %s: %w", request.SessionID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountSessionNotFoundError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_PASSWORD_RESET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	requesterAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), requesterSession.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", requesterSession.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_PASSWORD_RESET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if requesterAccount == nil {
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_PASSWORD_RESET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 管理者権限がなければ弾く
	if !requesterAccount.IsAdmin {
		err = fmt.Errorf("account not has admin user id = %s: %w", requesterSession.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotHasAdminError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_PASSWORD_RESET_NO_AUTH_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象のアカウント情報を取得して更新
	targetAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), request.TargetUserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", request.TargetUserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_PASSWORD_RESET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if targetAccount == nil {
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_PASSWORD_RESET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	passwordResetToken := GenerateNewID()
	updateTargetAccount := &account.Account{
		UserID:             targetAccount.UserID,
		IsAdmin:            targetAccount.IsAdmin,
		IsEnable:           targetAccount.IsEnable,
		PasswordSha256:     nil,
		PasswordResetToken: &passwordResetToken,
	}
	ok, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.UpdateAccount(r.Context(), updateTargetAccount)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at update account user id = %s: %w", request.TargetUserID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInfoUpdateError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_PASSWORD_RESET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.PasswordResetSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_PASSWORD_RESET_MESSAGE"}),
	})
	response.PasswordResetPathWithoutHost = *updateTargetAccount.PasswordResetToken
}
