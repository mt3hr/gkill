package gkill_server_api

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/account_state"
)

// /api/reset_password は wrapNoAuth で登録されており、認証をハンドラ自身が行う。
// 以前は LoginSessionDAO.GetLoginSession を直接呼んでいたため、
// 「セッションIDがDBに存在するか」しか見ておらず、他エンドポイントが
// getAccountFromSessionID で行っている
//   - 有効期限切れの拒否
//   - ApplicationName の照合
//   - 無効化済みアカウントの拒否
// をすべて素通りしていた。管理者権限さえあれば任意アカウントの
// リセットトークンを発行できるエンドポイントなので、ここが緩いと
// 失効済みセッションやブックマークレット用セッションから全アカウントを奪える。

// insertLoginSession はDAOへ直接ログインセッションを差し込む。
// 有効期限切れやブックマークレット用など、通常のログインでは作れない
// セッションを用意するために使う。
func insertLoginSession(t *testing.T, gkillAPI *GkillServerAPI, session *account_state.LoginSession) {
	t.Helper()

	ok, err := gkillAPI.GkillDAOManager.ConfigDAOs.LoginSessionDAO.AddLoginSession(context.Background(), session)
	if err != nil {
		t.Fatalf("AddLoginSession failed: %v", err)
	}
	if !ok {
		t.Fatal("AddLoginSession returned false")
	}
}

// resetPassword は /api/reset_password を叩いてレスポンスを返す。
func resetPassword(t *testing.T, tsURL, sessionID, targetUserID string) req_res.ResetPasswordResponse {
	t.Helper()

	req := &req_res.ResetPasswordRequest{
		SessionID:    sessionID,
		TargetUserID: targetUserID,
		LocaleName:   "en",
	}
	resp := postJSON(t, tsURL+"/api/reset_password", req)
	defer resp.Body.Close()

	var resetResp req_res.ResetPasswordResponse
	if err := json.NewDecoder(resp.Body).Decode(&resetResp); err != nil {
		t.Fatalf("decode reset password response: %v", err)
	}
	return resetResp
}

// TestHandleResetPassword_RejectsUnusableSessions は、通常の認証経路なら弾かれる
// セッションでパスワードリセットができないことを確認する。
func TestHandleResetPassword_RejectsUnusableSessions(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	// adminでログインできる状態にしておく（正常系の比較対象を作るため）
	adminSession := loginAndGetSession(t, tsURL.URL, gkillAPI, "admin", passwordHash)

	device, err := gkillAPI.GetDevice()
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}
	now := time.Now()

	cases := []struct {
		name    string
		session *account_state.LoginSession
	}{
		{
			// 有効期限切れ。auth.go の期限チェックに相当する
			name: "有効期限切れのセッション",
			session: &account_state.LoginSession{
				ID: GenerateNewID(), UserID: "admin", Device: device,
				ApplicationName: "gkill", SessionID: GenerateNewID(),
				ClientIPAddress: "127.0.0.1",
				LoginTime:       now.Add(-48 * time.Hour),
				ExpirationTime:  now.Add(-24 * time.Hour),
			},
		},
		{
			// ブックマークレット用セッション。URLのクエリ文字列に載る値なので
			// ブラウザ履歴やブックマーク同期から漏れうる
			name: "ブックマークレット用セッション",
			session: &account_state.LoginSession{
				ID: GenerateNewID(), UserID: "admin", Device: device,
				ApplicationName: "urlog_bookmarklet", SessionID: GenerateNewID(),
				ClientIPAddress: "127.0.0.1",
				LoginTime:       now,
				ExpirationTime:  now.Add(30 * 24 * time.Hour),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			insertLoginSession(t, gkillAPI, c.session)

			resetResp := resetPassword(t, tsURL.URL, c.session.SessionID, "admin")
			if len(resetResp.Errors) == 0 {
				t.Fatal("使えないはずのセッションでパスワードリセットが通ってしまっている")
			}
			if resetResp.PasswordResetPathWithoutHost != "" {
				t.Error("リセットトークンが返ってしまっている")
			}
		})
	}

	// 比較対象: 通常のセッションなら成功すること。
	// 上のケースが「そもそも常に失敗する」だけでないことを担保する。
	t.Run("通常のセッションは成功する", func(t *testing.T) {
		resetResp := resetPassword(t, tsURL.URL, adminSession, "admin")
		if len(resetResp.Errors) > 0 {
			t.Fatalf("通常のセッションでリセットが失敗している: %+v", resetResp.Errors)
		}
		if resetResp.PasswordResetPathWithoutHost == "" {
			t.Error("リセットトークンが返っていない")
		}
	})
}

// TestHandleResetPassword_RejectsDisabledAdmin は、無効化済みの管理者アカウントでは
// パスワードリセットできないことを確認する。
// getAccountFromSessionID を経由しない実装では IsEnable が見られておらず、
// 無効化したあとも生きているセッションでリセットを実行できていた。
func TestHandleResetPassword_RejectsDisabledAdmin(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	adminSession := loginAndGetSession(t, tsURL.URL, gkillAPI, "admin", passwordHash)

	ctx := context.Background()
	adminAccount, err := gkillAPI.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(ctx, "admin")
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}
	// セッションは有効なまま、アカウントだけ無効化する
	adminAccount.IsEnable = false
	if ok, err := gkillAPI.GkillDAOManager.ConfigDAOs.AccountDAO.UpdateAccount(ctx, adminAccount); err != nil || !ok {
		t.Fatalf("UpdateAccount failed: ok=%v err=%v", ok, err)
	}

	resetResp := resetPassword(t, tsURL.URL, adminSession, "admin")
	if len(resetResp.Errors) == 0 {
		t.Fatal("無効化済みアカウントのセッションでパスワードリセットが通ってしまっている")
	}
	for _, e := range resetResp.Errors {
		if e.ErrorCode == message.PasswordResetSuccessMessage {
			t.Error("成功扱いになっている")
		}
	}
	if resetResp.PasswordResetPathWithoutHost != "" {
		t.Error("リセットトークンが返ってしまっている")
	}
}
