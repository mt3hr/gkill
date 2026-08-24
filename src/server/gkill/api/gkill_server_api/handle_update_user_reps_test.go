package gkill_server_api

// /api/update_user_reps の対象アカウント不在まわりの回帰テスト。
// 正常系（全種別の一括更新・書き込み先重複の検知）は gkill_server_api_test.go の
// TestHandleUpdateUserReps / TestHandleUpdateUserReps_DuplicateWriteDetected にある。

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
)

// TestHandleUpdateUserReps_TargetAccountNotFoundReturns404 は、存在しないユーザIDへの
// リポジトリ一覧更新が TargetAccountNotFoundError(ERR000413) + HTTP 404 になることを確認する。
//
// AccountNotFoundError(ERR000002) は認証経路（auth.go）専用で、クライアントの
// check_auth がこれを見るとログアウトさせる。操作対象不在の経路で混ぜると
// 操作した管理者がその場で締め出される（2026-08 に ERR000413 へ分離）。
func TestHandleUpdateUserReps_TargetAccountNotFoundReturns404(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithConfigRoutes(t)
	defer cleanup()

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", passwordHash)

	// 対象アカウントの検査は UpdatedReps の書き込みより手前にあるので、
	// リポジトリ一覧は空（nil）で足りる。
	updateReq := &req_res.UpdateUserRepsRequest{
		SessionID:    sessionID,
		TargetUserID: "no_such_target_user",
		LocaleName:   "en",
	}
	resp := postJSON(t, tsURL+"/api/update_user_reps", updateReq)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}

	var updateResp req_res.UpdateUserRepsResponse
	if err := json.NewDecoder(resp.Body).Decode(&updateResp); err != nil {
		t.Fatalf("decode update user reps response: %v", err)
	}
	if len(updateResp.Errors) == 0 {
		t.Fatal("存在しないユーザへのリポジトリ更新が成功扱いになっている")
	}
	foundTargetNotFound := false
	for _, e := range updateResp.Errors {
		if e.ErrorCode == message.TargetAccountNotFoundError {
			foundTargetNotFound = true
		}
		if e.ErrorCode == message.AccountNotFoundError {
			t.Errorf("認証経路専用の %s が返っている（check_auth が実行者をログアウトさせる）", message.AccountNotFoundError)
		}
	}
	if !foundTargetNotFound {
		t.Errorf("errors に %s が積まれていない: %+v", message.TargetAccountNotFoundError, updateResp.Errors)
	}
}
