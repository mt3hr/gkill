package gkill_server_api

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// resolvePluginTargetUser はプラグインを見る対象の利用者IDを決めます。
//
// targetUserID が空なら自分自身（ログイン中の利用者）を対象にします。
// 空でなければ管理者権限を要求し、そのアカウントが実在することを確かめてから返します。
// requireAdmin が真のときは、自分自身を対象にする場合でも管理者でなければ弾きます。
//
// 実在確認を飛ばしてはいけません。GetPluginManager は与えた利用者IDでプラグインマネージャを
// 作ってキャッシュに載せるので、存在しない利用者名を通すとマネージャが居座ります。
//
// 戻り値の2つめが空でなければ、呼び出し元はそれを response.Errors に足して戻ること。
func (g *GkillServerAPI) resolvePluginTargetUser(ctx context.Context, targetUserID string, localeName string, requireAdmin bool) (string, []*message.GkillError) {
	auth := AuthFromContext(ctx)

	if requireAdmin || targetUserID != "" {
		if !auth.Account.IsAdmin {
			err := fmt.Errorf("account not has admin user id = %s", auth.Account.UserID)
			slog.Log(ctx, gkill_log.Debug, "account not has admin user id", "error", fmt.Sprintf("%q", err))
			return "", []*message.GkillError{{
				ErrorCode:    message.AccountNotHasAdminError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "NO_ADMIN_PRIVILEGE_MESSAGE"}),
			}}
		}
	}

	if targetUserID == "" {
		return auth.UserID, nil
	}

	targetAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(ctx, targetUserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", targetUserID, err)
		slog.Log(ctx, gkill_log.Debug, "error at get account user id", "error", fmt.Sprintf("%q", err))
		return "", []*message.GkillError{{
			ErrorCode:    message.GetAccountError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "TARGET_ACCOUNT_NOT_FOUND_MESSAGE"}),
			Cause:        err,
		}}
	}
	if targetAccount == nil {
		return "", []*message.GkillError{{
			ErrorCode:    message.TargetAccountNotFoundError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "TARGET_ACCOUNT_NOT_FOUND_MESSAGE"}),
		}}
	}
	return targetAccount.UserID, nil
}
