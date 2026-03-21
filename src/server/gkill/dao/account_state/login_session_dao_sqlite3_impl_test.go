package account_state

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func newTempLoginSessionDAO(t *testing.T) LoginSessionDAO {
	t.Helper()
	dir := t.TempDir()
	dao, err := NewLoginSessionDAOSQLite3Impl(context.Background(), filepath.Join(dir, "login_session.db"))
	if err != nil {
		t.Fatalf("failed to create login session dao: %v", err)
	}
	t.Cleanup(func() { dao.Close(context.Background()) })
	return dao
}

func makeTestLoginSession(id, userID, device, sessionID string) *LoginSession {
	now := time.Now()
	return &LoginSession{
		ID:              id,
		UserID:          userID,
		Device:          device,
		ApplicationName: "test_app",
		SessionID:       sessionID,
		ClientIPAddress: "127.0.0.1",
		LoginTime:       now,
		ExpirationTime:  now.Add(30 * 24 * time.Hour),
		IsLocalAppUser:  false,
	}
}

func TestLoginSessionAddAndGet(t *testing.T) {
	dao := newTempLoginSessionDAO(t)
	ctx := context.Background()

	session := makeTestLoginSession("ls-001", "user1", "device1", "sess-abc")
	ok, err := dao.AddLoginSession(ctx, session)
	if err != nil {
		t.Fatalf("AddLoginSession failed: %v", err)
	}
	if !ok {
		t.Fatal("AddLoginSession returned false")
	}

	got, err := dao.GetLoginSession(ctx, "sess-abc")
	if err != nil {
		t.Fatalf("GetLoginSession failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetLoginSession returned nil")
	}
	if got.UserID != "user1" {
		t.Errorf("UserID = %q, want %q", got.UserID, "user1")
	}
	if got.SessionID != "sess-abc" {
		t.Errorf("SessionID = %q, want %q", got.SessionID, "sess-abc")
	}
}

func TestLoginSessionGetByUserDevice(t *testing.T) {
	dao := newTempLoginSessionDAO(t)
	ctx := context.Background()

	s1 := makeTestLoginSession("ls-1", "user1", "device1", "sess-1")
	s2 := makeTestLoginSession("ls-2", "user1", "device1", "sess-2")
	s3 := makeTestLoginSession("ls-3", "user2", "device2", "sess-3")

	for _, s := range []*LoginSession{s1, s2, s3} {
		if _, err := dao.AddLoginSession(ctx, s); err != nil {
			t.Fatalf("AddLoginSession failed: %v", err)
		}
	}

	sessions, err := dao.GetLoginSessions(ctx, "user1", "device1")
	if err != nil {
		t.Fatalf("GetLoginSessions failed: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions for user1/device1, got %d", len(sessions))
	}
}

func TestLoginSessionUpdate(t *testing.T) {
	dao := newTempLoginSessionDAO(t)
	ctx := context.Background()

	session := makeTestLoginSession("ls-upd", "user1", "device1", "sess-upd")
	if _, err := dao.AddLoginSession(ctx, session); err != nil {
		t.Fatalf("AddLoginSession failed: %v", err)
	}

	session.ClientIPAddress = "192.168.1.1"
	ok, err := dao.UpdateLoginSession(ctx, session)
	if err != nil {
		t.Fatalf("UpdateLoginSession failed: %v", err)
	}
	if !ok {
		t.Fatal("UpdateLoginSession returned false")
	}

	got, err := dao.GetLoginSession(ctx, "sess-upd")
	if err != nil {
		t.Fatalf("GetLoginSession failed: %v", err)
	}
	if got.ClientIPAddress != "192.168.1.1" {
		t.Errorf("ClientIPAddress = %q, want %q", got.ClientIPAddress, "192.168.1.1")
	}
}

func TestLoginSessionDelete(t *testing.T) {
	dao := newTempLoginSessionDAO(t)
	ctx := context.Background()

	session := makeTestLoginSession("ls-del", "user1", "device1", "sess-del")
	if _, err := dao.AddLoginSession(ctx, session); err != nil {
		t.Fatalf("AddLoginSession failed: %v", err)
	}

	ok, err := dao.DeleteLoginSession(ctx, "sess-del")
	if err != nil {
		t.Fatalf("DeleteLoginSession failed: %v", err)
	}
	if !ok {
		t.Fatal("DeleteLoginSession returned false")
	}

	got, err := dao.GetLoginSession(ctx, "sess-del")
	if err != nil {
		return // not found error is acceptable
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}
