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

// HandleUpdateIDFKyou はファイルのKyouを更新します。書き換わるのはDB上のレコードだけで、
// ファイル実体には触れません。
//
// POST /api/update_idf_kyou（wrapAuthRepos）
// req_res.UpdateIDFKyouRequest / req_res.UpdateIDFKyouResponse
//
// 更新は追記です（同一IDに新しいUPDATE_TIME版を足す）。TXIDが非nilならtempリポジトリに積むだけで、
// commit_txするまで実リポジトリには反映されません。対象IDが存在しない場合はerrorsに載せて返します。
// WantResponseKyouがtrueのときだけIDFKyouとKyouを読み直して返します
// （読み直し先は実リポジトリなので、TXID指定時はtempに積んだ内容が載りません）。
func (g *GkillServerAPI) HandleUpdateIDFKyou(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateIDFKyouRequest{}
	response := &req_res.UpdateIDFKyouResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update idfKyou response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateIDFKyouResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update idfKyou request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateIDFKyouRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ミドルウェアで設定された認証情報を取得
	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device
	repositories := auth.Repositories

	gkillErrors, err := g.UsecaseCtx.UpdateIDFKyou(r.Context(), repositories, userID, device, request.LocaleName, request.IDFKyou, request.TXID)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.UpdateIDFKyouError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if len(gkillErrors) > 0 {
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}

	if request.WantResponseKyou {
		idfKyou, err := repositories.IDFKyouReps.GetIDFKyou(r.Context(), request.IDFKyou.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get idfKyou user id = %s device = %s id = %s: %w", userID, device, request.IDFKyou.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.GetIDFKyouError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_UPDATED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.UpdatedIDFKyou = idfKyou
		kyou, err := repositories.IDFKyouReps.GetKyou(r.Context(), request.IDFKyou.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get idfKyou user id = %s device = %s id = %s: %w", userID, device, request.IDFKyou.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.GetIDFKyouError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.UpdatedKyou = kyou
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateIDFKyouSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_IDFKYOU_MESSAGE"}),
	})
}
