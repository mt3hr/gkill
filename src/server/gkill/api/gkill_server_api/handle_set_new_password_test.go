package gkill_server_api

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/account"
)

// リセットトークンは72時間で期限切れになる。
// 期限切れであることが伝わらないと、利用者にはリンクが壊れているようにしか見えず、
// 管理者に再発行を頼めばよいことに気づけない。
// 一方で「期限切れである」と答えてよいのはトークンが一致したときだけで、
// 総当たりに対してトークンの存在をもらしてはいけない。

const setNewPasswordTestCredential = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// prepareResetTokenAccount は対象アカウントを「パスワード無効・リセットトークン発行済み」にする。
func prepareResetTokenAccount(t *testing.T, gkillAPI *GkillServerAPI, userID string, token string, expiration time.Time) {
	t.Helper()
	ctx := context.Background()

	targetAccount, err := gkillAPI.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(ctx, userID)
	if err != nil {
		t.Fatalf("GetAccount(%s) failed: %v", userID, err)
	}
	if targetAccount == nil {
		t.Fatalf("account %s not found", userID)
	}

	targetAccount.PasswordHash = nil
	targetAccount.PasswordResetToken = &token
	targetAccount.PasswordResetTokenExpiration = &expiration
	ok, err := gkillAPI.GkillDAOManager.ConfigDAOs.AccountDAO.UpdateAccount(ctx, targetAccount)
	if err != nil || !ok {
		t.Fatalf("UpdateAccount failed: ok=%v err=%v", ok, err)
	}
}

// setNewPassword は /api/set_new_password を叩いてレスポンスを返す。
func setNewPassword(t *testing.T, tsURL string, userID string, resetToken string) req_res.SetNewPasswordResponse {
	t.Helper()

	req := &req_res.SetNewPasswordRequest{
		UserID:            userID,
		ResetToken:        resetToken,
		NewPasswordSha256: setNewPasswordTestCredential,
		LocaleName:        "en",
	}
	resp := postJSON(t, tsURL+"/api/set_new_password", req)
	defer resp.Body.Close()

	var setResp req_res.SetNewPasswordResponse
	if err := json.NewDecoder(resp.Body).Decode(&setResp); err != nil {
		t.Fatalf("decode set new password response: %v", err)
	}
	return setResp
}

// getPasswordHash は対象アカウントのパスワードハッシュを返す。未設定ならnil。
func getPasswordHash(t *testing.T, gkillAPI *GkillServerAPI, userID string) *string {
	t.Helper()

	targetAccount, err := gkillAPI.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetAccount(%s) failed: %v", userID, err)
	}
	if targetAccount == nil {
		t.Fatalf("account %s not found", userID)
	}
	return targetAccount.PasswordHash
}

// TestHandleSetNewPassword_ExpiredTokenIsReportedAsExpired は、期限切れのトークンが
// 汎用の失敗ではなく期限切れとして返ることを確認する。
func TestHandleSetNewPassword_ExpiredTokenIsReportedAsExpired(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	token := GenerateNewID()
	prepareResetTokenAccount(t, gkillAPI, "admin", token, time.Now().Add(-1*time.Hour))

	setResp := setNewPassword(t, ts.URL, "admin", token)
	if len(setResp.Errors) == 0 {
		t.Fatal("期限切れのトークンでパスワード設定が通ってしまっている")
	}
	found := false
	for _, e := range setResp.Errors {
		if e.ErrorCode == message.ExpiredPasswordResetTokenError {
			found = true
		}
	}
	if !found {
		t.Errorf("期限切れとして返っていない: %+v", setResp.Errors)
	}
	if getPasswordHash(t, gkillAPI, "admin") != nil {
		t.Error("パスワードが設定されてしまっている")
	}
}

// TestHandleSetNewPassword_MismatchedTokenIsNotReportedAsExpired は、トークンが
// 一致しない場合に期限切れを名乗らないことを確認する。
// 名乗ると「そのアカウントに期限切れトークンがあるか」を総当たりで探れてしまう。
func TestHandleSetNewPassword_MismatchedTokenIsNotReportedAsExpired(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	prepareResetTokenAccount(t, gkillAPI, "admin", GenerateNewID(), time.Now().Add(-1*time.Hour))

	setResp := setNewPassword(t, ts.URL, "admin", GenerateNewID())
	if len(setResp.Errors) == 0 {
		t.Fatal("一致しないトークンでパスワード設定が通ってしまっている")
	}
	for _, e := range setResp.Errors {
		if e.ErrorCode == message.ExpiredPasswordResetTokenError {
			t.Error("トークンが一致しないのに期限切れとして返っている")
		}
	}
	if getPasswordHash(t, gkillAPI, "admin") != nil {
		t.Error("パスワードが設定されてしまっている")
	}
}

// TestHandleSetNewPassword_ValidTokenSucceeds は、期限内のトークンなら
// パスワードが設定されトークンが消えることを確認する。
// 上の2つが「そもそも常に失敗する」だけでないことの担保でもある。
func TestHandleSetNewPassword_ValidTokenSucceeds(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	token := GenerateNewID()
	prepareResetTokenAccount(t, gkillAPI, "admin", token, time.Now().Add(account.PasswordResetTokenTTL))

	setResp := setNewPassword(t, ts.URL, "admin", token)
	if len(setResp.Errors) != 0 {
		t.Fatalf("期限内のトークンでパスワード設定が失敗している: %+v", setResp.Errors)
	}
	if len(setResp.Messages) == 0 {
		t.Error("成功メッセージが返っていない")
	}

	updatedAccount, err := gkillAPI.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(context.Background(), "admin")
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}
	if updatedAccount.PasswordHash == nil {
		t.Error("パスワードが設定されていない")
	}
	if updatedAccount.PasswordResetToken != nil {
		t.Error("使い終わったリセットトークンが残っている")
	}
}
