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
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func (g *GkillServerAPI) HandleLogout(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.LogoutRequest{}
	response := &req_res.LogoutResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Warn, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse logout request to json: %w", err)
			slog.Log(r.Context(), gkill_log.Warn, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidLogoutResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_LOGOUT_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse logout request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Warn, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidLogoutRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_LOGOUT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.CloseDatabase {
		account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
		if err != nil {
			if err != nil {
				err = fmt.Errorf("error account from session id = %s: %w", request.SessionID, err)
				slog.Log(r.Context(), gkill_log.Warn, "error", "error", err)
			}
			response.Errors = append(response.Errors, gkillError)
			return
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

		_, err = g.GkillDAOManager.CloseUserRepositories(account.UserID, device)
		if err != nil {
			err = fmt.Errorf("error at close repository user id = %s device = %s: %w", account.UserID, device, err)
			slog.Log(r.Context(), gkill_log.Warn, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetDeviceError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	ok, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.DeleteLoginSession(r.Context(), request.SessionID)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at delete login session id = %s: %w", request.SessionID, err)
			slog.Log(r.Context(), gkill_log.Warn, "error", "error", err)
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountLogoutInternalServerError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_LOGOUT_INTERNAL_SERVER_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.LogoutSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_LOGOUT_MESSAGE"}),
	})
}
