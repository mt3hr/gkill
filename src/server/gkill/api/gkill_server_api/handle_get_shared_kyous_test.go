package gkill_server_api

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/account"
	"github.com/mt3hr/gkill/src/server/gkill/dao/share_kyou_info"
)

// HandleGetSharedKyous は wrapNoAuth で登録されている＝セッション無しで叩ける公開エンドポイント。
// 共有IDさえ知っていれば誰でも取得できるので、「共有対象として保存した検索条件に
// 一致するKyouだけが返る」ことが唯一の防壁になる。ここが崩れると共有していない
// ライフログが第三者に漏れるため、漏洩側（余計に返っていないか）を重点的に確認する。

// addSharedKyouList は共有情報を1件登録し、登録した内容を返す。
// 取り消しテストで同じ内容をそのまま削除リクエストに渡せるようにしている。
func addSharedKyouList(t *testing.T, tsURL, sessionID, device, findQueryJSON string) *req_res.ShareKyouListInfo {
	t.Helper()

	info := &req_res.ShareKyouListInfo{
		ShareID:              GenerateNewID(),
		UserID:               "admin",
		Device:               device,
		ShareTitle:           "共有テスト",
		FindQueryJSON:        share_kyou_info.JSONString(findQueryJSON),
		ViewType:             "kyou",
		IsShareTimeOnly:      false,
		IsShareWithTags:      true,
		IsShareWithTexts:     true,
		IsShareWithTimeIss:   false,
		IsShareWithLocations: false,
	}
	addReq := &req_res.AddShareKyouListInfoRequest{
		SessionID:         sessionID,
		LocaleName:        "en",
		ShareKyouListInfo: info,
	}
	resp := postJSON(t, tsURL+"/api/add_share_kyou_list_info", addReq)
	defer resp.Body.Close()

	var addResp req_res.AddShareKyouListInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		t.Fatalf("decode add share kyou list info response: %v", err)
	}
	if len(addResp.Errors) > 0 {
		t.Fatalf("add share kyou list info errors: %+v", addResp.Errors)
	}
	return info
}

// addSecondAccount は共有の所有者検証テスト用に、adminとは別の一般アカウントを1つ作る。
// prepareLoginReadyAccount は既存アカウントにパスワードを設定するだけなので、
// その前段としてDAOへ直接登録する。
func addSecondAccount(t *testing.T, gkillAPI *GkillServerAPI, userID string) {
	t.Helper()

	ok, err := gkillAPI.GkillDAOManager.ConfigDAOs.AccountDAO.AddAccount(context.Background(), &account.Account{
		UserID:   userID,
		IsAdmin:  false,
		IsEnable: true,
	})
	if err != nil {
		t.Fatalf("AddAccount(%s) failed: %v", userID, err)
	}
	if !ok {
		t.Fatalf("AddAccount(%s) returned false", userID)
	}
}

// getSharedKyous は共有エンドポイントを「セッションを一切送らずに」叩く。
func getSharedKyous(t *testing.T, tsURL, sharedID string) req_res.GetSharedKyousResponse {
	t.Helper()

	req := &req_res.GetSharedKyousRequest{
		SharedID:   sharedID,
		LocaleName: "en",
	}
	resp := postJSON(t, tsURL+"/api/get_shared_kyous", req)
	defer resp.Body.Close()

	var getResp req_res.GetSharedKyousResponse
	if err := json.NewDecoder(resp.Body).Decode(&getResp); err != nil {
		t.Fatalf("decode get shared kyous response: %v", err)
	}
	return getResp
}

// TestHandleGetSharedKyous_ReturnsOnlySharedKyous は、共有条件に一致するKyouだけが返り、
// 一致しないKyouが混ざらないことを確認する。共有ページからの情報漏洩の回帰テスト。
func TestHandleGetSharedKyous_ReturnsOnlySharedKyous(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", passwordHash)

	device, err := gkillAPI.GetDevice()
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}

	sharedID := addTestKmemo(t, tsURL, sessionID, "kyoyusuruwadai の記録")
	secretID := addTestKmemo(t, tsURL, sessionID, "himitsu の記録")

	// 共有する検索条件は「kyoyusuruwadai を含むもの」だけ
	shareInfo := addSharedKyouList(t, tsURL, sessionID, device,
		`{"use_words":true,"words":["kyoyusuruwadai"],"words_and":true}`)

	getResp := getSharedKyous(t, tsURL, shareInfo.ShareID)
	if len(getResp.Errors) > 0 {
		t.Fatalf("get shared kyous errors: %+v", getResp.Errors)
	}
	if getResp.Title != "共有テスト" {
		t.Errorf("Title = %q, want %q", getResp.Title, "共有テスト")
	}

	foundShared := false
	for _, k := range getResp.Kyous {
		if k.ID == sharedID {
			foundShared = true
		}
		if k.ID == secretID {
			t.Error("共有条件に一致しないKyouが共有ページから取得できてしまっている")
		}
	}
	if !foundShared {
		t.Error("共有条件に一致するKyouが取得できていない")
	}
}

// TestHandleGetSharedKyous_UnknownShareIDReturnsError は、存在しない共有IDが
// 総当たりされても何も返らないことを確認する。
func TestHandleGetSharedKyous_UnknownShareIDReturnsError(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", passwordHash)
	addTestKmemo(t, tsURL, sessionID, "誰にも共有していない記録")

	for _, sharedID := range []string{"", "not-a-real-share-id", GenerateNewID()} {
		t.Run("shared_id="+sharedID, func(t *testing.T) {
			getResp := getSharedKyous(t, tsURL, sharedID)
			if len(getResp.Errors) == 0 {
				t.Error("未知の共有IDでエラーにならなかった")
			}
			if len(getResp.Kyous) != 0 {
				t.Errorf("未知の共有IDでKyouが %d 件返っている", len(getResp.Kyous))
			}
		})
	}
}

// TestHandleGetSharedKyous_RevokedShareIsNotAccessible は、共有を取り消したあと
// 同じURLで取得できなくなることを確認する。
func TestHandleGetSharedKyous_RevokedShareIsNotAccessible(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", passwordHash)

	device, err := gkillAPI.GetDevice()
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}

	addTestKmemo(t, tsURL, sessionID, "kyoyusuruwadai の記録")
	shareInfo := addSharedKyouList(t, tsURL, sessionID, device,
		`{"use_words":true,"words":["kyoyusuruwadai"],"words_and":true}`)

	// 取り消し前は取得できる
	before := getSharedKyous(t, tsURL, shareInfo.ShareID)
	if len(before.Errors) > 0 {
		t.Fatalf("取り消し前の取得でエラー: %+v", before.Errors)
	}

	deleteReq := &req_res.DeleteShareKyouListInfoRequest{
		SessionID:         sessionID,
		LocaleName:        "en",
		ShareKyouListInfo: shareInfo,
	}
	resp := postJSON(t, tsURL+"/api/delete_share_kyou_list_infos", deleteReq)
	defer resp.Body.Close()

	var deleteResp req_res.DeleteShareKyouListInfosResponse
	if err := json.NewDecoder(resp.Body).Decode(&deleteResp); err != nil {
		t.Fatalf("decode delete share kyou list infos response: %v", err)
	}
	if len(deleteResp.Errors) > 0 {
		t.Fatalf("delete share kyou list infos errors: %+v", deleteResp.Errors)
	}

	after := getSharedKyous(t, tsURL, shareInfo.ShareID)
	if len(after.Errors) == 0 {
		t.Error("共有を取り消したあとも共有ページから取得できてしまっている")
	}
	if len(after.Kyous) != 0 {
		t.Errorf("共有取り消し後にKyouが %d 件返っている", len(after.Kyous))
	}
}

// 以下3本は共有情報の「所有者」の扱いを固定する。
// 共有の閲覧側 /api/get_shared_kyous は認証不要で、保存されたレコードの user_id を
// そのまま使って対象ユーザーのリポジトリを開く。つまり誰の共有として保存されるかが
// そのままアクセス範囲になるため、作成・更新・削除の3経路すべてで
// 「セッションの持ち主以外の共有には触れない」ことを担保する必要がある。

// TestHandleAddShareKyouListInfo_IgnoresRequestUserID は、リクエスト本文で
// 他人のuser_idを指定しても、その人のライフログを共有できないことを確認する。
func TestHandleAddShareKyouListInfo_IgnoresRequestUserID(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	adminSession := loginAndGetSession(t, tsURL, gkillAPI, "admin", passwordHash)

	device, err := gkillAPI.GetDevice()
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}

	// adminのライフログを1件用意する
	secretID := addTestKmemo(t, tsURL, adminSession, "himitsu の記録")

	// 別の一般ユーザーでログインし、user_id に admin を詐称した共有を作る
	addSecondAccount(t, gkillAPI, "attacker")
	attackerSession := loginAndGetSession(t, tsURL, gkillAPI, "attacker", passwordHash)

	shareID := GenerateNewID()
	addReq := &req_res.AddShareKyouListInfoRequest{
		SessionID:  attackerSession,
		LocaleName: "en",
		ShareKyouListInfo: &req_res.ShareKyouListInfo{
			ShareID:          shareID,
			UserID:           "admin", // ← 詐称。セッションはattacker
			Device:           device,
			ShareTitle:       "詐称共有",
			FindQueryJSON:    share_kyou_info.JSONString(`{}`),
			ViewType:         "kyou",
			IsShareWithTags:  true,
			IsShareWithTexts: true,
		},
	}
	resp := postJSON(t, tsURL+"/api/add_share_kyou_list_info", addReq)
	defer resp.Body.Close()

	var addResp req_res.AddShareKyouListInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		t.Fatalf("decode add share kyou list info response: %v", err)
	}
	if len(addResp.Errors) > 0 {
		t.Fatalf("add share kyou list info errors: %+v", addResp.Errors)
	}

	// 保存された所有者がセッション側になっていること
	stored, err := gkillAPI.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(context.Background(), shareID)
	if err != nil {
		t.Fatalf("GetKyouShareInfo failed: %v", err)
	}
	if stored == nil {
		t.Fatal("共有情報が保存されていない")
	}
	if stored.UserID != "attacker" {
		t.Errorf("保存された共有の UserID = %q, want %q（リクエスト本文のuser_idが採用されている）", stored.UserID, "attacker")
	}

	// 共有ページから他ユーザーのKyouが取得できないこと。
	// attackerにリポジトリが無くエラーになる場合もあるので、漏洩の有無だけを見る。
	getResp := getSharedKyous(t, tsURL, shareID)
	for _, k := range getResp.Kyous {
		if k.ID == secretID {
			t.Fatal("他ユーザーのKyouが共有ページから取得できてしまっている")
		}
	}
}

// TestHandleUpdateShareKyouListInfo_OtherUsersShareIsRejected は、共有IDを知っているだけの
// 別ユーザーが共有条件を書き換えられないことを確認する。
func TestHandleUpdateShareKyouListInfo_OtherUsersShareIsRejected(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	adminSession := loginAndGetSession(t, tsURL, gkillAPI, "admin", passwordHash)

	device, err := gkillAPI.GetDevice()
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}

	secretID := addTestKmemo(t, tsURL, adminSession, "himitsu の記録")
	addTestKmemo(t, tsURL, adminSession, "kyoyusuruwadai の記録")

	// admin が「kyoyusuruwadai を含むもの」だけを共有する
	shareInfo := addSharedKyouList(t, tsURL, adminSession, device,
		`{"use_words":true,"words":["kyoyusuruwadai"],"words_and":true}`)

	// 別ユーザーが同じ共有IDに対して、全件が返る条件へ広げようとする
	addSecondAccount(t, gkillAPI, "attacker")
	attackerSession := loginAndGetSession(t, tsURL, gkillAPI, "attacker", passwordHash)

	widened := *shareInfo
	widened.FindQueryJSON = share_kyou_info.JSONString(`{}`)
	updateReq := &req_res.UpdateShareKyouListInfoRequest{
		SessionID:         attackerSession,
		LocaleName:        "en",
		ShareKyouListInfo: &widened,
	}
	resp := postJSON(t, tsURL+"/api/update_share_kyou_list_info", updateReq)
	defer resp.Body.Close()

	var updateResp req_res.UpdateShareKyouListInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&updateResp); err != nil {
		t.Fatalf("decode update share kyou list info response: %v", err)
	}
	if len(updateResp.Errors) == 0 {
		t.Error("他ユーザーの共有を更新できてしまっている")
	}

	// 共有条件が広がっていないこと
	getResp := getSharedKyous(t, tsURL, shareInfo.ShareID)
	for _, k := range getResp.Kyous {
		if k.ID == secretID {
			t.Fatal("共有条件が第三者に書き換えられ、共有対象外のKyouが漏れている")
		}
	}
}

// TestHandleDeleteShareKyouListInfos_OtherUsersShareIsRejected は、共有IDを知っているだけの
// 別ユーザーが共有を取り消せないことを確認する。
func TestHandleDeleteShareKyouListInfos_OtherUsersShareIsRejected(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	adminSession := loginAndGetSession(t, tsURL, gkillAPI, "admin", passwordHash)

	device, err := gkillAPI.GetDevice()
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}

	addTestKmemo(t, tsURL, adminSession, "kyoyusuruwadai の記録")
	shareInfo := addSharedKyouList(t, tsURL, adminSession, device,
		`{"use_words":true,"words":["kyoyusuruwadai"],"words_and":true}`)

	addSecondAccount(t, gkillAPI, "attacker")
	attackerSession := loginAndGetSession(t, tsURL, gkillAPI, "attacker", passwordHash)

	deleteReq := &req_res.DeleteShareKyouListInfoRequest{
		SessionID:         attackerSession,
		LocaleName:        "en",
		ShareKyouListInfo: shareInfo,
	}
	resp := postJSON(t, tsURL+"/api/delete_share_kyou_list_infos", deleteReq)
	defer resp.Body.Close()

	var deleteResp req_res.DeleteShareKyouListInfosResponse
	if err := json.NewDecoder(resp.Body).Decode(&deleteResp); err != nil {
		t.Fatalf("decode delete share kyou list infos response: %v", err)
	}
	if len(deleteResp.Errors) == 0 {
		t.Error("他ユーザーの共有を取り消せてしまっている")
	}

	// 共有が残っていること（所有者本人からは引き続き見える）
	after := getSharedKyous(t, tsURL, shareInfo.ShareID)
	if len(after.Errors) > 0 {
		t.Errorf("第三者の削除要求で共有が消えている: %+v", after.Errors)
	}
}
