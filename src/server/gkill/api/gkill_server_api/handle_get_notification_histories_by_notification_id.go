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

// HandleGetNotificationHistoriesByNotificationID は指定IDの通知の履歴を新しい順に返します。
//
// POST /api/get_gkill_notification_histories_by_notification_id（wrapAuthRepos）
// req_res.GetNotificationHistoryByNotificationIDRequest / req_res.GetNotificationHistoryByNotificationIDResponse
//
// UpdateTimeが非nilならその版だけを1件の配列で返し、nilならID一致の全版を返します。
// タグ・テキストの同種APIと違い、リクエストにRepNameがないのでrepでの絞り込みはできません。
// 削除済み（IsDeleted）の版も履歴として含まれます。
func (g *GkillServerAPI) HandleGetNotificationHistoriesByNotificationID(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetNotificationHistoryByNotificationIDRequest{}
	response := &req_res.GetNotificationHistoryByNotificationIDResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get notification histories by notification id response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetNotificationHistoriesByNotificationIDResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_NOTIFICATION_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get notification histories by notification id request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetNotificationHistoriesByNotificationIDRequestDataError,
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

	notifications, gkillErrors, err := g.UsecaseCtx.GetNotificationHistoriesByNotificationID(r.Context(), repositories, userID, device, request.LocaleName, request.ID, request.UpdateTime)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		// 失敗したのにGkillErrorが1つも無いことがある(repのSQLエラーなど)。
		// そのまま返すと errors:null + 0件 になり、呼び出し側からは
		// 「成功・該当0件」と区別が付かない。理由は message.EnsureNotEmpty のコメント
		gkillErrors = message.EnsureNotEmpty(gkillErrors, message.GetNotificationHistoriesByNotificationIDError,
			api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_NOTIFICATION_MESSAGE"}))
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}
	if len(gkillErrors) > 0 {
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}

	response.NotificationHistories = notifications
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetNotificationHistoriesByNotificationIDSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_NOTIFICATION_MESSAGE"}),
	})
}
