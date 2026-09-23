package gkill_server_api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/skills"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// HandleUploadSkill は zip の中身でスキルを丸ごと置き換えます（新規なら作ります）。画面のアップロードです。
//
// POST /api/upload_skill（wrapNoAuth）
// req_res.UploadSkillRequest / req_res.UploadSkillResponse
//
// wrapNoAuth 登録（本文の上限がアップロード用の枠になる）ですが、ハンドラ内で SessionID から
// アカウントを解決するので未認証では使えません。保存量に上限を設けない方針（ADR-0634）なので、
// 認証付き経路の 32MB 枠ではなく /api/upload_files と同じ枠に載せています。
//
// dry_run が true なら書き込まずに「追加・削除・変更・無視されるファイル」を返します（画面の確認の1段目）。
// false なら置き換えます（DELETE_WRITE）。スキル名は zip の中の SKILL.md の frontmatter の name で決まり、
// 中身がフォルダ1段で包まれていれば剥がします。置き換えは一時ディレクトリへ展開してから入れ替えるので、
// 途中で失敗しても既存のスキルは元のまま残ります。アップロードの衝突（別の書き手の変更を上書きする）は
// 検出しません（利用者単位で管理しているため。ADR-0634）。
func (g *GkillServerAPI) HandleUploadSkill(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UploadSkillRequest{}
	response := &req_res.UploadSkillResponse{}

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
			err = fmt.Errorf("error at parse upload skill response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error at parse upload skill response to json", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.UploadSkillError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_SKILL_MESSAGE"}),
				Cause:        err,
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse upload skill request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error at parse upload skill request to json", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUploadSkillRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_SKILL_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}
	userID := account.UserID

	zipBytes, err := decodeUploadedBase64(request.ZipBase64)
	if err != nil {
		err = fmt.Errorf("error at decode uploaded skill zip user id = %s: %w", userID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error at decode uploaded skill zip", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUploadSkillRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_SKILL_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	store := g.GkillDAOManager.SkillStore
	var plan *skills.ReplacePlan
	if request.DryRun {
		plan, err = store.PlanReplace(userID, zipBytes)
	} else {
		plan, err = store.Replace(r.Context(), userID, zipBytes)
	}
	if err != nil {
		err = fmt.Errorf("error at upload skill user id = %s dry run = %t: %w", userID, request.DryRun, err)
		slog.Log(r.Context(), gkill_log.Debug, "error at upload skill", "error", fmt.Sprintf("%q", err))
		response.Errors = append(response.Errors, skillStoreGkillError(err, request.LocaleName, message.UploadSkillError, "FAILED_UPLOAD_SKILL_MESSAGE"))
		return
	}
	response.Plan = &req_res.SkillReplacePlan{
		Name:    plan.Name,
		IsNew:   plan.IsNew,
		Added:   plan.Added,
		Removed: plan.Removed,
		Changed: plan.Changed,
		Ignored: plan.Ignored,
	}
	if !request.DryRun {
		response.Applied = true
		response.Messages = append(response.Messages, &message.GkillMessage{
			MessageCode: message.UploadSkillSuccessMessage,
			Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPLOAD_SKILL_MESSAGE"}),
		})
	}
}

// decodeUploadedBase64 は base64（data URI の接頭辞付きでもよい）を復号する。
// 画面は FileReader.readAsDataURL の結果をそのまま送るので、"," より前を捨てる（handle_upload_files.go と同じ）。
func decodeUploadedBase64(value string) ([]byte, error) {
	parts := strings.SplitN(value, ",", 2)
	encoded := strings.TrimSpace(parts[len(parts)-1])
	if encoded == "" {
		return nil, errors.New("zip_base64 is empty")
	}
	return base64.StdEncoding.DecodeString(encoded)
}
