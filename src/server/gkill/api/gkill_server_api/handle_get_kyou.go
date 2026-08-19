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

// HandleGetKyou は、指定IDのKyouを取得します。
//
// POST /api/get_kyou（wrapAuthRepos）
// req_res.GetKyouRequest / req_res.GetKyouResponse
//
// UpdateTimeを指定するとその版1件だけを、指定しなければID一致の全履歴を
// KyouHistoriesに入れて返します。RepNameは履歴取得時のrep絞り込みにだけ効き、
// UpdateTime指定時は最新データを持つrepだけを見るので参照されません。
// 対象が見つからない場合は空配列で、エラーにはしません。
func (g *GkillServerAPI) HandleGetKyou(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetKyouRequest{}
	response := &req_res.GetKyouResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get kyou response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetKyouResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get kyou request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetKyouRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device
	repositories := auth.Repositories

	kyouHistories, gkillErrors, err := g.UsecaseCtx.GetKyouHistories(r.Context(), repositories, userID, device, request.LocaleName, request.ID, request.UpdateTime, request.RepName)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		// 失敗したのにGkillErrorが1つも無いことがある(repのSQLエラーなど)。
		// そのまま返すと errors:null + 0件 になり、呼び出し側からは
		// 「成功・該当0件」と区別が付かない。理由は message.EnsureNotEmpty のコメント
		gkillErrors = message.EnsureNotEmpty(gkillErrors, message.GetKyouError,
			api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}))
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}
	if len(gkillErrors) > 0 {
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}

	response.KyouHistories = kyouHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetKyouSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_KYOU_MESSAGE"}),
	})
}
