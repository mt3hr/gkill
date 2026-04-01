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
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func (g *GkillServerAPI) HandleAddAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddAccountRequest{}
	response := &req_res.AddAccountResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add account response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidAddAccountResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_ACCOUNT_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add account request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidAddAccountRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_ACCOUNT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device

	// 管理者権限がなければ弾く
	if !auth.Account.IsAdmin {
		err = fmt.Errorf("account not has admin user id = %s", userID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotHasAdminError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_ACCOUNT_NO_AUTH_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), request.AccountInfo.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user device = %s id = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetAccountError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_ACCOUNT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existAccount != nil {
		err = fmt.Errorf("exist account id = %s", userID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AlreadyExistAccountError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_ACCOUNT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウント情報を追加
	defaultApplicationConfig := user_config.GetDefaultApplicationConfig(request.AccountInfo.UserID, device)
	_, err = g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.AddApplicationConfig(r.Context(), defaultApplicationConfig)
	if err != nil {
		err = fmt.Errorf("error at add application config user id = %s id = %s: %w", userID, request.AccountInfo.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AddApplicationConfig,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_ACCOUNT_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	passwordResetToken := GenerateNewID()
	account := &account.Account{
		UserID:             request.AccountInfo.UserID,
		IsAdmin:            request.AccountInfo.IsAdmin,
		IsEnable:           request.AccountInfo.IsEnable,
		PasswordResetToken: &passwordResetToken,
	}
	_, err = g.GkillDAOManager.ConfigDAOs.AccountDAO.AddAccount(r.Context(), account)
	if err != nil {
		err = fmt.Errorf("error at add account user id = %s id = %s: %w", userID, request.AccountInfo.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AddApplicationConfig,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_ACCOUNT_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	requesterAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), request.AccountInfo.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s id = %s: %w", userID, request.AccountInfo.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetAccountError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_ACCOUNT_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if requesterAccount == nil {
		gkillError := &message.GkillError{
			ErrorCode:    message.GetAccountError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_ACCOUNT_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.DoInitialize {
		err := g.initializeNewUserReps(r.Context(), requesterAccount)
		if err != nil {
			err = fmt.Errorf("error at initialize new user reps user id = %s device = %s account = %#v: %w", userID, device, request.AccountInfo, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddAccountError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_ACCOUNT_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	response.AddedAccountInfo = requesterAccount
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddAccountSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_ADD_ACCOUNT_MESSAGE"}),
	})
}
