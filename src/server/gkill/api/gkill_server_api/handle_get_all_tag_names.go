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

// HandleGetAllTagNames は、利用者の全レポジトリに存在するタグ名の一覧を返します。
//
// POST /api/get_all_tag_names（wrapAuthRepos）
// req_res.GetAllTagNamesRequest / req_res.GetAllTagNamesResponse
//
// rep跨ぎでタグIDごとの最新版を決めてから名前を集めるので、
// 別repに改名後の版があるタグの旧名は混ざりません。削除済みタグは含めません。
// 同名タグは1つにまとめられ、順序は保証しません。
func (g *GkillServerAPI) HandleGetAllTagNames(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetAllTagNamesRequest{}
	response := &req_res.GetAllTagNamesResponse{}

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
				ErrorCode:    message.InvalidGetAllTagNamesResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_ALL_TAG_NAMES_MESSAGE"}),
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
			ErrorCode:    message.InvalidGetAllTagNamesRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_ALL_TAG_NAMES_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device
	repositories := auth.Repositories

	allTagNames, gkillErrors, err := g.UsecaseCtx.GetAllTagNames(r.Context(), repositories, userID, device, request.LocaleName)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		// 失敗したのにGkillErrorが1つも無いことがある(repのSQLエラーなど)。
		// そのまま返すと errors:null + 0件 になり、呼び出し側からは
		// 「成功・該当0件」と区別が付かない。理由は message.EnsureNotEmpty のコメント
		gkillErrors = message.EnsureNotEmpty(gkillErrors, message.GetAllTagNamesError,
			api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_ALL_TAG_NAMES_MESSAGE"}))
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}
	if len(gkillErrors) > 0 {
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}

	response.TagNames = allTagNames
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetAllTagNamesSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_ALL_TAG_NAMES_MESSAGE"}),
	})
}
