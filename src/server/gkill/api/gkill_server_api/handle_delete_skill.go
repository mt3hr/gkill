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

// HandleDeleteSkill はスキルを丸ごと、またはスキル内の1ファイルを消します。
//
// POST /api/delete_skill（wrapAuth）
// req_res.DeleteSkillRequest / req_res.DeleteSkillResponse
//
// path を省くとスキルを丸ごと消します（画面の削除ボタン）。path を指定するとそのファイルだけを消し、
// revision を渡せば一致したときだけ消します。SKILL.md は単独では消せません（ERR000438）。
// 履歴を持たないので、消したものは戻せません（ADR-0634）。そのため MCP の削除ツールは
// 実装だけして公開していません（src/server/gkill/mcp/skill_delete_tool.go）。
func (g *GkillServerAPI) HandleDeleteSkill(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.DeleteSkillRequest{}
	response := &req_res.DeleteSkillResponse{}

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
			err = fmt.Errorf("error at parse delete skill response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error at parse delete skill response to json", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.DeleteSkillError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DELETE_SKILL_MESSAGE"}),
				Cause:        err,
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse delete skill request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error at parse delete skill request to json", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidDeleteSkillRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DELETE_SKILL_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := AuthFromContext(r.Context()).UserID
	store := g.GkillDAOManager.SkillStore
	if request.Path == "" {
		err = store.DeleteSkill(r.Context(), userID, request.Name)
	} else {
		err = store.DeleteFile(r.Context(), userID, request.Name, request.Path, request.Revision)
	}
	if err != nil {
		err = fmt.Errorf("error at delete skill user id = %s name = %q path = %q: %w", userID, request.Name, request.Path, err)
		slog.Log(r.Context(), gkill_log.Debug, "error at delete skill", "error", fmt.Sprintf("%q", err))
		response.Errors = append(response.Errors, skillStoreGkillError(err, request.LocaleName, message.DeleteSkillError, "FAILED_DELETE_SKILL_MESSAGE"))
		return
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.DeleteSkillSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_DELETE_SKILL_MESSAGE"}),
	})
}
