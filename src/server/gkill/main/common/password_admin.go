package common

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	Use:           "reset_password",
	Short:         "指定したアカウントのパスワードを無効化してリセットURLを発行する",
	Args:          cobra.ArbitraryArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Usage()
		}
		configDBRootDir := os.ExpandEnv(gkill_options.ConfigDir)
		return runResetPassword(cmd.Context(), configDBRootDir, args, os.Stdout)
	},
}

// runResetPassword は各ユーザのパスワードを無効化し、成功したユーザのリセットURLをその場で out へ書き出す。
//
// 成功URLをstrings.Builderに溜めて最後にまとめて出すと、途中のユーザで失敗して return した瞬間に
// それまで成功したぶんのURLごと失われる。UpdateAccount成功の直後に即印字することで、
// 何人目で失敗しても成功済みのURLは確実に手元へ残す。失敗したユーザは return せず errors.Join に積んで続行する。
// out を注入可能にしてあるのは、出力内容をテストで検証できるようにするため。
func runResetPassword(ctx context.Context, configDBRootDir string, userIDs []string, out io.Writer) error {
	accountDAO, err := account.NewAccountDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "account.db"))
	if err != nil {
		return fmt.Errorf("error at create account dao: %w", err)
	}
	defer func() {
		if err := accountDAO.Close(ctx); err != nil {
			slog.Log(ctx, gkill_log.Debug, "error at close account dao", "error", err)
		}
	}()

	expiration := time.Now().Add(account.PasswordResetTokenTTL)

	// ヘッダは最初に成功したユーザの直前で1回だけ出す(全員失敗ならヘッダも出さない)。
	headerPrinted := false
	printHeader := func() {
		if headerPrinted {
			return
		}
		headerPrinted = true
		fmt.Fprint(out, "パスワードを無効化しました。下記のURLから設定しなおしてください。\n")
		fmt.Fprintf(out, "有効期限: %s\n", expiration.Format(sqlite3impl.TimeLayout))
	}

	var errs []error
	for _, userID := range userIDs {
		targetAccount, err := accountDAO.GetAccount(ctx, userID)
		if err != nil {
			errs = append(errs, fmt.Errorf("error at get account %s: %w", userID, err))
			continue
		}
		if targetAccount == nil {
			errs = append(errs, fmt.Errorf("error: account not found %s", userID))
			continue
		}

		token := sqlite3impl.GenerateNewID()
		targetAccount.PasswordHash = nil
		targetAccount.PasswordResetToken = &token
		targetAccount.PasswordResetTokenExpiration = &expiration
		if _, err := accountDAO.UpdateAccount(ctx, targetAccount); err != nil {
			errs = append(errs, fmt.Errorf("error at update account %s: %w", userID, err))
			continue
		}
		// 成功したのでこの場で即印字する(途中で失敗しても失われないように)。
		printHeader()
		fmt.Fprintf(out, "  %s : %s\n", userID, PasswordResetPath(userID, token))
	}
	return errors.Join(errs...)
}

// issueLocalAdminSession はローカルのDBを直接触って短命の管理者セッションを発行する。
// 返り値のcleanupは、発行したセッションを削除してDAOを閉じる。
//
// サブコマンドがサーバのAPIを叩くための認証手段。
// サーバと同じマシンでconfigディレクトリを読み書きできることを信頼の根拠にしている。
func issueLocalAdminSession(ctx context.Context, configDBRootDir string, device string) (sessionID string, cleanup func(), err error) {
	adminUserID, err := findLocalAdminUserID(ctx, configDBRootDir)
	if err != nil {
		return "", nil, err
	}
	// update_cacheは単発のPOSTで終わるためTTL延長(refresh)は不要。捨てる。
	sessionID, _, cleanup, err = issueLocalSession(ctx, configDBRootDir, device, adminUserID)
	return sessionID, cleanup, err
}

// findLocalAdminUserID は有効な管理者アカウントのユーザIDを返す。
// 同じ条件のアカウントが複数あっても結果が変わらないよう、ユーザIDの昇順で最初の1つを選ぶ。
func findLocalAdminUserID(ctx context.Context, configDBRootDir string) (string, error) {
	accountDBFilename := filepath.Join(configDBRootDir, "account.db")
	accountDAO, err := account.NewAccountDAOSQLite3Impl(ctx, accountDBFilename)
	if err != nil {
		return "", fmt.Errorf("error at create account dao: %w", err)
	}
	defer func() {
		if err := accountDAO.Close(ctx); err != nil {
			slog.Log(ctx, gkill_log.Debug, "error at close account dao", "error", err)
		}
	}()

	accounts, err := accountDAO.GetAllAccounts(ctx)
	if err != nil {
		return "", fmt.Errorf("error at get all accounts: %w", err)
	}
	slices.SortFunc(accounts, func(a *account.Account, b *account.Account) int {
		return strings.Compare(a.UserID, b.UserID)
	})

	for _, a := range accounts {
		if a.IsAdmin && a.IsEnable {
			return a.UserID, nil
		}
	}
	return "", fmt.Errorf("error: no enabled admin account found in %s", accountDBFilename)
}

// issueLocalSession は指定したユーザの短命セッションをローカルのDBへ直接書いて発行する。
//   - refresh は、発行したセッションの有効期限を今から localAdminSessionTTL 先へ延ばす。
//     auto_tag のように長時間走るコマンドが、処理の合間に呼んで期限切れを防ぐために使う。
//   - cleanup は、発行したセッションを削除してDAOを閉じる。
//
// APIはセッションのユーザとして動くので、あるユーザのKyouを扱うサブコマンドは
// そのユーザのセッションが要る（管理者セッションでは管理者のリポジトリを見てしまう）。
// 信頼の根拠はissueLocalAdminSessionと同じで、管理者を騙らないぶん権限は狭い。
func issueLocalSession(ctx context.Context, configDBRootDir string, device string, userID string) (sessionID string, refresh func() error, cleanup func(), err error) {
	accountDBFilename := filepath.Join(configDBRootDir, "account.db")
	accountDAO, err := account.NewAccountDAOSQLite3Impl(ctx, accountDBFilename)
	if err != nil {
		return "", nil, nil, fmt.Errorf("error at create account dao: %w", err)
	}
	defer func() {
		if err := accountDAO.Close(ctx); err != nil {
			slog.Log(ctx, gkill_log.Debug, "error at close account dao", "error", err)
		}
	}()

	targetAccount, err := accountDAO.GetAccount(ctx, userID)
	if err != nil {
		return "", nil, nil, fmt.Errorf("error at get account %s: %w", userID, err)
	}
	if targetAccount == nil {
		return "", nil, nil, fmt.Errorf("error: account not found %s in %s", userID, accountDBFilename)
	}
	if !targetAccount.IsEnable {
		return "", nil, nil, fmt.Errorf("error: account is disabled %s", userID)
	}

	loginSessionDAO, err := account_state.NewLoginSessionDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "account_state.db"))
	if err != nil {
		return "", nil, nil, fmt.Errorf("error at create login session dao: %w", err)
	}

	loginSession := &account_state.LoginSession{
		ID:              sqlite3impl.GenerateNewID(),
		UserID:          targetAccount.UserID,
		Device:          device,
		ApplicationName: "gkill",
		SessionID:       sqlite3impl.GenerateNewID(),
		ClientIPAddress: "127.0.0.1",
		LoginTime:       time.Now(),
		ExpirationTime:  time.Now().Add(localAdminSessionTTL),
		// IsLocalAppUser は open_file/open_directory（サーバ上でコマンド起動）のゲート。
		// CLI が発行する短命セッション（update_cache / auto_tag）はどれも不要なので false にする（最小権限）。
		IsLocalAppUser: false,
	}
	if _, err := loginSessionDAO.AddLoginSession(ctx, loginSession); err != nil {
		if err := loginSessionDAO.Close(ctx); err != nil {
			slog.Log(ctx, gkill_log.Debug, "error at close login session dao", "error", err)
		}
		return "", nil, nil, fmt.Errorf("error at add login session: %w", err)
	}

	// refresh は同じ行(ID)の EXPIRATION_TIME を今からTTL先へ更新する。SESSION_IDは維持するので発行済みの認証はそのまま使える。
	// handle_login.go のブックマークレットセッションの期限延長と同じ流儀。
	refresh = func() error {
		loginSession.ExpirationTime = time.Now().Add(localAdminSessionTTL)
		if _, err := loginSessionDAO.UpdateLoginSession(ctx, loginSession); err != nil {
			return fmt.Errorf("error at refresh local session ttl: %w", err)
		}
		return nil
	}

	cleanup = func() {
		if _, err := loginSessionDAO.DeleteLoginSession(ctx, loginSession.SessionID); err != nil {
			slog.Log(ctx, gkill_log.Debug, "error at delete local session", "error", err)
		}
		if err := loginSessionDAO.Close(ctx); err != nil {
			slog.Log(ctx, gkill_log.Debug, "error at close login session dao", "error", err)
		}
	}
	return loginSession.SessionID, refresh, cleanup, nil
}

// PasswordResetPath はパスワード設定画面のパスを組み立てる。ホスト部分は含まない。
func PasswordResetPath(userID string, resetToken string) string {
	return fmt.Sprintf("/set_new_password?user_id=%s&reset_token=%s",
		url.QueryEscape(userID), url.QueryEscape(resetToken))
}
