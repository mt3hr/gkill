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

// HandleAddMiReKyou は既存Kyouをタスク化したもの（MiReKyou。TargetID＋Miの予定項目）を1件追加します。
//
// POST /api/add_mirekyou（wrapAuthRepos）
// req_res.AddMiReKyouRequest / req_res.AddMiReKyouResponse
//
// TXIDがnilなら書き込み用リポジトリへ確定登録し、非nilならそのトランザクションの
// 一時リポジトリへ積む（commit_txするまで検索には出ない）。
// 同じIDのMiReKyouが既にある場合は追加せず、AlreadyExistMiReKyouErrorをerrorsに積んで返す。
// MiReKyouは後から追加されたrep種別なので、既存の設定DBには書き込み用repが無いことがある。
// その場合（TXIDがnilかつWriteMiReKyouRepがnil）はAddMiReKyouErrorを返す。
// WantResponseKyouがtrueのときだけ、登録後にリポジトリから読み直してAddedMiReKyouとAddedKyouを返す
// （TXID指定時は一時リポジトリにしか無いためどちらもnilになる）。
func (g *GkillServerAPI) HandleAddMiReKyou(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddMiReKyouRequest{}
	response := &req_res.AddMiReKyouResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add mirekyou response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddMiReKyouResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_REKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add mirekyou request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddMiReKyouRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_REKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ミドルウェアで設定された認証情報を取得
	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device
	repositories := auth.Repositories

	gkillErrors, err := g.UsecaseCtx.AddMiReKyou(r.Context(), repositories, userID, device, request.LocaleName, request.MiReKyou, request.TXID)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.AddMiReKyouError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_REKYOU_MESSAGE"}),
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
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_REKYOU_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.AddedMiReKyou = mirekyou

		kyou, err := repositories.MiReKyouReps.GetKyou(r.Context(), request.MiReKyou.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get mirekyou user id = %s device = %s id = %s: %w", userID, device, request.MiReKyou.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.GetMiReKyouError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_REKYOU_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.AddedKyou = kyou
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddMiReKyouSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_ADD_MI_REKYOU_MESSAGE"}),
	})
}
