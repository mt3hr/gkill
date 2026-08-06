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

// HandleGetMiReKyousByTargetID は対象Kyouをタスク化しているものを新しい順に返します。
//
// POST /api/get_mirekyous_by_target_id（wrapAuthRepos）
// req_res.GetMiReKyousByTargetIDRequest / req_res.GetMiReKyousByTargetIDResponse
//
// 全repを横断し、IDごとに最新版だけを残したうえで削除済み（IsDeleted）を除きます。
// 参照先Kyouが削除済みかどうかは見ません。Kyou削除の連鎖処理から、削除の前後どちらでも
// 呼べるようにするためです。確定済みリポジトリのみが対象で、TX中の未確定分は含みません。
func (g *GkillServerAPI) HandleGetMiReKyousByTargetID(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetMiReKyousByTargetIDRequest{}
	response := &req_res.GetMiReKyousByTargetIDResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get mirekyous by target id response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetMiReKyousByTargetIDResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_MI_REKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get mirekyous by target id request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetMiReKyousByTargetIDRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_MI_REKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ミドルウェアで設定された認証情報を取得
	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device
	repositories := auth.Repositories

	mirekyous, gkillErrors, err := g.UsecaseCtx.GetMiReKyousByTargetID(r.Context(), repositories, userID, device, request.LocaleName, request.TargetID)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiReKyousByTargetIDError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_MI_REKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if len(gkillErrors) > 0 {
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}

	response.MiReKyous = mirekyous
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetMiReKyousByTargetIDSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_MI_REKYOU_MESSAGE"}),
	})
}
