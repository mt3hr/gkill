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

// HandleGetTextHistoriesByTextID は指定IDのテキストの履歴を新しい順に返します。
//
// POST /api/get_text_histories_by_text_id（wrapAuthRepos）
// req_res.GetTextHistoryByTextIDRequest / req_res.GetTextHistoryByTextIDResponse
//
// UpdateTimeが非nilならその版だけを1件の配列で返し、nilならID一致の全版を返します。
// RepNameでのrep絞り込みが効くのは全版取得の側だけで、UpdateTime指定時は無視されます。
// 削除済み（IsDeleted）の版も履歴として含まれます。
func (g *GkillServerAPI) HandleGetTextHistoriesByTextID(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetTextHistoryByTextIDRequest{}
	response := &req_res.GetTextHistoryByTextIDResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get text histories by text id response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetTextHistoriesByTextIDResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TEXTS_BY_TARGET_ID_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get text histories by text id request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetTextHistoriesByTextIDRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TEXTS_BY_TARGET_ID_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ミドルウェアで設定された認証情報を取得
	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device
	repositories := auth.Repositories

	texts, gkillErrors, err := g.UsecaseCtx.GetTextHistoriesByTextID(r.Context(), repositories, userID, device, request.LocaleName, request.ID, request.UpdateTime, request.RepName)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextHistoriesByTextIDError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TEXTS_BY_TARGET_ID_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if len(gkillErrors) > 0 {
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}

	response.TextHistories = texts
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetTextHistoriesByTextIDSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_TEXTS_BY_TARGET_ID_MESSAGE"}),
	})
}
