package reps

import (
	"context"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
)

func makeNotification(id, targetID, content string) Notification {
	now := testTime()
	return Notification{IsDeleted: false, ID: id, TargetID: targetID, Content: content,
		IsNotificated: false, NotificationTime: now,
		CreateTime: now, CreateApp: "test_app", CreateDevice: "test_device", CreateUser: "test_user",
		UpdateTime: now, UpdateApp: "test_app", UpdateUser: "test_user", UpdateDevice: "test_device"}
}

// makeNotificationWithDistinctAuditFields は作成・更新の欄6つを全部違う値にした通知を作る。
// makeNotification は作成と更新で同じ値を使うので、作成と更新の取り違えまでは検出できない。
func makeNotificationWithDistinctAuditFields(id, targetID, content string) Notification {
	n := makeNotification(id, targetID, content)
	n.CreateApp, n.CreateDevice, n.CreateUser = "create_app", "create_device", "create_user"
	n.UpdateApp, n.UpdateDevice, n.UpdateUser = "update_app", "update_device", "update_user"
	return n
}

// assertNotificationAuditFieldsRoundTripOnAllReads は通知を1件足し、読み取り口5つすべてで
// 作成・更新の欄6つが入れ替わらずに戻ることを確かめる（assertAuditFieldsRoundTrip の通知版）。
//
// 通知の実 rep とキャッシュ版は SELECT が CREATE_USER, CREATE_DEVICE の順なのに rows.Scan が
// (CreateDevice, CreateUser) の順で、読むたびに作成者と作成端末が入れ替わっていた。読んだ値を
// そのまま新しい版として書き戻す経路（通知の編集・連鎖削除・MCP の更新/削除/復元）では入れ替わりが保存される。
func assertNotificationAuditFieldsRoundTripOnAllReads(t *testing.T, repo NotificationRepository) {
	t.Helper()
	ctx := context.Background()

	n := makeNotificationWithDistinctAuditFields("notif-audit", "target-audit", "作成・更新の欄の往復")
	if err := repo.AddNotificationInfo(ctx, n); err != nil {
		t.Fatalf("AddNotificationInfo failed: %v", err)
	}
	want := auditFieldsOf(n)

	assertOne := func(t *testing.T, notifications []Notification, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if len(notifications) != 1 {
			t.Fatalf("expected 1 notification, got %d", len(notifications))
		}
		assertAuditFieldsRoundTrip(t, want, auditFieldsOf(notifications[0]))
	}

	t.Run("FindNotifications", func(t *testing.T) {
		got, err := repo.FindNotifications(ctx, &find.FindQuery{OnlyLatestData: true})
		assertOne(t, got, err)
	})
	t.Run("GetNotification", func(t *testing.T) {
		got, err := repo.GetNotification(ctx, n.ID, nil)
		if err != nil {
			t.Fatalf("GetNotification failed: %v", err)
		}
		if got == nil {
			t.Fatal("GetNotification returned nil")
		}
		assertAuditFieldsRoundTrip(t, want, auditFieldsOf(got))
	})
	t.Run("GetNotificationsByTargetID", func(t *testing.T) {
		got, err := repo.GetNotificationsByTargetID(ctx, n.TargetID)
		assertOne(t, got, err)
	})
	t.Run("GetNotificationsBetweenNotificationTime", func(t *testing.T) {
		got, err := repo.GetNotificationsBetweenNotificationTime(ctx, n.NotificationTime.Add(-time.Hour), n.NotificationTime.Add(time.Hour))
		assertOne(t, got, err)
	})
	t.Run("GetNotificationHistories", func(t *testing.T) {
		got, err := repo.GetNotificationHistories(ctx, n.ID)
		assertOne(t, got, err)
	})
}

func TestNotificationAuditFieldsRoundTrip(t *testing.T) {
	assertNotificationAuditFieldsRoundTripOnAllReads(t, newTempNotificationRepo(t))
}

func TestNotificationAddAndGet(t *testing.T) {
	repo := newTempNotificationRepo(t)
	ctx := context.Background()

	n := makeNotification("notif-001", "target-001", "テスト通知")
	if err := repo.AddNotificationInfo(ctx, n); err != nil {
		t.Fatalf("AddNotificationInfo failed: %v", err)
	}

	got, err := repo.GetNotification(ctx, "notif-001", nil)
	if err != nil {
		t.Fatalf("GetNotification failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetNotification returned nil")
	}
	if got.ID != "notif-001" {
		t.Errorf("ID = %q, want %q", got.ID, "notif-001")
	}
	if got.TargetID != "target-001" {
		t.Errorf("TargetID = %q, want %q", got.TargetID, "target-001")
	}
	if got.Content != "テスト通知" {
		t.Errorf("Content = %q, want %q", got.Content, "テスト通知")
	}
	if got.IsDeleted != false {
		t.Errorf("IsDeleted = %v, want false", got.IsDeleted)
	}
	if got.IsNotificated != false {
		t.Errorf("IsNotificated = %v, want false", got.IsNotificated)
	}
}

func TestNotificationGetByTargetID(t *testing.T) {
	repo := newTempNotificationRepo(t)
	ctx := context.Background()

	n1 := makeNotification("notif-a1", "target-same", "通知A")
	n2 := makeNotification("notif-a2", "target-same", "通知B")
	n2.UpdateTime = n2.UpdateTime.Add(time.Second)
	n3 := makeNotification("notif-b1", "target-other", "通知C")
	n3.UpdateTime = n3.UpdateTime.Add(2 * time.Second)

	if err := repo.AddNotificationInfo(ctx, n1); err != nil {
		t.Fatalf("AddNotificationInfo failed: %v", err)
	}
	if err := repo.AddNotificationInfo(ctx, n2); err != nil {
		t.Fatalf("AddNotificationInfo failed: %v", err)
	}
	if err := repo.AddNotificationInfo(ctx, n3); err != nil {
		t.Fatalf("AddNotificationInfo failed: %v", err)
	}

	results, err := repo.GetNotificationsByTargetID(ctx, "target-same")
	if err != nil {
		t.Fatalf("GetNotificationsByTargetID failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 notifications for target-same, got %d", len(results))
	}
}

func TestNotificationGetHistories(t *testing.T) {
	repo := newTempNotificationRepo(t)
	ctx := context.Background()

	n1 := makeNotification("notif-hist", "target-hist", "初版通知")
	if err := repo.AddNotificationInfo(ctx, n1); err != nil {
		t.Fatalf("AddNotificationInfo failed: %v", err)
	}

	n2 := makeNotification("notif-hist", "target-hist", "改訂通知")
	n2.UpdateTime = n2.UpdateTime.Add(time.Hour)
	if err := repo.AddNotificationInfo(ctx, n2); err != nil {
		t.Fatalf("AddNotificationInfo failed: %v", err)
	}

	histories, err := repo.GetNotificationHistories(ctx, "notif-hist")
	if err != nil {
		t.Fatalf("GetNotificationHistories failed: %v", err)
	}
	if len(histories) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(histories))
	}
}

// GetNotificationsBetweenNotificationTime は通知スケジューラが
// 「これから飛ばす通知」を拾うための唯一の入口。
// 以前は全版が返っていたので、通知時刻を後ろへずらしても
// 旧版の通知時刻がそのまま残り、消したはずの時刻で通知が飛んでいた。
// 今は最新版だけを見る(onlyLatestData固定true)。
func TestNotificationGetBetweenNotificationTimeSeesOnlyLatestVersion(t *testing.T) {
	repo := newTempNotificationRepo(t)
	ctx := context.Background()

	windowStart := testTime().Add(-1 * time.Hour)
	windowEnd := testTime().Add(1 * time.Hour)

	// 範囲内 → 範囲外へ通知時刻を動かした通知。最新版は範囲外なので拾ってはいけない
	movedOut1 := makeNotification("notif-between-moved", "target-between", "範囲内の初版")
	if err := repo.AddNotificationInfo(ctx, movedOut1); err != nil {
		t.Fatalf("AddNotificationInfo(v1) failed: %v", err)
	}
	movedOut2 := makeNotification("notif-between-moved", "target-between", "範囲外へ動かした新版")
	movedOut2.UpdateTime = movedOut1.UpdateTime.Add(time.Hour)
	movedOut2.NotificationTime = testTime().Add(24 * time.Hour)
	if err := repo.AddNotificationInfo(ctx, movedOut2); err != nil {
		t.Fatalf("AddNotificationInfo(v2) failed: %v", err)
	}

	// 範囲外 → 範囲内へ動かした通知。最新版が範囲内なので拾わなければいけない
	movedIn1 := makeNotification("notif-between-movedin", "target-between", "範囲外の初版")
	movedIn1.NotificationTime = testTime().Add(24 * time.Hour)
	if err := repo.AddNotificationInfo(ctx, movedIn1); err != nil {
		t.Fatalf("AddNotificationInfo(v1) failed: %v", err)
	}
	movedIn2 := makeNotification("notif-between-movedin", "target-between", "範囲内へ動かした新版")
	movedIn2.UpdateTime = movedIn1.UpdateTime.Add(time.Hour)
	if err := repo.AddNotificationInfo(ctx, movedIn2); err != nil {
		t.Fatalf("AddNotificationInfo(v2) failed: %v", err)
	}

	results, err := repo.GetNotificationsBetweenNotificationTime(ctx, windowStart, windowEnd)
	if err != nil {
		t.Fatalf("GetNotificationsBetweenNotificationTime failed: %v", err)
	}

	gotIDs := map[string]bool{}
	for _, notification := range results {
		gotIDs[notification.ID] = true
	}
	if gotIDs["notif-between-moved"] {
		t.Errorf("通知時刻を範囲外へ動かした旧版が拾われた: got %d件", len(results))
	}
	if !gotIDs["notif-between-movedin"] {
		t.Errorf("通知時刻を範囲内へ動かした最新版が拾われていない: got %d件", len(results))
	}
	if len(results) != 1 {
		t.Errorf("最新版だけなら1件のはず: got %d件", len(results))
	}
}

// 範囲の両端(start==通知時刻 / end==通知時刻)を含むこと。
// 通知スケジューラは前回時刻〜現在で区切って繰り返し呼ぶので、
// 端が抜けるとちょうどその時刻の通知だけが永久に飛ばない。
func TestNotificationGetBetweenNotificationTimeIncludesBoundaries(t *testing.T) {
	repo := newTempNotificationRepo(t)
	ctx := context.Background()

	notification := makeNotification("notif-boundary", "target-boundary", "境界の通知")
	if err := repo.AddNotificationInfo(ctx, notification); err != nil {
		t.Fatalf("AddNotificationInfo failed: %v", err)
	}
	notificationTime := notification.NotificationTime

	for name, window := range map[string][2]time.Time{
		"startがちょうど通知時刻":  {notificationTime, notificationTime.Add(time.Hour)},
		"endがちょうど通知時刻":    {notificationTime.Add(-time.Hour), notificationTime},
		"start,endともちょうど": {notificationTime, notificationTime},
	} {
		results, err := repo.GetNotificationsBetweenNotificationTime(ctx, window[0], window[1])
		if err != nil {
			t.Fatalf("%s: GetNotificationsBetweenNotificationTime failed: %v", name, err)
		}
		if len(results) != 1 {
			t.Errorf("%s: 両端は含むはず: got %d件", name, len(results))
		}
	}
}

// 保存値と検索範囲でオフセット表記が違っても正しく比較できること。
// unixepochで正規化せずに文字列のまま比べると、
// "2025-01-15T10:30:00+09:00" と "2025-01-15T02:30:00+00:00" の辞書順比較になって外れる。
func TestNotificationGetBetweenNotificationTimeNormalizesTimeZoneOffset(t *testing.T) {
	repo := newTempNotificationRepo(t)
	ctx := context.Background()

	notification := makeNotification("notif-offset", "target-offset", "オフセット違いの通知")
	if err := repo.AddNotificationInfo(ctx, notification); err != nil {
		t.Fatalf("AddNotificationInfo failed: %v", err)
	}

	// 保存はJST(+09:00)、検索範囲はUTC(+00:00)表記
	windowStart := notification.NotificationTime.Add(-time.Hour).UTC()
	windowEnd := notification.NotificationTime.Add(time.Hour).UTC()

	results, err := repo.GetNotificationsBetweenNotificationTime(ctx, windowStart, windowEnd)
	if err != nil {
		t.Fatalf("GetNotificationsBetweenNotificationTime failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("オフセット表記が違っても同じ時刻として比較されるはず: got %d件", len(results))
	}
}
