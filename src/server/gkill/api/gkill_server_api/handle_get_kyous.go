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

// HandleGetKyous は、検索条件に一致するKyou一覧を返します。
//
// POST /api/get_kyous（wrapAuthRepos）
// req_res.GetKyousRequest / req_res.GetKyousResponse
//
// query.OnlyLatestDataはユースケース側で常にtrueへ上書きされるので、
// リクエストでfalseを指定しても最新版だけが返ります。
// 検索対象repは認証ミドルウェアが用意したRepositoriesではなく、
// UserIDとDeviceからFindFilterが取り直します。
func (g *GkillServerAPI) HandleGetKyous(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetKyousRequest{}
	response := &req_res.GetKyousResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get kyous response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetKyousResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get kyous request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetKyousRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device

	kyous, warningMessages, gkillErrors, err := g.UsecaseCtx.GetKyous(r.Context(), userID, device, request.LocaleName, request.Query)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		// 失敗したのにGkillErrorが1つも無いことがある(repのSQLエラーなど)。
		// そのまま返すと errors:null + 0件 になり、呼び出し側からは
		// 「成功・該当0件」と区別が付かない。理由は message.EnsureNotEmpty のコメント
		gkillErrors = message.EnsureNotEmpty(gkillErrors, message.FindKyousError,
			api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}))
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}
	if len(gkillErrors) > 0 {
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}

	response.Kyous = kyous
	// プラグイン検索失敗などの警告(検索自体は成功)
	response.Messages = append(response.Messages, warningMessages...)
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetKyousSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_KYOUS_MESSAGE"}),
	})
}
