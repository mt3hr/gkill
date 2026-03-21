package account

import (
	"context"
	"path/filepath"
	"testing"
)

func newTempAccountDAO(t *testing.T) AccountDAO {
	t.Helper()
	dir := t.TempDir()
	dao, err := NewAccountDAOSQLite3Impl(context.Background(), filepath.Join(dir, "account.db"))
	if err != nil {
		t.Fatalf("failed to create account dao: %v", err)
	}
	t.Cleanup(func() { dao.Close(context.Background()) })
	return dao
}

func makeTestAccount(userID string) *Account {
	pw := "dummysha256hash"
	return &Account{
		UserID:         userID,
		PasswordSha256: &pw,
		IsAdmin:        false,
		IsEnable:       true,
	}
}

func TestAccountAddAndGet(t *testing.T) {
	dao := newTempAccountDAO(t)
	ctx := context.Background()

	acc := makeTestAccount("user1")
	ok, err := dao.AddAccount(ctx, acc)
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}
	if !ok {
		t.Fatal("AddAccount returned false")
	}

	got, err := dao.GetAccount(ctx, "user1")
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetAccount returned nil")
	}
	if got.UserID != "user1" {
		t.Errorf("UserID = %q, want %q", got.UserID, "user1")
	}
	if !got.IsEnable {
		t.Error("IsEnable should be true")
	}
}

func TestAccountGetAll(t *testing.T) {
	dao := newTempAccountDAO(t)
	ctx := context.Background()

	for _, uid := range []string{"user1", "user2", "user3"} {
		acc := makeTestAccount(uid)
		if _, err := dao.AddAccount(ctx, acc); err != nil {
			t.Fatalf("AddAccount failed: %v", err)
		}
	}

	all, err := dao.GetAllAccounts(ctx)
	if err != nil {
		t.Fatalf("GetAllAccounts failed: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 accounts, got %d", len(all))
	}
}

func TestAccountUpdate(t *testing.T) {
	dao := newTempAccountDAO(t)
	ctx := context.Background()

	acc := makeTestAccount("user-upd")
	if _, err := dao.AddAccount(ctx, acc); err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}

	acc.IsAdmin = true
	ok, err := dao.UpdateAccount(ctx, acc)
	if err != nil {
		t.Fatalf("UpdateAccount failed: %v", err)
	}
	if !ok {
		t.Fatal("UpdateAccount returned false")
	}

	got, err := dao.GetAccount(ctx, "user-upd")
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}
	if !got.IsAdmin {
		t.Error("IsAdmin should be true after update")
	}
}

func TestAccountDelete(t *testing.T) {
	dao := newTempAccountDAO(t)
	ctx := context.Background()

	acc := makeTestAccount("user-del")
	if _, err := dao.AddAccount(ctx, acc); err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}

	ok, err := dao.DeleteAccount(ctx, "user-del")
	if err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}
	if !ok {
		t.Fatal("DeleteAccount returned false")
	}

	got, err := dao.GetAccount(ctx, "user-del")
	if err != nil {
		// Some implementations return error for not found
		return
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestAccountGetNonExistent(t *testing.T) {
	dao := newTempAccountDAO(t)
	ctx := context.Background()

	got, err := dao.GetAccount(ctx, "nonexistent")
	if err != nil {
		// Not found may return error, which is acceptable
		return
	}
	if got != nil {
		t.Error("expected nil for non-existent account")
	}
}
