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

// HandleGetNotificationsByTargetID は対象Kyouに設定されている通知を新しい順に返します。
//
// POST /api/get_gkill_notifications_by_id（wrapAuthRepos）
// req_res.GetNotificationsByTargetIDRequest / req_res.GetNotificationsByTargetIDResponse
//
// 全repを横断し、通知IDごとに最新版だけを残したうえで削除済み（IsDeleted）を除きます。
// 通知済みかどうかでは絞らないので、IsNotificatedがtrueの通知も含まれます。
func (g *GkillServerAPI) HandleGetNotificationsByTargetID(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetNotificationsByTargetIDRequest{}
	response := &req_res.GetNotificationsByTargetIDResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get notifications by target id response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetNotificationsByTargetIDResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_NOTIFICATION_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get notifications by target id request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetNotificationsByTargetIDRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ミドルウェアで設定された認証情報を取得
	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device
	repositories := auth.Repositories

	notifications, gkillErrors, err := g.UsecaseCtx.GetNotificationsByTargetID(r.Context(), repositories, userID, device, request.LocaleName, request.TargetID)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		// 失敗したのにGkillErrorが1つも無いことがある(repのSQLエラーなど)。
		// そのまま返すと errors:null + 0件 になり、呼び出し側からは
		// 「成功・該当0件」と区別が付かない。理由は message.EnsureNotEmpty のコメント
		gkillErrors = message.EnsureNotEmpty(gkillErrors, message.GetNotificationsByTargetIDError,
			api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_NOTIFICATION_MESSAGE"}))
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}
	if len(gkillErrors) > 0 {
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}

	response.Notifications = notifications
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetNotificationsByTargetIDSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_NOTIFICATION_MESSAGE"}),
	})
}
