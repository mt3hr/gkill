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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
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
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidSetNewPasswordResponseDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SET_NEW_PASSWORD_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象のアカウント情報を取得してパスワード設定
	targetAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), request.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", request.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
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

	// リセットトークンがあっているか確認
	if targetAccount.PasswordResetToken == nil || request.ResetToken != *targetAccount.PasswordResetToken {
		err = fmt.Errorf("error at reset token is not match user id = %s requested token = %s: %w", request.UserID, request.ResetToken, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidPasswordResetTokenError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SET_NEW_PASSWORD_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	updateTargetAccount := &account.Account{
		UserID:             targetAccount.UserID,
		IsAdmin:            targetAccount.IsAdmin,
		IsEnable:           targetAccount.IsEnable,
		PasswordSha256:     &request.NewPasswordSha256,
		PasswordResetToken: nil,
	}
	ok, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.UpdateAccount(r.Context(), updateTargetAccount)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at update account user id = %s: %w", request.UserID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInfoUpdateError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SET_NEW_PASSWORD_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.SetNewPasswordSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_SET_NEW_PASSWORD_MESSAGE"}),
	})
}
