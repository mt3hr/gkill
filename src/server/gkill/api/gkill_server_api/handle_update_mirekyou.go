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

// HandleUpdateMiReKyou は既存Kyouをタスク化したMiReKyouを更新します。
// 書き換わるのはタスク化した側の予定情報だけで、対象のKyouには触れません。
//
// POST /api/update_mirekyou（wrapAuthRepos）
// req_res.UpdateMiReKyouRequest / req_res.UpdateMiReKyouResponse
//
// 更新は追記です（同一IDに新しいUPDATE_TIME版を足す）。TXIDが非nilならtempリポジトリに積むだけで、
// commit_txするまで実リポジトリには反映されません。対象IDが存在しない場合はerrorsに載せて返します。
// 他の型と違い書き込み先repのnil検査があり、MiReKyouリポジトリが設定に無い環境では
// TXIDなしの更新がerrorsになります。
// WantResponseKyouがtrueのときだけMiReKyouとKyouを読み直して返します
// （読み直し先は実リポジトリなので、TXID指定時はtempに積んだ内容が載りません）。
func (g *GkillServerAPI) HandleUpdateMiReKyou(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateMiReKyouRequest{}
	response := &req_res.UpdateMiReKyouResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update mirekyou response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateMiReKyouResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_REKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update mirekyou request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateMiReKyouRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_REKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ミドルウェアで設定された認証情報を取得
	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device
	repositories := auth.Repositories

	gkillErrors, err := g.UsecaseCtx.UpdateMiReKyou(r.Context(), repositories, userID, device, request.LocaleName, request.MiReKyou, request.TXID)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.UpdateMiReKyouError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_REKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if len(gkillErrors) > 0 {
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}

	if request.WantResponseKyou {
		mirekyou, err := repositories.MiReKyouReps.GetMiReKyou(r.Context(), request.MiReKyou.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get mirekyou user id = %s device = %s id = %s: %w", userID, device, request.MiReKyou.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.GetMiReKyouError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_REKYOU_UPDATED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.UpdatedMiReKyou = mirekyou

		kyou, err := repositories.MiReKyouReps.GetKyou(r.Context(), request.MiReKyou.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get mirekyou user id = %s device = %s id = %s: %w", userID, device, request.MiReKyou.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.GetMiReKyouError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_REKYOU_UPDATED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.UpdatedKyou = kyou
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateMiReKyouSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_MI_REKYOU_MESSAGE"}),
	})
}
