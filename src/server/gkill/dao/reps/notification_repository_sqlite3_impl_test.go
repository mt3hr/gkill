package reps

import (
	"context"
	"testing"
	"time"
)

func makeNotification(id, targetID, content string) Notification {
	now := testTime()
	return Notification{IsDeleted: false, ID: id, TargetID: targetID, Content: content,
		IsNotificated: false, NotificationTime: now,
		CreateTime: now, CreateApp: "test_app", CreateDevice: "test_device", CreateUser: "test_user",
		UpdateTime: now, UpdateApp: "test_app", UpdateUser: "test_user", UpdateDevice: "test_device"}
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
