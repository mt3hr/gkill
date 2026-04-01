package gkill_server_api

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/account"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func (g *GkillServerAPI) getAccountFromSessionIDWithApplicationName(ctx context.Context, sessionID string, applicationName string, localeName string) (*account.Account, *message.GkillError, error) {
	loginSession, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.GetLoginSession(ctx, sessionID)
	if loginSession == nil || err != nil {
		err = fmt.Errorf("error at get login session session id = %s: %w", sessionID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountSessionNotFoundError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ACCOUNT_AUTH_MESSAGE"}),
		}
		return nil, gkillError, err
	}
	if time.Now().After(loginSession.ExpirationTime) {
		err = fmt.Errorf("session expired for session id = %s", sessionID)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountSessionExpiredError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "SESSION_EXPIRED_MESSAGE"}),
		}
		return nil, gkillError, err
	}

	if loginSession.ApplicationName != applicationName {
		err = fmt.Errorf("error at get account user id = %s: %w", loginSession.UserID, err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ACCOUNT_AUTH_MESSAGE"}),
		}
		return nil, gkillError, err
	}

	account, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(ctx, loginSession.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", loginSession.UserID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ACCOUNT_AUTH_MESSAGE"}),
		}
		return nil, gkillError, err
	}

	if account == nil {
		err = fmt.Errorf("error at get account user id = %s: %w", loginSession.UserID, err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ACCOUNT_AUTH_MESSAGE"}),
		}
		return nil, gkillError, err
	}

	if !account.IsEnable {
		err = fmt.Errorf("error at disable account user id = %s: %w", loginSession.UserID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountDisabledError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "ACCOUNT_DISABLED_MESSAGE"}),
		}
		return nil, gkillError, err
	}

	if info := accessLogInfoFromContext(ctx); info != nil {
		info.UserID = account.UserID
	}

	return account, nil, nil

}

func (g *GkillServerAPI) getAccountFromSessionID(ctx context.Context, sessionID string, localeName string) (*account.Account, *message.GkillError, error) {
	return g.getAccountFromSessionIDWithApplicationName(ctx, sessionID, "gkill", localeName)
}
