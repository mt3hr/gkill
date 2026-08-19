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

// HandleGetGitCommitLog は、指定IDのGitCommitLogを取得します。
//
// POST /api/get_git_commit_log（wrapAuthRepos）
// req_res.GetGitCommitLogRequest / req_res.GetGitCommitLogResponse
//
// レスポンスのフィールド名に反して履歴は返さず、最新版1件だけを詰めて返します
// （request.UpdateTime は参照しません）。他のGet系と違い、対象が見つからない場合は
// 空配列ではなくエラーを返します。
func (g *GkillServerAPI) HandleGetGitCommitLog(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetGitCommitLogRequest{}
	response := &req_res.GetGitCommitLogResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get gitCommitLog response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetGitCommitLogResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_GIT_COMMIT_LOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get gitCommitLog request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetGitCommitLogRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_GIT_COMMIT_LOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device
	repositories := auth.Repositories

	gitCommitLogHistories, gkillErrors, err := g.UsecaseCtx.GetGitCommitLog(r.Context(), repositories, userID, device, request.LocaleName, request.ID)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		// 失敗したのにGkillErrorが1つも無いことがある(repのSQLエラーなど)。
		// そのまま返すと errors:null + 0件 になり、呼び出し側からは
		// 「成功・該当0件」と区別が付かない。理由は message.EnsureNotEmpty のコメント
		gkillErrors = message.EnsureNotEmpty(gkillErrors, message.GetGitCommitLogError,
			api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_GIT_COMMIT_LOG_MESSAGE"}))
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}
	if len(gkillErrors) > 0 {
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}

	response.GitCommitLogHistories = gitCommitLogHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetGitCommitLogSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_GIT_COMMIT_LOG_MESSAGE"}),
	})
}
