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

func (g *GkillServerAPI) HandleAddNotification(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddNotificationRequest{}
	response := &req_res.AddNotificationResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add notification response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddNotificationResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add notification request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddNotificationRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ミドルウェアで設定された認証情報を取得
	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device
	repositories := auth.Repositories

	addedNotification, gkillErrors, err := g.UsecaseCtx.AddNotification(r.Context(), repositories, userID, device, request.LocaleName, request.Notification, request.TXID)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AddNotificationError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if len(gkillErrors) > 0 {
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}

	// 通知情報を更新する
	notificator, err := g.GkillDAOManager.GetNotificator(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get notificator: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificatorError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	err = notificator.UpdateNotificationTargets(context.Background())
	if err != nil {
		err = fmt.Errorf("error at update notification targetrs: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificatorError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.AddedNotification = addedNotification
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddNotificationSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_ADD_NOTIFICATION_MESSAGE"}),
	})
}
