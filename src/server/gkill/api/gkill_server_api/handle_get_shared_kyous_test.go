package gkill_server_api

import (
	"encoding/json"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
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
