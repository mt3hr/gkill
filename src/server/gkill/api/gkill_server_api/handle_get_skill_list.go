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

// HandleGetSkillList はログイン中の利用者のスキル一覧を返します。
//
// POST /api/get_skill_list（wrapAuth）
// req_res.GetSkillListRequest / req_res.GetSkillListResponse
//
// スキルは $GKILL_HOME/skills/<user_id>/<name>/ に置いた SKILL.md と付属ファイルです（ADR-0634）。
// 画面の一覧と、MCP の gkill_get_skill_list / gkill_status が使います。
// SKILL.md が無い・frontmatter が壊れているスキルも invalid_reason 付きで返します
// （一覧から消すと、利用者が壊れていることに気づけず直せないため）。
func (g *GkillServerAPI) HandleGetSkillList(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetSkillListRequest{}
	response := &req_res.GetSkillListResponse{Skills: []*req_res.SkillInfo{}}

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
			err = fmt.Errorf("error at parse get skill list response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error at parse get skill list response to json", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.GetSkillListError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SKILL_LIST_MESSAGE"}),
				Cause:        err,
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get skill list request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error at parse get skill list request to json", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetSkillListRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SKILL_LIST_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := AuthFromContext(r.Context()).UserID
	summaries, err := g.GkillDAOManager.SkillStore.List(userID)
	if err != nil {
		err = fmt.Errorf("error at list skills user id = %s: %w", userID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error at list skills", "error", fmt.Sprintf("%q", err))
		response.Errors = append(response.Errors, skillStoreGkillError(err, request.LocaleName, message.GetSkillListError, "FAILED_GET_SKILL_LIST_MESSAGE"))
		return
	}
	for _, summary := range summaries {
		response.Skills = append(response.Skills, &req_res.SkillInfo{
			Name:          summary.Name,
			Description:   summary.Description,
			UpdatedTime:   summary.UpdatedTime,
			FileCount:     summary.FileCount,
			InvalidReason: summary.InvalidReason,
		})
	}
}
