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

// HandleUpdateUserReps は対象利用者のリポジトリ一覧を、送られてきた内容で置き換えます。
//
// POST /api/update_user_reps（wrapAuth）
// req_res.UpdateUserRepsRequest / req_res.UpdateUserRepsResponse
//
// 管理者でなければ弾きます。
// 差分更新ではなく対象利用者のリポジトリを全削除してからUpdatedRepsを書き込むので、
// 送り漏らしたリポジトリは設定から消えます。
// 書き換えるのは設定DBの登録内容だけで、読み込み済みのリポジトリには反映されません
// （反映には /api/reload_repositories が要ります）。
func (g *GkillServerAPI) HandleUpdateUserReps(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateUserRepsRequest{}
	response := &req_res.UpdateUserRepsResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update userReps response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateUserRepsResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REP_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update userReps request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateUserRepsRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REP_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	auth := AuthFromContext(r.Context())
	device := auth.Device

	// 管理者権限がなければ弾く
	if !auth.Account.IsAdmin {
		err = fmt.Errorf("account not has admin user id = %s", auth.Account.UserID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotHasAdminError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REP_NO_AUTH_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象のアカウント情報を取得して更新
	targetAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), request.TargetUserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", request.TargetUserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REP_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if targetAccount == nil {
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REP_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := targetAccount.UserID

	ok, err := g.GkillDAOManager.ConfigDAOs.RepositoryDAO.DeleteWriteRepositories(r.Context(), userID, request.UpdatedReps)
	if !ok || err != nil {
		gkillError := &message.GkillError{
			ErrorCode:    message.AddUpdatedRepositoriesByUser,
			ErrorMessage: fmt.Sprintf("%s%s", api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REP_WITH_ERROR_MESSAGE"}), err),
		}
		response.Errors = append(response.Errors, gkillError)
		if err != nil {
			err = fmt.Errorf("error at delete add all repositories by users user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		}

		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateRepositoriesSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_REP_WITH_ERROR_MESSAGE"}),
	})
}
