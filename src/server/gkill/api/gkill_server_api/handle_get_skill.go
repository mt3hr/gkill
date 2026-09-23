package gkill_server_api

import (
	"context"
	"encoding/base64"
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

// HandleGetSkill はスキルの中身を返します。
//
// POST /api/get_skill（wrapAuth）
// req_res.GetSkillRequest / req_res.GetSkillResponse
//
// path を省くと SKILL.md の全文（frontmatter を含む）とファイル一覧を skill に、
// path を指定するとそのファイルの中身を file に入れて返します。テキスト（UTF-8 で NUL を含まない。
// 拡張子ではなく中身で判定）は content、バイナリは content_base64 です。
// max_bytes が正でファイルがそれより大きいときは中身を返さず content_omitted を立てます
// （保存量に上限は無いが、MCP が AI へ渡す量だけを抑えるため。ADR-0634）。
// 大文字小文字だけ違うパスやシンボリックリンクは辿りません。
func (g *GkillServerAPI) HandleGetSkill(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetSkillRequest{}
	response := &req_res.GetSkillResponse{}

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
			err = fmt.Errorf("error at parse get skill response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error at parse get skill response to json", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.GetSkillError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SKILL_MESSAGE"}),
				Cause:        err,
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get skill request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error at parse get skill request to json", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetSkillRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SKILL_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := AuthFromContext(r.Context()).UserID
	store := g.GkillDAOManager.SkillStore

	if request.Path == "" {
		skill, err := store.Get(userID, request.Name)
		if err != nil {
			err = fmt.Errorf("error at get skill user id = %s name = %q: %w", userID, request.Name, err)
			slog.Log(r.Context(), gkill_log.Debug, "error at get skill", "error", fmt.Sprintf("%q", err))
			response.Errors = append(response.Errors, skillStoreGkillError(err, request.LocaleName, message.GetSkillError, "FAILED_GET_SKILL_MESSAGE"))
			return
		}
		detail := &req_res.SkillDetail{
			Name:          skill.Name,
			Description:   skill.Description,
			InvalidReason: skill.InvalidReason,
			UpdatedTime:   skill.UpdatedTime,
			Content:       string(skill.Manifest),
			Revision:      skill.ManifestRevision,
			Files:         []*req_res.SkillFileInfo{},
		}
		for _, file := range skill.Files {
			detail.Files = append(detail.Files, toSkillFileInfo(file))
		}
		response.Skill = detail
		return
	}

	maxBytes := max(request.MaxBytes, 0)
	content, err := store.ReadFile(userID, request.Name, request.Path, maxBytes)
	if err != nil {
		err = fmt.Errorf("error at read skill file user id = %s name = %q path = %q: %w", userID, request.Name, request.Path, err)
		slog.Log(r.Context(), gkill_log.Debug, "error at read skill file", "error", fmt.Sprintf("%q", err))
		response.Errors = append(response.Errors, skillStoreGkillError(err, request.LocaleName, message.GetSkillError, "FAILED_GET_SKILL_MESSAGE"))
		return
	}
	file := &req_res.SkillFileContent{
		SkillFileInfo:  *toSkillFileInfo(&content.FileInfo),
		ContentOmitted: content.Omitted,
	}
	if !content.Omitted {
		if content.IsText {
			file.Content = string(content.Content)
		} else {
			file.ContentBase64 = base64.StdEncoding.EncodeToString(content.Content)
		}
	}
	response.File = file
}
