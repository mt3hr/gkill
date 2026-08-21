package common

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/account"
	"github.com/mt3hr/gkill/src/server/gkill/dao/account_state"
)

// newTestConfigDir は account.db に指定アカウントを1件仕込んだ config ディレクトリを返す。
func newTestConfigDir(t *testing.T, acc *account.Account) string {
	t.Helper()
	ctx := context.Background()

	// os.MkdirTemp を使うのは、SQLite のファイルハンドルが残っていると
	// t.TempDir の自動削除が Windows で失敗するため(既存テストと同じ理由)。
	dir, err := os.MkdirTemp("", "gkill_password_admin_test_*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	accountDAO, err := account.NewAccountDAOSQLite3Impl(ctx, filepath.Join(dir, "account.db"))
	if err != nil {
		t.Fatalf("NewAccountDAOSQLite3Impl: %v", err)
	}
	if _, err := accountDAO.AddAccount(ctx, acc); err != nil {
		t.Fatalf("AddAccount: %v", err)
	}
	// issueLocalSession / runResetPassword が同じファイルを開き直すので、先に閉じてハンドルを解放する。
	if err := accountDAO.Close(ctx); err != nil {
		t.Fatalf("account dao close: %v", err)
	}
	return dir
}

// 途中のユーザで失敗しても、それより前に成功したユーザのURLは出力済みで、かつエラーが返る。
func TestRunResetPassword_PartialFailureStillEmitsSucceededURL(t *testing.T) {
	ctx := context.Background()
	dir := newTestConfigDir(t, &account.Account{
		UserID:   "user1",
		IsAdmin:  false,
		IsEnable: true,
	})

	var out bytes.Buffer
	// user1 は存在するので成功、2人目は存在しないので失敗する。
	err := runResetPassword(ctx, dir, []string{"user1", "missing_user"}, &out)
	if err == nil {
		t.Fatal("expected an error because missing_user does not exist")
	}
	if !strings.Contains(err.Error(), "missing_user") {
		t.Errorf("error should mention the failed user, got: %v", err)
	}

	output := out.String()
	// 失敗ユーザがいても、先に成功した user1 のURLは出力されている。
	if !strings.Contains(output, "/set_new_password?user_id=user1") {
		t.Errorf("succeeded user's reset URL should be printed, got:\n%s", output)
	}
	if !strings.Contains(output, "パスワードを無効化しました") {
		t.Errorf("header should be printed once, got:\n%s", output)
	}
	// 失敗ユーザのURLは出ない。
	if strings.Contains(output, "user_id=missing_user") {
		t.Errorf("failed user should not have a reset URL, got:\n%s", output)
	}
}

// 全員が失敗するとヘッダも出さず、エラーを返す。
func TestRunResetPassword_AllFailNoHeader(t *testing.T) {
	ctx := context.Background()
	dir := newTestConfigDir(t, &account.Account{UserID: "user1", IsEnable: true})

	var out bytes.Buffer
	err := runResetPassword(ctx, dir, []string{"nobody1", "nobody2"}, &out)
	if err == nil {
		t.Fatal("expected an error because no user exists")
	}
	if out.Len() != 0 {
		t.Errorf("no header/URL should be printed when all fail, got:\n%s", out.String())
	}
}

// issueLocalSession が発行するセッションは IsLocalAppUser=false(最小権限)で、
// refresh は有効期限を将来へ進める。
func TestIssueLocalSession_MinimalPrivilegeAndRefresh(t *testing.T) {
	ctx := context.Background()
	dir := newTestConfigDir(t, &account.Account{
		UserID:   "admin",
		IsAdmin:  true,
		IsEnable: true,
	})

	sessionID, refresh, cleanup, err := issueLocalSession(ctx, dir, "device1", "admin")
	if err != nil {
		t.Fatalf("issueLocalSession: %v", err)
	}
	defer cleanup()

	// 発行済みセッションを別接続から読み、内容を検証する。
	loginSessionDAO, err := account_state.NewLoginSessionDAOSQLite3Impl(ctx, filepath.Join(dir, "account_state.db"))
	if err != nil {
		t.Fatalf("NewLoginSessionDAOSQLite3Impl: %v", err)
	}
	defer func() {
		if err := loginSessionDAO.Close(ctx); err != nil {
			t.Errorf("login session dao close: %v", err)
		}
	}()

	got, err := loginSessionDAO.GetLoginSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetLoginSession: %v", err)
	}
	if got == nil {
		t.Fatal("issued session not found")
	}
	// open_file/open_directory のゲートを開けない最小権限であること。
	if got.IsLocalAppUser {
		t.Error("IsLocalAppUser should be false for a CLI-issued session")
	}

	// 有効期限をいったん過去へ動かしてから refresh すると、将来へ戻ることを確かめる
	// (秒精度なので、初期値と refresh 後を同一秒内で比べても差が出ない。過去起点で確実に判定する)。
	got.ExpirationTime = time.Now().Add(-1 * time.Hour)
	if _, err := loginSessionDAO.UpdateLoginSession(ctx, got); err != nil {
		t.Fatalf("UpdateLoginSession (push to past): %v", err)
	}
	past, err := loginSessionDAO.GetLoginSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetLoginSession (past): %v", err)
	}
	if !past.ExpirationTime.Before(time.Now()) {
		t.Fatalf("precondition failed: expiration should be in the past, got %s", past.ExpirationTime)
	}

	if err := refresh(); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	after, err := loginSessionDAO.GetLoginSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetLoginSession (after refresh): %v", err)
	}
	if !after.ExpirationTime.After(time.Now()) {
		t.Errorf("refresh should advance expiration into the future, got %s", after.ExpirationTime)
	}
}
