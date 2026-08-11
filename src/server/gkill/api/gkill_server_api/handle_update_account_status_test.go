package gkill_server_api

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
)

// 管理者アカウントが1つしかない構成が普通なので、自分を無効化すると
// 誰も管理画面に入れなくなり、復帰手段がサーバと同じマシンでのCLI操作だけになる。
// 画面側でもチェックボックスを操作できなくしてあるが、APIを直接叩かれても弾く必要がある。

// updateAccountStatus は /api/update_account_status を叩いてレスポンスを返す。
func updateAccountStatus(t *testing.T, tsURL string, sessionID string, targetUserID string, enable bool) req_res.UpdateAccountStatusResponse {
	t.Helper()

	req := &req_res.UpdateAccountStatusRequest{
		SessionID:    sessionID,
		TargetUserID: targetUserID,
		Enable:       enable,
		LocaleName:   "en",
	}
	resp := postJSON(t, tsURL+"/api/update_account_status", req)
	defer resp.Body.Close()

	var updateResp req_res.UpdateAccountStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&updateResp); err != nil {
		t.Fatalf("decode update account status response: %v", err)
	}
	return updateResp
}

// TestHandleUpdateAccountStatus_RejectsDisablingOwnAccount は、ログイン中の
// アカウント自身を無効化できないことを確認する。
func TestHandleUpdateAccountStatus_RejectsDisablingOwnAccount(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	adminSession := loginAndGetSession(t, ts.URL, gkillAPI, "admin", passwordHash)

	updateResp := updateAccountStatus(t, ts.URL, adminSession, "admin", false)
	if len(updateResp.Errors) == 0 {
		t.Fatal("自分自身の無効化が通ってしまっている")
	}
	found := false
	for _, e := range updateResp.Errors {
		if e.ErrorCode == message.CannotDisableOwnAccountError {
			found = true
		}
	}
	if !found {
		t.Errorf("自分自身の無効化として弾かれていない: %+v", updateResp.Errors)
	}

	adminAccount, err := gkillAPI.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(context.Background(), "admin")
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}
	if !adminAccount.IsEnable {
		t.Error("弾いたはずなのにアカウントが無効化されている")
	}
}

// TestHandleUpdateAccountStatus_AllowsEnablingOwnAccount は、禁止しているのが
// 無効化だけで、自分自身への有効化までは巻き込んでいないことを確認する。
func TestHandleUpdateAccountStatus_AllowsEnablingOwnAccount(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	adminSession := loginAndGetSession(t, ts.URL, gkillAPI, "admin", passwordHash)

	updateResp := updateAccountStatus(t, ts.URL, adminSession, "admin", true)
	if len(updateResp.Errors) != 0 {
		t.Fatalf("自分自身の有効化が弾かれている: %+v", updateResp.Errors)
	}

	adminAccount, err := gkillAPI.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(context.Background(), "admin")
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}
	if !adminAccount.IsEnable {
		t.Error("アカウントが有効になっていない")
	}
}

// TestHandleUpdateAccountStatus_NotFoundAccountDoesNotPanic は、存在しない
// ユーザIDを渡してもpanicせずエラーとして返ることを確認する。
// GetAccount は「見つからなかった」をerrorではなくnilで返すので、
// nilチェックを落とすとこの経路でnilポインタ参照になる。
func TestHandleUpdateAccountStatus_NotFoundAccountDoesNotPanic(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	adminSession := loginAndGetSession(t, ts.URL, gkillAPI, "admin", passwordHash)

	updateResp := updateAccountStatus(t, ts.URL, adminSession, "no_such_user", false)
	if len(updateResp.Errors) == 0 {
		t.Fatal("存在しないユーザの更新が成功扱いになっている")
	}
}
