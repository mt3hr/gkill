package gkill_server_api

// プラグイン設定の認可（ADR-0714）。
//
// 設定の中身は source_dirs ＝「gkill のプロセスが読めるフォルダの指定」なので、
// 一般利用者に開放すると自分のデータの外へ手が届く。画面（アカウント管理 → プラグイン設定）を
// 管理者だけに見せるだけでは、API を直接叩けば通ってしまうのでサーバ側でも弾く。
//
// 一方で get_plugin_list は一般利用者も呼ぶ（プラグイン記録の編集・削除メニューの抑止に使う）。
// ここを一緒に塞ぐと、エラーも出ないまま一般利用者のコンテキストメニューが壊れるので、
// 「target_user_id が空なら誰でも、空でなければ管理者だけ」の境目をここで固定する。

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
)

const pluginConfigTestPasswordHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// createNonAdminSessionForPluginTest は非管理者のアカウントを作ってログインし、セッションIDを返す。
func createNonAdminSessionForPluginTest(t *testing.T, tsURL string, gkillAPI *GkillServerAPI, userID string) string {
	t.Helper()

	adminSession := loginAndGetSession(t, tsURL, gkillAPI, "admin", pluginConfigTestPasswordHash)
	addReq := &req_res.AddAccountRequest{
		SessionID: adminSession,
		AccountInfo: &req_res.Account{
			UserID:   userID,
			IsAdmin:  false,
			IsEnable: true,
		},
		DoInitialize: false,
		LocaleName:   "en",
	}
	resp := postJSON(t, tsURL+"/api/add_user", addReq)
	resp.Body.Close()

	prepareLoginReadyAccount(t, gkillAPI, userID, pluginConfigTestPasswordHash)
	return loginAndGetSession(t, tsURL, gkillAPI, userID, pluginConfigTestPasswordHash)
}

func TestHandleGetPluginConfigHTML_NonAdminIsRejected(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	normalSession := createNonAdminSessionForPluginTest(t, ts.URL, gkillAPI, "normaluser_plugin_get")

	// 自分自身を対象にしても（target_user_id が空でも）弾かれる
	resp := postJSON(t, ts.URL+"/api/get_plugin_config_html", &req_res.GetPluginConfigHTMLRequest{
		SessionID:  normalSession,
		LocaleName: "en",
		RepName:    "Example",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
	var res req_res.GetPluginConfigHTMLResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !hasErrorCode(res.Errors, message.AccountNotHasAdminError) {
		t.Errorf("expected error code %s, got: %+v", message.AccountNotHasAdminError, res.Errors)
	}
	if res.HTML != "" {
		t.Error("設定画面のHTMLを返してはいけない（取り込み元のフォルダ構成が出る）")
	}
}

func TestHandlePostPluginConfig_NonAdminIsRejected(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	normalSession := createNonAdminSessionForPluginTest(t, ts.URL, gkillAPI, "normaluser_plugin_post")

	resp := postJSON(t, ts.URL+"/api/post_plugin_config", &req_res.PostPluginConfigRequest{
		SessionID:  normalSession,
		LocaleName: "en",
		RepName:    "Example",
		FormData:   map[string]string{"source_dirs": "/"},
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
	var res req_res.PostPluginConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !hasErrorCode(res.Errors, message.AccountNotHasAdminError) {
		t.Errorf("expected error code %s, got: %+v", message.AccountNotHasAdminError, res.Errors)
	}
}

// get_plugin_list は一般利用者も呼ぶ（プラグイン記録の編集・削除メニューの抑止）。
// 自分の分は通し、他人を指したときだけ弾く。
func TestHandleGetPluginList_TargetUserIDRequiresAdmin(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	normalSession := createNonAdminSessionForPluginTest(t, ts.URL, gkillAPI, "normaluser_plugin_list")

	ownResp := postJSON(t, ts.URL+"/api/get_plugin_list", &req_res.GetPluginListRequest{
		SessionID:  normalSession,
		LocaleName: "en",
	})
	defer ownResp.Body.Close()
	if ownResp.StatusCode != http.StatusOK {
		t.Errorf("自分の一覧の status = %d, want 200", ownResp.StatusCode)
	}
	var ownRes req_res.GetPluginListResponse
	if err := json.NewDecoder(ownResp.Body).Decode(&ownRes); err != nil {
		t.Fatalf("decode own response: %v", err)
	}
	if len(ownRes.Errors) != 0 {
		t.Errorf("自分の一覧はエラーにならないこと: %+v", ownRes.Errors)
	}

	otherResp := postJSON(t, ts.URL+"/api/get_plugin_list", &req_res.GetPluginListRequest{
		SessionID:    normalSession,
		LocaleName:   "en",
		TargetUserID: "admin",
	})
	defer otherResp.Body.Close()
	if otherResp.StatusCode != http.StatusForbidden {
		t.Errorf("他人の一覧の status = %d, want %d", otherResp.StatusCode, http.StatusForbidden)
	}
	var otherRes req_res.GetPluginListResponse
	if err := json.NewDecoder(otherResp.Body).Decode(&otherRes); err != nil {
		t.Fatalf("decode other response: %v", err)
	}
	if !hasErrorCode(otherRes.Errors, message.AccountNotHasAdminError) {
		t.Errorf("expected error code %s, got: %+v", message.AccountNotHasAdminError, otherRes.Errors)
	}
	if len(otherRes.Plugins) != 0 {
		t.Error("弾いたのにプラグインを返してはいけない")
	}
}

// 実在しない利用者を通すと、GetPluginManager がその名前でマネージャを作って居座る。
func TestHandleGetPluginList_UnknownTargetUserIsRejected(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	adminSession := loginAndGetSession(t, ts.URL, gkillAPI, "admin", pluginConfigTestPasswordHash)

	resp := postJSON(t, ts.URL+"/api/get_plugin_list", &req_res.GetPluginListRequest{
		SessionID:    adminSession,
		LocaleName:   "en",
		TargetUserID: "no_such_user",
	})
	defer resp.Body.Close()

	var res req_res.GetPluginListResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(res.Errors) == 0 {
		t.Fatal("実在しない利用者はエラーにすること")
	}
	if !hasErrorCode(res.Errors, message.TargetAccountNotFoundError) && !hasErrorCode(res.Errors, message.GetAccountError) {
		t.Errorf("expected %s or %s, got: %+v", message.TargetAccountNotFoundError, message.GetAccountError, res.Errors)
	}
}

// 管理者は他人を指しても認可では弾かれない（プラグインが無いので「見つかりません」になる）。
func TestHandleGetPluginConfigHTML_AdminIsNotRejectedByAuth(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	adminSession := loginAndGetSession(t, ts.URL, gkillAPI, "admin", pluginConfigTestPasswordHash)
	createNonAdminSessionForPluginTest(t, ts.URL, gkillAPI, "normaluser_plugin_target")

	resp := postJSON(t, ts.URL+"/api/get_plugin_config_html", &req_res.GetPluginConfigHTMLRequest{
		SessionID:    adminSession,
		LocaleName:   "en",
		RepName:      "Example",
		TargetUserID: "normaluser_plugin_target",
	})
	defer resp.Body.Close()

	var res req_res.GetPluginConfigHTMLResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if hasErrorCode(res.Errors, message.AccountNotHasAdminError) {
		t.Errorf("管理者を認可で弾いてはいけない: %+v", res.Errors)
	}
	if hasErrorCode(res.Errors, message.TargetAccountNotFoundError) {
		t.Errorf("実在するアカウントを「見つからない」にしてはいけない: %+v", res.Errors)
	}
}
