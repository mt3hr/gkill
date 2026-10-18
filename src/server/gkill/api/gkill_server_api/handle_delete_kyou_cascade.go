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

// HandleDeleteKyouCascade は記録1件を、付随データ（タグ・テキスト・通知）と参照元（リポスト・タスク化）ごと論理削除します。
//
// POST /api/delete_kyou_cascade（wrapAuthRepos）
// req_res.DeleteKyouCascadeRequest / req_res.DeleteKyouCascadeResponse
//
// 処理の本体は usecase.DeleteKyouCascade で、Web の画面（削除の確認ダイアログ）と MCP の gkill_delete_kyou が
// この1実装を使います（ADR-0636）。全件を1つの tx で積んで commit_tx と同じ1つの SQLite トランザクションで
// 確定するので、全部消えるか何も消えないかのどちらかです。成功時は消したもの全部を deleted に返します。
// update_user / update_device はセッションから決め、update_app だけを要求から受けます（空なら "gkill"）。
func (g *GkillServerAPI) HandleDeleteKyouCascade(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.DeleteKyouCascadeRequest{}
	response := &req_res.DeleteKyouCascadeResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close request body", "error", fmt.Sprintf("%q", err))
		}
	}()
	defer func() {
		writeErrorStatus(r.Context(), w, response.Errors)
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse delete kyou cascade response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error at parse delete kyou cascade response to json", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidDeleteKyouCascadeResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DELETE_KYOU_NOTHING_DELETED_MESSAGE"}),
				Cause:        err,
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse delete kyou cascade request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error at parse delete kyou cascade request to json", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidDeleteKyouCascadeRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DELETE_KYOU_NOTHING_DELETED_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ミドルウェアで設定された認証情報を取得
	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device
	repositories := auth.Repositories

	deleted, gkillErrors, err := g.UsecaseCtx.DeleteKyouCascade(r.Context(), repositories, userID, device, request.LocaleName, request.TargetID, request.DataType, request.UpdateApp)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error at delete kyou cascade", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.DeleteKyouCascadeError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DELETE_KYOU_NOTHING_DELETED_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if len(gkillErrors) > 0 {
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}

	response.Deleted = make([]*req_res.DeletedRecord, 0, len(deleted))
	for _, record := range deleted {
		response.Deleted = append(response.Deleted, &req_res.DeletedRecord{ID: record.ID, DataType: record.DataType})
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.DeleteKyouCascadeSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_DELETE_KYOU_MESSAGE"}),
	})
}
