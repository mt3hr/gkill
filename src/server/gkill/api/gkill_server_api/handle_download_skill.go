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

// HandleDownloadSkill はスキルを zip にして base64 で返します（画面のダウンロード）。
//
// POST /api/download_skill（wrapAuth）
// req_res.DownloadSkillRequest / req_res.DownloadSkillResponse
//
// zip の中身はスキル名のフォルダ1段で包んであり、/api/upload_skill はこの1段を剥がすので、
// ダウンロードした zip を手元で直してそのまま上げ直せます。生のバイナリではなく JSON の中の base64 で
// 返すのは、クライアントが JSON 以外の応答を受け付けず、セッションも本文の JSON で渡すためです。
func (g *GkillServerAPI) HandleDownloadSkill(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.DownloadSkillRequest{}
	response := &req_res.DownloadSkillResponse{}

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
			err = fmt.Errorf("error at parse download skill response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error at parse download skill response to json", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.DownloadSkillError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DOWNLOAD_SKILL_MESSAGE"}),
				Cause:        err,
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse download skill request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error at parse download skill request to json", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidDownloadSkillRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DOWNLOAD_SKILL_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := AuthFromContext(r.Context()).UserID
	fileName, data, err := g.GkillDAOManager.SkillStore.BuildZip(userID, request.Name)
	if err != nil {
		err = fmt.Errorf("error at build skill zip user id = %s name = %q: %w", userID, request.Name, err)
		slog.Log(r.Context(), gkill_log.Debug, "error at build skill zip", "error", fmt.Sprintf("%q", err))
		response.Errors = append(response.Errors, skillStoreGkillError(err, request.LocaleName, message.DownloadSkillError, "FAILED_DOWNLOAD_SKILL_MESSAGE"))
		return
	}
	response.FileName = fileName
	response.ZipBase64 = base64.StdEncoding.EncodeToString(data)
}
