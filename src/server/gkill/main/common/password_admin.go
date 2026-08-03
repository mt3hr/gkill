package common

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/account"
	"github.com/mt3hr/gkill/src/server/gkill/dao/account_state"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	"github.com/spf13/cobra"
)

// localAdminSessionTTL はサブコマンドが自分用に発行する管理者セッションの有効期間。
// 1回のコマンド実行が終わるまで持てばよい。
const localAdminSessionTTL = 5 * time.Minute

// ResetPasswordCmd は指定したアカウントのパスワードを無効化し、
// リセットトークンを発行しなおしてURLを表示する。
//
// パスワードの保存方式がArgon2idになったので、DBを読んでもログインはできない。
// そのため「管理者が自分のパスワードを忘れた」「リセットトークンの期限が切れた」
// といった状況から復帰する手段が、サーバと同じマシン上での操作しか残っていない。
// このサブコマンドがその出口になる。
var ResetPasswordCmd = &cobra.Command{
	Use:   "reset_password",
	Short: "指定したアカウントのパスワードを無効化してリセットURLを発行する",
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Usage()
			return
		}
		ctx := cmd.Context()

		configDBRootDir := os.ExpandEnv(gkill_options.ConfigDir)
		accountDAO, err := account.NewAccountDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "account.db"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error at create account dao: %s\n", err)
			return
		}
		defer func() {
			if err := accountDAO.Close(ctx); err != nil {
				slog.Log(ctx, gkill_log.Debug, "error at close account dao", "error", err)
			}
		}()

		expiration := time.Now().Add(account.PasswordResetTokenTTL)
		var sb strings.Builder
		for _, userID := range args {
			targetAccount, err := accountDAO.GetAccount(ctx, userID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error at get account %s: %s\n", userID, err)
				return
			}
			if targetAccount == nil {
				fmt.Fprintf(os.Stderr, "error: account not found %s\n", userID)
				return
			}

			token := sqlite3impl.GenerateNewID()
			targetAccount.PasswordHash = nil
			targetAccount.PasswordResetToken = &token
			targetAccount.PasswordResetTokenExpiration = &expiration
			if _, err := accountDAO.UpdateAccount(ctx, targetAccount); err != nil {
				fmt.Fprintf(os.Stderr, "error at update account %s: %s\n", userID, err)
				return
			}
			sb.WriteString(fmt.Sprintf("  %s : %s\n", userID, PasswordResetPath(userID, token)))
		}

		os.Stdout.WriteString("パスワードを無効化しました。下記のURLから設定しなおしてください。\n")
		os.Stdout.WriteString(fmt.Sprintf("有効期限: %s\n", expiration.Format(sqlite3impl.TimeLayout)))
		os.Stdout.WriteString(sb.String())
	},
}

// issueLocalAdminSession はローカルのDBを直接触って短命の管理者セッションを発行する。
// 返り値のcleanupは、発行したセッションを削除してDAOを閉じる。
//
// サブコマンドがサーバのAPIを叩くための認証手段。
// サーバと同じマシンでconfigディレクトリを読み書きできることを信頼の根拠にしている。
func issueLocalAdminSession(ctx context.Context, configDBRootDir string, device string) (sessionID string, cleanup func(), err error) {
	accountDBFilename := filepath.Join(configDBRootDir, "account.db")
	accountDAO, err := account.NewAccountDAOSQLite3Impl(ctx, accountDBFilename)
	if err != nil {
		return "", nil, fmt.Errorf("error at create account dao: %w", err)
	}
	defer func() {
		if err := accountDAO.Close(ctx); err != nil {
			slog.Log(ctx, gkill_log.Debug, "error at close account dao", "error", err)
		}
	}()

	accounts, err := accountDAO.GetAllAccounts(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("error at get all accounts: %w", err)
	}
	slices.SortFunc(accounts, func(a *account.Account, b *account.Account) int {
		return strings.Compare(a.UserID, b.UserID)
	})

	var adminAccount *account.Account
	for _, a := range accounts {
		if a.IsAdmin && a.IsEnable {
			adminAccount = a
			break
		}
	}
	if adminAccount == nil {
		return "", nil, fmt.Errorf("error: no enabled admin account found in %s", accountDBFilename)
	}

	loginSessionDAO, err := account_state.NewLoginSessionDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "account_state.db"))
	if err != nil {
		return "", nil, fmt.Errorf("error at create login session dao: %w", err)
	}

	loginSession := &account_state.LoginSession{
		ID:              sqlite3impl.GenerateNewID(),
		UserID:          adminAccount.UserID,
		Device:          device,
		ApplicationName: "gkill",
		SessionID:       sqlite3impl.GenerateNewID(),
		ClientIPAddress: "127.0.0.1",
		LoginTime:       time.Now(),
		ExpirationTime:  time.Now().Add(localAdminSessionTTL),
		IsLocalAppUser:  true,
	}
	if _, err := loginSessionDAO.AddLoginSession(ctx, loginSession); err != nil {
		if err := loginSessionDAO.Close(ctx); err != nil {
			slog.Log(ctx, gkill_log.Debug, "error at close login session dao", "error", err)
		}
		return "", nil, fmt.Errorf("error at add login session: %w", err)
	}

	cleanup = func() {
		if _, err := loginSessionDAO.DeleteLoginSession(ctx, loginSession.SessionID); err != nil {
			slog.Log(ctx, gkill_log.Debug, "error at delete local admin session", "error", err)
		}
		if err := loginSessionDAO.Close(ctx); err != nil {
			slog.Log(ctx, gkill_log.Debug, "error at close login session dao", "error", err)
		}
	}
	return loginSession.SessionID, cleanup, nil
}

// PasswordResetPath はパスワード設定画面のパスを組み立てる。ホスト部分は含まない。
func PasswordResetPath(userID string, resetToken string) string {
	return fmt.Sprintf("/set_new_password?user_id=%s&reset_token=%s",
		url.QueryEscape(userID), url.QueryEscape(resetToken))
}
