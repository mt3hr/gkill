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
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/account"
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

	// 自分自身の ID を明示しても弾く（target_user_id の指定そのものが管理者専用）
	selfResp := postJSON(t, ts.URL+"/api/get_plugin_list", &req_res.GetPluginListRequest{
		SessionID:    normalSession,
		LocaleName:   "en",
		TargetUserID: "normaluser_plugin_list",
	})
	defer selfResp.Body.Close()
	if selfResp.StatusCode != http.StatusForbidden {
		t.Errorf("自分の ID を明示したときの status = %d, want %d", selfResp.StatusCode, http.StatusForbidden)
	}
	var selfRes req_res.GetPluginListResponse
	if err := json.NewDecoder(selfResp.Body).Decode(&selfRes); err != nil {
		t.Fatalf("decode self response: %v", err)
	}
	if !hasErrorCode(selfRes.Errors, message.AccountNotHasAdminError) {
		t.Errorf("expected error code %s, got: %+v", message.AccountNotHasAdminError, selfRes.Errors)
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
	// GetAccount は未知の ID に (nil, nil) を返すので、ここは必ず「見つからない」（404）になる
	if !hasErrorCode(res.Errors, message.TargetAccountNotFoundError) {
		t.Errorf("expected %s, got: %+v", message.TargetAccountNotFoundError, res.Errors)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
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

// 管理者は実在する他ユーザの target_user_id を指定して一覧を取得できる。
func TestHandleGetPluginList_AdminCanInspectOtherUser(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	adminSession := loginAndGetSession(t, ts.URL, gkillAPI, "admin", pluginConfigTestPasswordHash)
	createNonAdminSessionForPluginTest(t, ts.URL, gkillAPI, "normaluser_plugin_inspect")

	resp := postJSON(t, ts.URL+"/api/get_plugin_list", &req_res.GetPluginListRequest{
		SessionID:    adminSession,
		LocaleName:   "en",
		TargetUserID: "normaluser_plugin_inspect",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	var res req_res.GetPluginListResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(res.Errors) != 0 {
		t.Errorf("管理者の対象指定がエラーになった: %+v", res.Errors)
	}
}

// HandlePostPluginConfig でも管理者が実在する他ユーザを指定すれば認可を通過し、
// 未知の target_user_id は弾かれる。
func TestHandlePostPluginConfig_AdminTargetUserValidation(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	adminSession := loginAndGetSession(t, ts.URL, gkillAPI, "admin", pluginConfigTestPasswordHash)
	createNonAdminSessionForPluginTest(t, ts.URL, gkillAPI, "normaluser_post_target")

	t.Run("実在する他ユーザは認可・存在確認を通過する", func(t *testing.T) {
		resp := postJSON(t, ts.URL+"/api/post_plugin_config", &req_res.PostPluginConfigRequest{
			SessionID:    adminSession,
			LocaleName:   "en",
			RepName:      "Example",
			TargetUserID: "normaluser_post_target",
			FormData:     map[string]string{"source_dirs": "/example"},
		})
		defer resp.Body.Close()

		var res req_res.PostPluginConfigResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if hasErrorCode(res.Errors, message.AccountNotHasAdminError) {
			t.Errorf("管理者を認可で弾いてはいけない: %+v", res.Errors)
		}
		if hasErrorCode(res.Errors, message.TargetAccountNotFoundError) {
			t.Errorf("実在するアカウントを「見つからない」にしてはいけない: %+v", res.Errors)
		}
	})

	t.Run("実在しない target_user_id は弾かれる", func(t *testing.T) {
		resp := postJSON(t, ts.URL+"/api/post_plugin_config", &req_res.PostPluginConfigRequest{
			SessionID:    adminSession,
			LocaleName:   "en",
			RepName:      "Example",
			TargetUserID: "no_such_user_post",
			FormData:     map[string]string{"source_dirs": "/example"},
		})
		defer resp.Body.Close()

		var res req_res.PostPluginConfigResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(res.Errors) == 0 {
			t.Fatal("実在しない利用者はエラーにすること")
		}
		// GetAccount は未知の ID に (nil, nil) を返すので、ここは必ず「見つからない」（404）になる
		if !hasErrorCode(res.Errors, message.TargetAccountNotFoundError) {
			t.Errorf("expected %s, got: %+v", message.TargetAccountNotFoundError, res.Errors)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})
}

// アカウントの読み出しそのものが失敗したときは、「見つからない」ではなくサーバ側の失敗（GetAccountError）として返す。
// 以前は DB の失敗にも「対象のアカウントが見つかりません」を付けていて、ID の打ち間違いと区別できなかった。
func TestResolvePluginTargetUser_AccountReadFailureIsNotNotFound(t *testing.T) {
	_, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	ctx := contextWithAuth(context.Background(), &AuthContext{
		Account: &account.Account{UserID: "admin", IsAdmin: true},
		UserID:  "admin",
	})
	// 読み出しを失敗させる
	if err := gkillAPI.GkillDAOManager.ConfigDAOs.AccountDAO.Close(ctx); err != nil {
		t.Fatalf("close account dao: %v", err)
	}

	userID, gkillErrors := gkillAPI.resolvePluginTargetUser(ctx, "someone_else", "ja", true)
	if userID != "" {
		t.Errorf("失敗したのに利用者 ID %q を返している", userID)
	}
	if !hasErrorCode(gkillErrors, message.GetAccountError) {
		t.Fatalf("expected %s, got: %+v", message.GetAccountError, gkillErrors)
	}
	if hasErrorCode(gkillErrors, message.TargetAccountNotFoundError) {
		t.Errorf("読み出しの失敗を「見つからない」にしている: %+v", gkillErrors)
	}
	if gkillErrors[0].Cause == nil {
		t.Error("Cause が無い（ログに原因が残らない）")
	}
}
