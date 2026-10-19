package gkill_server_api

// /api/delete_kyou_cascade（ADR-0636）。画面の旧 TypeScript 実装（cascade-delete-kyou.ts）のテストのうち、
// サーバへ移った挙動をここで固定する。キャッシュ ON / OFF の両方で回す（書き戻しの穴はキャッシュ ON でしか出ない）。

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

const cascadeTestPasswordHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// cascadeHistoryEndpoints は種別ごとの「履歴を引く口」と応答のキー（MCP の EntityTargets と同じ対応）。
var cascadeHistoryEndpoints = map[string][2]string{
	"kmemo":        {"/api/get_kmemo", "kmemo_histories"},
	"tag":          {"/api/get_tag_histories_by_tag_id", "tag_histories"},
	"text":         {"/api/get_text_histories_by_text_id", "text_histories"},
	"notification": {"/api/get_gkill_notification_histories_by_notification_id", "notification_histories"},
	"rekyou":       {"/api/get_rekyou", "rekyou_histories"},
	"mirekyou":     {"/api/get_mirekyou", "mirekyou_histories"},
}

// latestIsDeleted は id の最新版（update_time が最大の版）が削除済みかを返す。
func latestIsDeleted(t *testing.T, tsURL, sessionID, dataType, id string) bool {
	t.Helper()
	endpoint, ok := cascadeHistoryEndpoints[dataType]
	if !ok {
		t.Fatalf("no history endpoint for %s", dataType)
	}
	resp := postJSON(t, tsURL+endpoint[0], map[string]any{"session_id": sessionID, "id": id, "locale_name": "en"})
	defer resp.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode %s: %v", endpoint[0], err)
	}
	histories, _ := body[endpoint[1]].([]any)
	if len(histories) == 0 {
		t.Fatalf("%s %s has no history: %v", dataType, id, body["errors"])
	}
	var latest map[string]any
	var latestTime time.Time
	for _, item := range histories {
		version, _ := item.(map[string]any)
		updateTime, err := time.Parse(time.RFC3339Nano, fmt.Sprint(version["update_time"]))
		if err != nil {
			t.Fatalf("parse update_time %v: %v", version["update_time"], err)
		}
		if latest == nil || updateTime.After(latestTime) {
			latest, latestTime = version, updateTime
		}
	}
	return latest["is_deleted"] == true
}

func postDeleteKyouCascade(t *testing.T, tsURL, sessionID, targetID, dataType, updateApp string) req_res.DeleteKyouCascadeResponse {
	t.Helper()
	resp := postJSON(t, tsURL+"/api/delete_kyou_cascade", &req_res.DeleteKyouCascadeRequest{
		SessionID:  sessionID,
		TargetID:   targetID,
		DataType:   dataType,
		UpdateApp:  updateApp,
		LocaleName: "en",
	})
	defer resp.Body.Close()
	var result req_res.DeleteKyouCascadeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode delete kyou cascade response: %v", err)
	}
	return result
}

func deletedSet(records []*req_res.DeletedRecord) []string {
	out := []string{}
	for _, record := range records {
		out = append(out, record.DataType+":"+record.ID)
	}
	sort.Strings(out)
	return out
}

func addCascadeTag(t *testing.T, tsURL, sessionID, targetID string) string {
	t.Helper()
	now := time.Now().Truncate(time.Second)
	id := GenerateNewID()
	resp := postJSON(t, tsURL+"/api/add_tag", &req_res.AddTagRequest{SessionID: sessionID, LocaleName: "en", Tag: reps.Tag{
		ID: id, TargetID: targetID, Tag: "TagA", RelatedTime: now, CreateTime: now, CreateApp: "test", CreateUser: "admin", UpdateTime: now, UpdateApp: "test", UpdateUser: "admin",
	}})
	defer resp.Body.Close()
	var result req_res.AddTagResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Errors) > 0 {
		t.Fatalf("add tag: %v %+v", err, result.Errors)
	}
	return id
}

func addCascadeText(t *testing.T, tsURL, sessionID, targetID string) string {
	t.Helper()
	now := time.Now().Truncate(time.Second)
	id := GenerateNewID()
	resp := postJSON(t, tsURL+"/api/add_text", &req_res.AddTextRequest{SessionID: sessionID, LocaleName: "en", Text: reps.Text{
		ID: id, TargetID: targetID, Text: "note", RelatedTime: now, CreateTime: now, CreateApp: "test", CreateUser: "admin", UpdateTime: now, UpdateApp: "test", UpdateUser: "admin",
	}})
	defer resp.Body.Close()
	var result req_res.AddTextResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Errors) > 0 {
		t.Fatalf("add text: %v %+v", err, result.Errors)
	}
	return id
}

func addCascadeNotification(t *testing.T, tsURL, sessionID, targetID string) string {
	t.Helper()
	now := time.Now().Truncate(time.Second)
	id := GenerateNewID()
	resp := postJSON(t, tsURL+"/api/add_gkill_notification", &req_res.AddNotificationRequest{SessionID: sessionID, LocaleName: "en", Notification: reps.Notification{
		ID: id, TargetID: targetID, Content: "remind", NotificationTime: now.Add(24 * time.Hour),
		CreateTime: now, CreateApp: "test", CreateUser: "admin", UpdateTime: now, UpdateApp: "test", UpdateUser: "admin",
	}})
	defer resp.Body.Close()
	var result req_res.AddNotificationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Errors) > 0 {
		t.Fatalf("add notification: %v %+v", err, result.Errors)
	}
	return id
}

// addCascadeReKyou は id を指定してリポストを足す（循環参照を作るため、まだ無い id を target にできる）。
func addCascadeReKyou(t *testing.T, tsURL, sessionID, id, targetID string) {
	t.Helper()
	now := time.Now().Truncate(time.Second)
	resp := postJSON(t, tsURL+"/api/add_rekyou", &req_res.AddReKyouRequest{SessionID: sessionID, LocaleName: "en", ReKyou: reps.ReKyou{
		ID: id, TargetID: targetID, RelatedTime: now, DataType: "rekyou",
		CreateTime: now, CreateApp: "test", CreateUser: "admin", UpdateTime: now, UpdateApp: "test", UpdateUser: "admin",
	}})
	defer resp.Body.Close()
	var result req_res.AddReKyouResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Errors) > 0 {
		t.Fatalf("add rekyou: %v %+v", err, result.Errors)
	}
}

// forEachCacheMode はキャッシュ OFF / ON の両方でテストサーバを立てて回す。
func forEachCacheMode(t *testing.T, run func(t *testing.T, tsURL, sessionID string)) {
	for _, cacheInMemory := range []bool{false, true} {
		t.Run(fmt.Sprintf("cache_in_memory=%v", cacheInMemory), func(t *testing.T) {
			if cacheInMemory {
				useCacheInMemory(t)
			}
			tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
			defer cleanup()
			sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", cascadeTestPasswordHash)
			run(t, tsURL, sessionID)
		})
	}
}

// 付随するタグ・テキスト・通知と、参照しているリポスト・タスク化（とそれに付いたタグ）をまとめて消す。
// 追加と同じ秒に消しても新しい版が最新になる（update_time を1秒進める）ことも、ここで一緒に見る。
func TestHandleDeleteKyouCascade_DeletesAttachedAndReferrers(t *testing.T) {
	forEachCacheMode(t, func(t *testing.T, tsURL, sessionID string) {
		kmemoID := addTestKmemo(t, tsURL, sessionID, "連鎖削除の起点")
		tagID := addCascadeTag(t, tsURL, sessionID, kmemoID)
		textID := addCascadeText(t, tsURL, sessionID, kmemoID)
		notificationID := addCascadeNotification(t, tsURL, sessionID, kmemoID)
		rekyouID := GenerateNewID()
		addCascadeReKyou(t, tsURL, sessionID, rekyouID, kmemoID)
		rekyouTagID := addCascadeTag(t, tsURL, sessionID, rekyouID)
		mirekyouID := addTestMiReKyou(t, tsURL, sessionID, kmemoID, "inbox")

		result := postDeleteKyouCascade(t, tsURL, sessionID, kmemoID, "kmemo", "gkill_mcp_write")
		if len(result.Errors) > 0 {
			t.Fatalf("unexpected errors: %+v", result.Errors)
		}
		want := []string{"kmemo:" + kmemoID, "tag:" + tagID, "text:" + textID, "notification:" + notificationID,
			"rekyou:" + rekyouID, "tag:" + rekyouTagID, "mirekyou:" + mirekyouID}
		sort.Strings(want)
		got := deletedSet(result.Deleted)
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("deleted = %v, want %v", got, want)
		}
		for _, record := range result.Deleted {
			if !latestIsDeleted(t, tsURL, sessionID, record.DataType, record.ID) {
				t.Errorf("%s %s is not deleted", record.DataType, record.ID)
			}
		}
	})
}

// リポストのリポスト（多段）も辿り、削除済みのリポストは辿らない。
func TestHandleDeleteKyouCascade_FollowsChainsAndSkipsDeleted(t *testing.T) {
	forEachCacheMode(t, func(t *testing.T, tsURL, sessionID string) {
		kmemoID := addTestKmemo(t, tsURL, sessionID, "多段の起点")
		firstID, secondID, deletedID := GenerateNewID(), GenerateNewID(), GenerateNewID()
		addCascadeReKyou(t, tsURL, sessionID, firstID, kmemoID)
		addCascadeReKyou(t, tsURL, sessionID, secondID, firstID)
		addCascadeReKyou(t, tsURL, sessionID, deletedID, kmemoID)
		// 先に消しておいたリポストは対象に入らない
		if result := postDeleteKyouCascade(t, tsURL, sessionID, deletedID, "rekyou", ""); len(result.Errors) > 0 {
			t.Fatalf("pre-delete: %+v", result.Errors)
		}

		result := postDeleteKyouCascade(t, tsURL, sessionID, kmemoID, "kmemo", "")
		if len(result.Errors) > 0 {
			t.Fatalf("unexpected errors: %+v", result.Errors)
		}
		want := []string{"kmemo:" + kmemoID, "rekyou:" + firstID, "rekyou:" + secondID}
		sort.Strings(want)
		if got := deletedSet(result.Deleted); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("deleted = %v, want %v", got, want)
		}
	})
}

// 循環参照（A→B→A）と自己参照でも止まる。
func TestHandleDeleteKyouCascade_StopsOnCycles(t *testing.T) {
	forEachCacheMode(t, func(t *testing.T, tsURL, sessionID string) {
		aID, bID, selfID := GenerateNewID(), GenerateNewID(), GenerateNewID()
		addCascadeReKyou(t, tsURL, sessionID, aID, bID)
		addCascadeReKyou(t, tsURL, sessionID, bID, aID)
		addCascadeReKyou(t, tsURL, sessionID, selfID, selfID)

		result := postDeleteKyouCascade(t, tsURL, sessionID, aID, "rekyou", "")
		if len(result.Errors) > 0 {
			t.Fatalf("cycle: unexpected errors: %+v", result.Errors)
		}
		want := []string{"rekyou:" + aID, "rekyou:" + bID}
		sort.Strings(want)
		if got := deletedSet(result.Deleted); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("cycle: deleted = %v, want %v", got, want)
		}

		result = postDeleteKyouCascade(t, tsURL, sessionID, selfID, "rekyou", "")
		if len(result.Errors) > 0 {
			t.Fatalf("self: unexpected errors: %+v", result.Errors)
		}
		if got := deletedSet(result.Deleted); fmt.Sprint(got) != fmt.Sprint([]string{"rekyou:" + selfID}) {
			t.Fatalf("self: deleted = %v", got)
		}
	})
}

// 参照が深さの上限を超えたら打ち切ってエラーを返し、何も消さない。
func TestHandleDeleteKyouCascade_DepthLimitDeletesNothing(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", cascadeTestPasswordHash)

	kmemoID := addTestKmemo(t, tsURL, sessionID, "深すぎる起点")
	previous := kmemoID
	// 起点から 34 段（usecase.MaxCascadeDeleteDepth = 32 を超える。コード由来）
	for range 34 {
		id := GenerateNewID()
		addCascadeReKyou(t, tsURL, sessionID, id, previous)
		previous = id
	}
	result := postDeleteKyouCascade(t, tsURL, sessionID, kmemoID, "kmemo", "")
	if len(result.Errors) != 1 || result.Errors[0].ErrorCode != message.DeleteKyouCascadeDepthExceededError {
		t.Fatalf("errors = %+v, want %s", result.Errors, message.DeleteKyouCascadeDepthExceededError)
	}
	if len(result.Deleted) != 0 {
		t.Fatalf("deleted = %v, want none", deletedSet(result.Deleted))
	}
	if latestIsDeleted(t, tsURL, sessionID, "kmemo", kmemoID) {
		t.Fatal("the root was deleted although the cascade was aborted")
	}
}

// 起点にできないもの・無いもの・既に消えているものは、それぞれのエラーコードで断り、何も消さない。
func TestHandleDeleteKyouCascade_RejectsBadRoots(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", cascadeTestPasswordHash)

	kmemoID := addTestKmemo(t, tsURL, sessionID, "起点の検査")
	tagID := addCascadeTag(t, tsURL, sessionID, kmemoID)

	cases := []struct {
		name     string
		id       string
		dataType string
		want     string
	}{
		{name: "付随データは起点にできない", id: tagID, dataType: "tag", want: message.DeleteKyouCascadeUnsupportedDataTypeError},
		{name: "git のコミットは消せない", id: kmemoID, dataType: "git_commit_log", want: message.DeleteKyouCascadeUnsupportedDataTypeError},
		{name: "無い記録", id: GenerateNewID(), dataType: "kmemo", want: message.DeleteKyouCascadeTargetNotFoundError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := postDeleteKyouCascade(t, tsURL, sessionID, tc.id, tc.dataType, "")
			if len(result.Errors) != 1 || result.Errors[0].ErrorCode != tc.want {
				t.Fatalf("errors = %+v, want %s", result.Errors, tc.want)
			}
		})
	}
	if latestIsDeleted(t, tsURL, sessionID, "tag", tagID) {
		t.Fatal("the tag was deleted by a rejected request")
	}

	if result := postDeleteKyouCascade(t, tsURL, sessionID, kmemoID, "kmemo", ""); len(result.Errors) > 0 {
		t.Fatalf("first delete: %+v", result.Errors)
	}
	result := postDeleteKyouCascade(t, tsURL, sessionID, kmemoID, "kmemo", "")
	if len(result.Errors) != 1 || result.Errors[0].ErrorCode != message.DeleteKyouCascadeAlreadyDeletedError {
		t.Fatalf("second delete errors = %+v, want %s", result.Errors, message.DeleteKyouCascadeAlreadyDeletedError)
	}
}

// 射影名（mirekyou_create など）でも起点の種別を取り違えない（mirekyou は mi より先に判定する）。
func TestHandleDeleteKyouCascade_AcceptsProjectionNames(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", cascadeTestPasswordHash)

	kmemoID := addTestKmemo(t, tsURL, sessionID, "タスク化の対象")
	mirekyouID := addTestMiReKyou(t, tsURL, sessionID, kmemoID, "inbox")
	result := postDeleteKyouCascade(t, tsURL, sessionID, mirekyouID, "mirekyou_create", "")
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %+v", result.Errors)
	}
	if got := deletedSet(result.Deleted); fmt.Sprint(got) != fmt.Sprint([]string{"mirekyou:" + mirekyouID}) {
		t.Fatalf("deleted = %v", got)
	}
	// 対象の記録そのものは残る（タスク化を消しても元の記録は消えない）
	if latestIsDeleted(t, tsURL, sessionID, "kmemo", kmemoID) {
		t.Fatal("deleting the mirekyou deleted its target")
	}
}

// 通知の履歴は削除済みの版も返す（タグ・テキストと同じ）。落とすと「消したのに履歴の先頭は消えていない版」になり、
// MCP の gkill_delete_kyou / gkill_restore_kyou の「既に消えている / 既に戻っている」の判定が狂う。
func TestHandleGetNotificationHistories_IncludesDeletedVersion(t *testing.T) {
	forEachCacheMode(t, func(t *testing.T, tsURL, sessionID string) {
		kmemoID := addTestKmemo(t, tsURL, sessionID, "通知の対象")
		notificationID := addCascadeNotification(t, tsURL, sessionID, kmemoID)
		result := postDeleteKyouCascade(t, tsURL, sessionID, kmemoID, "kmemo", "")
		if len(result.Errors) > 0 {
			t.Fatalf("unexpected errors: %+v", result.Errors)
		}
		if !latestIsDeleted(t, tsURL, sessionID, "notification", notificationID) {
			t.Fatal("the notification history does not show the deleted version")
		}
	})
}

// 連鎖削除で書いた削除の版は、作成の欄を元の版のまま持ち、更新の欄に要求者（セッションの利用者・
// サーバの端末名・要求のアプリ名）を刻む。ファイルの記録は temp rep の INSERT で作成者と作成端末が
// 入れ替わり、消すたびにその向きで実 rep へ確定していた（アップロードした記録は作成者=利用者・
// 作成端末=サーバの端末名で値が違うので、入れ替わりが見える）。
func TestHandleDeleteKyouCascade_KeepsCreateFieldsAndStampsUpdateFields(t *testing.T) {
	forEachCacheMode(t, func(t *testing.T, tsURL, sessionID string) {
		uploadResp := postJSON(t, tsURL+"/api/upload_files", &req_res.UploadFilesRequest{
			SessionID: sessionID,
			Files: []*req_res.FileData{{
				FileName:     "cascade_meta.txt",
				DataBase64:   base64.StdEncoding.EncodeToString([]byte("hello")),
				LastModified: time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local), // テストの固定時刻
			}},
			TargetRepName:    writableDirectoryRepName(t, tsURL, sessionID),
			ConflictBehavior: req_res.Rename,
			LocaleName:       "en",
		})
		var uploaded req_res.UploadFilesResponse
		if err := json.NewDecoder(uploadResp.Body).Decode(&uploaded); err != nil || len(uploaded.UploadedKyous) != 1 {
			t.Fatalf("upload: %v %+v", err, uploaded.Errors)
		}
		uploadResp.Body.Close()
		id := uploaded.UploadedKyous[0].ID

		histories := func() []reps.IDFKyou {
			t.Helper()
			resp := postJSON(t, tsURL+"/api/get_idf_kyou", map[string]any{"session_id": sessionID, "id": id, "locale_name": "en"})
			defer resp.Body.Close()
			var got req_res.GetIDFKyouResponse
			if err := json.NewDecoder(resp.Body).Decode(&got); err != nil || len(got.IDFKyouHistories) == 0 {
				t.Fatalf("get idf kyou: %v %+v", err, got.Errors)
			}
			return got.IDFKyouHistories
		}
		original := histories()[0]
		if original.CreateUser != "admin" || original.CreateDevice == "" || original.CreateDevice == original.CreateUser {
			t.Fatalf("前提: アップロードした記録の作成者と作成端末が区別できない: user=%q device=%q", original.CreateUser, original.CreateDevice)
		}
		serverDevice := original.CreateDevice

		if result := postDeleteKyouCascade(t, tsURL, sessionID, id, "idf", "gkill_cascade_test"); len(result.Errors) > 0 {
			t.Fatalf("cascade delete: %+v", result.Errors)
		}
		deleted := histories()[0]
		if !deleted.IsDeleted {
			t.Fatalf("履歴の先頭が削除の版ではない: %+v", deleted)
		}
		if deleted.CreateApp != original.CreateApp || deleted.CreateDevice != original.CreateDevice || deleted.CreateUser != original.CreateUser ||
			!deleted.CreateTime.Equal(original.CreateTime) {
			t.Errorf("削除の版の作成の欄が変わった: got app=%q device=%q user=%q time=%v, want app=%q device=%q user=%q time=%v",
				deleted.CreateApp, deleted.CreateDevice, deleted.CreateUser, deleted.CreateTime,
				original.CreateApp, original.CreateDevice, original.CreateUser, original.CreateTime)
		}
		if deleted.UpdateApp != "gkill_cascade_test" || deleted.UpdateDevice != serverDevice || deleted.UpdateUser != "admin" {
			t.Errorf("削除の版の更新の欄 = app=%q device=%q user=%q, want gkill_cascade_test / %q / admin",
				deleted.UpdateApp, deleted.UpdateDevice, deleted.UpdateUser, serverDevice)
		}
	})
}

// 要求の update_device は、連鎖して書く削除の版すべての update_device にだけ載る（MCP は "mcp" を送る）。
// temp rep のキーには使わないので確定も通り、作成の欄は元のまま。
func TestHandleDeleteKyouCascade_UpdateDeviceStampsEveryDeletedVersion(t *testing.T) {
	forEachCacheMode(t, func(t *testing.T, tsURL, sessionID string) {
		kmemoID := addTestKmemo(t, tsURL, sessionID, "端末名を刻む連鎖削除の起点")
		tagID := addCascadeTag(t, tsURL, sessionID, kmemoID)

		resp := postJSON(t, tsURL+"/api/delete_kyou_cascade", &req_res.DeleteKyouCascadeRequest{
			SessionID: sessionID, TargetID: kmemoID, DataType: "kmemo",
			UpdateApp: "gkill_mcp_write", UpdateDevice: "mcp", LocaleName: "en",
		})
		var result req_res.DeleteKyouCascadeResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Errors) > 0 {
			t.Fatalf("cascade delete: %v %+v", err, result.Errors)
		}
		resp.Body.Close()

		kmemoResp := postJSON(t, tsURL+"/api/get_kmemo", &req_res.GetKmemoRequest{SessionID: sessionID, ID: kmemoID, LocaleName: "en"})
		var kmemos req_res.GetKmemoResponse
		if err := json.NewDecoder(kmemoResp.Body).Decode(&kmemos); err != nil || len(kmemos.KmemoHistories) < 2 {
			t.Fatalf("get kmemo: %v %+v", err, kmemos.Errors)
		}
		kmemoResp.Body.Close()
		deleted, original := kmemos.KmemoHistories[0], kmemos.KmemoHistories[len(kmemos.KmemoHistories)-1]
		if !deleted.IsDeleted || deleted.UpdateDevice != "mcp" || deleted.UpdateApp != "gkill_mcp_write" || deleted.UpdateUser != "admin" {
			t.Errorf("kmemo の削除の版 = deleted=%v app=%q device=%q user=%q, want true / gkill_mcp_write / mcp / admin",
				deleted.IsDeleted, deleted.UpdateApp, deleted.UpdateDevice, deleted.UpdateUser)
		}
		if deleted.CreateDevice != original.CreateDevice || deleted.CreateUser != original.CreateUser || deleted.CreateApp != original.CreateApp {
			t.Errorf("kmemo の作成の欄が変わった: got %q/%q/%q want %q/%q/%q",
				deleted.CreateApp, deleted.CreateDevice, deleted.CreateUser, original.CreateApp, original.CreateDevice, original.CreateUser)
		}

		tagResp := postJSON(t, tsURL+"/api/get_tag_histories_by_tag_id", map[string]any{"session_id": sessionID, "id": tagID, "locale_name": "en"})
		var tags struct {
			TagHistories []reps.Tag `json:"tag_histories"`
		}
		if err := json.NewDecoder(tagResp.Body).Decode(&tags); err != nil || len(tags.TagHistories) < 2 {
			t.Fatalf("get tag histories: %v (%d versions)", err, len(tags.TagHistories))
		}
		tagResp.Body.Close()
		latestTag := tags.TagHistories[0]
		for _, tag := range tags.TagHistories {
			if tag.UpdateTime.After(latestTag.UpdateTime) {
				latestTag = tag
			}
		}
		if !latestTag.IsDeleted || latestTag.UpdateDevice != "mcp" {
			t.Errorf("tag の削除の版 = deleted=%v device=%q, want true / mcp", latestTag.IsDeleted, latestTag.UpdateDevice)
		}
	})
}
