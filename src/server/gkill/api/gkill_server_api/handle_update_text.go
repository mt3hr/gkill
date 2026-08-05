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

// HandleUpdateText はKyouに付けられたテキストを更新します。
//
// POST /api/update_text（wrapAuthRepos）
// req_res.UpdateTextRequest / req_res.UpdateTextResponse
//
// 更新は追記です（同一IDに新しいUPDATE_TIME版を足す）。TXIDが非nilならtempリポジトリに積むだけで、
// commit_txするまで実リポジトリには反映されません。対象IDが存在しない場合はerrorsに載せて返します。
// UpdatedTextは実リポジトリを読み直した値なので、TXID指定時はtempに積んだ内容が載りません。
func (g *GkillServerAPI) HandleUpdateText(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateTextRequest{}
	response := &req_res.UpdateTextResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update text response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateTextResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TEXT_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update text request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateTextRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TEXT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ミドルウェアで設定された認証情報を取得
	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device
	repositories := auth.Repositories

	updatedText, gkillErrors, err := g.UsecaseCtx.UpdateText(r.Context(), repositories, userID, device, request.LocaleName, request.Text, request.TXID)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.UpdateTextError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TEXT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if len(gkillErrors) > 0 {
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}

	response.UpdatedText = updatedText
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateTextSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_TEXT_MESSAGE"}),
	})
}
