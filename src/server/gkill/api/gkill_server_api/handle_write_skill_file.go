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

// HandleWriteSkillFile はスキル内の1ファイル（テキスト）を書きます。MCP の gkill_add_skill / gkill_update_skill が使います。
//
// POST /api/write_skill_file（wrapAuth）
// req_res.WriteSkillFileRequest / req_res.WriteSkillFileResponse
//
// revision を省くと新規作成だけを許し、既にあれば 409（ERR000436）です。revision を渡すと、
// 今の中身の revision と一致したときだけ上書きし、食い違えば 409（ERR000437）で今の revision を文言に添えます。
// スキルが無いときは SKILL.md の新規作成だけがスキルを作れます。SKILL.md は frontmatter
// （name がスキル名と一致・description が空でない）を検査します。反映はすぐで、履歴は残りません（ADR-0634）。
// 画面からは呼びません（画面は zip の丸ごと置き換えだけ）。
func (g *GkillServerAPI) HandleWriteSkillFile(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.WriteSkillFileRequest{}
	response := &req_res.WriteSkillFileResponse{}

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
			err = fmt.Errorf("error at parse write skill file response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error at parse write skill file response to json", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.WriteSkillFileError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_WRITE_SKILL_FILE_MESSAGE"}),
				Cause:        err,
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse write skill file request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error at parse write skill file request to json", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidWriteSkillFileRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_WRITE_SKILL_FILE_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := AuthFromContext(r.Context()).UserID
	revision, err := g.GkillDAOManager.SkillStore.WriteFile(r.Context(), userID, request.Name, request.Path, []byte(request.Content), request.Revision)
	if err != nil {
		err = fmt.Errorf("error at write skill file user id = %s name = %q path = %q: %w", userID, request.Name, request.Path, err)
		slog.Log(r.Context(), gkill_log.Debug, "error at write skill file", "error", fmt.Sprintf("%q", err))
		response.Errors = append(response.Errors, skillStoreGkillError(err, request.LocaleName, message.WriteSkillFileError, "FAILED_WRITE_SKILL_FILE_MESSAGE"))
		return
	}
	response.Path = request.Path
	response.Revision = revision
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.WriteSkillFileSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_WRITE_SKILL_FILE_MESSAGE"}),
	})
}
