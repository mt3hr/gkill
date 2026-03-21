package gkill_notification

import (
	"context"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func newTempNotificateTargetDAO(t *testing.T) GkillNotificateTargetDAO {
	t.Helper()
	dir := t.TempDir()
	dao, err := NewGkillNotificateTargetDAOSQLite3Impl(context.Background(), filepath.Join(dir, "notification.db"))
	if err != nil {
		t.Fatalf("failed to create dao: %v", err)
	}
	t.Cleanup(func() { dao.Close(context.Background()) })
	return dao
}

func makeTestTarget(id, userID, publicKey string, subscription JSONString) *GkillNotificateTarget {
	return &GkillNotificateTarget{
		ID:           id,
		UserID:       userID,
		PublicKey:    publicKey,
		Subscription: subscription,
	}
}

func TestNotificateTargetAddAndGetAll(t *testing.T) {
	dao := newTempNotificateTargetDAO(t)
	ctx := context.Background()

	target := makeTestTarget("test-id-1", "user1", "pk-1", `{"endpoint":"https://example.com"}`)
	ok, err := dao.AddGkillNotificationTarget(ctx, target)
	if err != nil {
		t.Fatalf("AddGkillNotificationTarget failed: %v", err)
	}
	if !ok {
		t.Fatal("AddGkillNotificationTarget returned false")
	}

	all, err := dao.GetAllGkillNotificationTargets(ctx)
	if err != nil {
		t.Fatalf("GetAllGkillNotificationTargets failed: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 target, got %d", len(all))
	}
	if all[0].ID != "test-id-1" {
		t.Errorf("ID = %q, want %q", all[0].ID, "test-id-1")
	}
	if all[0].UserID != "user1" {
		t.Errorf("UserID = %q, want %q", all[0].UserID, "user1")
	}
	if all[0].PublicKey != "pk-1" {
		t.Errorf("PublicKey = %q, want %q", all[0].PublicKey, "pk-1")
	}
}

func TestNotificateTargetGetByUserAndKey(t *testing.T) {
	dao := newTempNotificateTargetDAO(t)
	ctx := context.Background()

	t1 := makeTestTarget("test-id-1", "user1", "pk-a", `{"endpoint":"https://a.example.com"}`)
	t2 := makeTestTarget("test-id-2", "user1", "pk-b", `{"endpoint":"https://b.example.com"}`)
	if _, err := dao.AddGkillNotificationTarget(ctx, t1); err != nil {
		t.Fatalf("AddGkillNotificationTarget failed: %v", err)
	}
	if _, err := dao.AddGkillNotificationTarget(ctx, t2); err != nil {
		t.Fatalf("AddGkillNotificationTarget failed: %v", err)
	}

	got, err := dao.GetGkillNotificationTargets(ctx, "user1", "pk-a")
	if err != nil {
		t.Fatalf("GetGkillNotificationTargets failed: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 target, got %d", len(got))
	}
	if got[0].ID != "test-id-1" {
		t.Errorf("ID = %q, want %q", got[0].ID, "test-id-1")
	}
}

func TestNotificateTargetUpdate(t *testing.T) {
	dao := newTempNotificateTargetDAO(t)
	ctx := context.Background()

	target := makeTestTarget("test-id-1", "user1", "pk-1", `{"endpoint":"https://old.example.com"}`)
	if _, err := dao.AddGkillNotificationTarget(ctx, target); err != nil {
		t.Fatalf("AddGkillNotificationTarget failed: %v", err)
	}

	target.Subscription = `{"endpoint":"https://new.example.com"}`
	ok, err := dao.UpdateGkillNotificationTarget(ctx, target)
	if err != nil {
		t.Fatalf("UpdateGkillNotificationTarget failed: %v", err)
	}
	if !ok {
		t.Fatal("UpdateGkillNotificationTarget returned false")
	}

	got, err := dao.GetGkillNotificationTargets(ctx, "user1", "pk-1")
	if err != nil {
		t.Fatalf("GetGkillNotificationTargets failed: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 target, got %d", len(got))
	}
	if string(got[0].Subscription) != `{"endpoint":"https://new.example.com"}` {
		t.Errorf("Subscription = %q, want updated value", got[0].Subscription)
	}
}

func TestNotificateTargetDelete(t *testing.T) {
	dao := newTempNotificateTargetDAO(t)
	ctx := context.Background()

	target := makeTestTarget("test-id-1", "user1", "pk-1", `{"endpoint":"https://example.com"}`)
	if _, err := dao.AddGkillNotificationTarget(ctx, target); err != nil {
		t.Fatalf("AddGkillNotificationTarget failed: %v", err)
	}

	ok, err := dao.DeleteGkillNotificationTarget(ctx, "test-id-1")
	if err != nil {
		t.Fatalf("DeleteGkillNotificationTarget failed: %v", err)
	}
	if !ok {
		t.Fatal("DeleteGkillNotificationTarget returned false")
	}

	all, err := dao.GetAllGkillNotificationTargets(ctx)
	if err != nil {
		t.Fatalf("GetAllGkillNotificationTargets failed: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 targets after delete, got %d", len(all))
	}
}

func TestNotificateTargetEmptyDB(t *testing.T) {
	dao := newTempNotificateTargetDAO(t)
	ctx := context.Background()

	all, err := dao.GetAllGkillNotificationTargets(ctx)
	if err != nil {
		t.Fatalf("GetAllGkillNotificationTargets failed: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 targets on empty DB, got %d", len(all))
	}
}

func TestNotificateTargetMultipleUsers(t *testing.T) {
	dao := newTempNotificateTargetDAO(t)
	ctx := context.Background()

	t1 := makeTestTarget("test-id-1", "user1", "pk-1", `{"endpoint":"https://u1.example.com"}`)
	t2 := makeTestTarget("test-id-2", "user2", "pk-2", `{"endpoint":"https://u2.example.com"}`)
	t3 := makeTestTarget("test-id-3", "user1", "pk-3", `{"endpoint":"https://u1b.example.com"}`)
	for _, tgt := range []*GkillNotificateTarget{t1, t2, t3} {
		if _, err := dao.AddGkillNotificationTarget(ctx, tgt); err != nil {
			t.Fatalf("AddGkillNotificationTarget failed: %v", err)
		}
	}

	got, err := dao.GetGkillNotificationTargets(ctx, "user1", "pk-1")
	if err != nil {
		t.Fatalf("GetGkillNotificationTargets failed: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 target for user1/pk-1, got %d", len(got))
	}
	if got[0].ID != "test-id-1" {
		t.Errorf("ID = %q, want %q", got[0].ID, "test-id-1")
	}

	got2, err := dao.GetGkillNotificationTargets(ctx, "user2", "pk-2")
	if err != nil {
		t.Fatalf("GetGkillNotificationTargets failed: %v", err)
	}
	if len(got2) != 1 {
		t.Fatalf("expected 1 target for user2/pk-2, got %d", len(got2))
	}
	if got2[0].ID != "test-id-2" {
		t.Errorf("ID = %q, want %q", got2[0].ID, "test-id-2")
	}

	all, err := dao.GetAllGkillNotificationTargets(ctx)
	if err != nil {
		t.Fatalf("GetAllGkillNotificationTargets failed: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 total targets, got %d", len(all))
	}
}
