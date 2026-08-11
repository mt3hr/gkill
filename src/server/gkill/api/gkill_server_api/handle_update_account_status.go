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
	"github.com/mt3hr/gkill/src/server/gkill/dao/account"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// HandleUpdateAccountStatus は対象アカウントの有効・無効を切り替えます。
//
// POST /api/update_account_status（wrapAuth）
// req_res.UpdateAccountStatusRequest / req_res.UpdateAccountStatusResponse
//
// 管理者でなければ弾きます。
// 自分自身を無効化する要求も弾きます (最後の管理者が締め出される事故を防ぐため)。
// 書き換えるのはIsEnableだけで、パスワードハッシュ・管理者フラグ・
// パスワードリセットトークンは既存の値をそのまま引き継ぎます。
func (g *GkillServerAPI) HandleUpdateAccountStatus(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateAccountStatusRequest{}
	response := &req_res.UpdateAccountStatusResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update accountStatus response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateAccountStatusResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_ACCOUNT_STATUS_STRUCT_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update accountStatus request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateAccountStatusRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_ACCOUNT_STATUS_STRUCT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device

	// 管理者権限がなければ弾く
	if !auth.Account.IsAdmin {
		err = fmt.Errorf("account not has admin user id = %s", auth.Account.UserID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotHasAdminError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_ACCOUNT_STATUS_NO_AUTH_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 自分自身は無効化させない。
	// 管理者が1人しかいない構成が普通なので、自分を無効化すると
	// 誰も管理画面に入れなくなり、復帰手段がサーバと同じマシンでのCLI操作だけになる。
	// 画面側でもチェックボックスを操作できなくしてあるが、APIを直接叩かれても同じように弾く
	if request.TargetUserID == auth.Account.UserID && !request.Enable {
		err = fmt.Errorf("error at disable own account user id = %s", auth.Account.UserID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.CannotDisableOwnAccountError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DISABLE_OWN_ACCOUNT_MESSAGE"}),
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
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_PASSWORD_RESET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	// GetAccount は「見つからなかった」をerrorではなくnilで返す。
	// nilのまま進むとこの下のフィールド参照でpanicする
	if targetAccount == nil {
		err = fmt.Errorf("error at account not found user id = %s", request.TargetUserID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_ACCOUNT_STATUS_STRUCT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	targetAccountUpdated := &account.Account{
		UserID:                       targetAccount.UserID,
		PasswordHash:                 targetAccount.PasswordHash,
		IsAdmin:                      targetAccount.IsAdmin,
		IsEnable:                     request.Enable,
		PasswordResetToken:           targetAccount.PasswordResetToken,
		PasswordResetTokenExpiration: targetAccount.PasswordResetTokenExpiration,
	}

	ok, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.UpdateAccount(r.Context(), targetAccountUpdated)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at update users account user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.UpdateUsersAccountStatusError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_ACCOUNT_STATUS_STRUCT_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateAccountStatusSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_ACCOUNT_STATUS_STRUCT_UPDATED_GET_MESSAGE"}),
	})
}
