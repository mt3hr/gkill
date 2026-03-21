package gkill_cache

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestNewLatestDataRepositoryAddressSQLite3Impl(t *testing.T) {
	db := openTestDB(t)
	mu := &sync.RWMutex{}

	dao, err := NewLatestDataRepositoryAddressSQLite3Impl("testuser", db, mu)
	if err != nil {
		t.Fatalf("NewLatestDataRepositoryAddressSQLite3Impl: %v", err)
	}
	if dao == nil {
		t.Fatal("expected non-nil DAO")
	}
}

func TestAddAndGetLatestDataRepositoryAddress(t *testing.T) {
	db := openTestDB(t)
	mu := &sync.RWMutex{}
	ctx := context.Background()

	dao, err := NewLatestDataRepositoryAddressSQLite3Impl("testuser", db, mu)
	if err != nil {
		t.Fatalf("NewLatestDataRepositoryAddressSQLite3Impl: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	addr := LatestDataRepositoryAddress{
		IsDeleted:                              false,
		TargetID:                               "target-001",
		TargetIDInData:                         nil,
		LatestDataRepositoryName:               "repo-alpha",
		DataUpdateTime:                         now,
		LatestDataRepositoryAddressUpdatedTime: now,
	}

	ok, err := dao.AddOrUpdateLatestDataRepositoryAddress(ctx, addr)
	if err != nil {
		t.Fatalf("AddOrUpdateLatestDataRepositoryAddress: %v", err)
	}
	if !ok {
		t.Error("expected true from AddOrUpdate")
	}

	// Get by ID
	got, err := dao.GetLatestDataRepositoryAddress(ctx, "target-001")
	if err != nil {
		t.Fatalf("GetLatestDataRepositoryAddress: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if got.TargetID != "target-001" {
		t.Errorf("TargetID = %q, want %q", got.TargetID, "target-001")
	}
	if got.LatestDataRepositoryName != "repo-alpha" {
		t.Errorf("LatestDataRepositoryName = %q, want %q", got.LatestDataRepositoryName, "repo-alpha")
	}
}

func TestGetAllLatestDataRepositoryAddresses(t *testing.T) {
	db := openTestDB(t)
	mu := &sync.RWMutex{}
	ctx := context.Background()

	dao, err := NewLatestDataRepositoryAddressSQLite3Impl("testuser", db, mu)
	if err != nil {
		t.Fatalf("NewLatestDataRepositoryAddressSQLite3Impl: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	for _, id := range []string{"id-1", "id-2", "id-3"} {
		addr := LatestDataRepositoryAddress{
			TargetID:                               id,
			LatestDataRepositoryName:               "repo-" + id,
			DataUpdateTime:                         now,
			LatestDataRepositoryAddressUpdatedTime: now,
		}
		if _, err := dao.AddOrUpdateLatestDataRepositoryAddress(ctx, addr); err != nil {
			t.Fatalf("AddOrUpdate %s: %v", id, err)
		}
	}

	all, err := dao.GetAllLatestDataRepositoryAddresses(ctx)
	if err != nil {
		t.Fatalf("GetAllLatestDataRepositoryAddresses: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("got %d addresses, want 3", len(all))
	}
}

func TestGetByRepName(t *testing.T) {
	db := openTestDB(t)
	mu := &sync.RWMutex{}
	ctx := context.Background()

	dao, err := NewLatestDataRepositoryAddressSQLite3Impl("testuser", db, mu)
	if err != nil {
		t.Fatalf("NewLatestDataRepositoryAddressSQLite3Impl: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	_, _ = dao.AddOrUpdateLatestDataRepositoryAddress(ctx, LatestDataRepositoryAddress{
		TargetID: "a1", LatestDataRepositoryName: "repoA",
		DataUpdateTime: now, LatestDataRepositoryAddressUpdatedTime: now,
	})
	_, _ = dao.AddOrUpdateLatestDataRepositoryAddress(ctx, LatestDataRepositoryAddress{
		TargetID: "b1", LatestDataRepositoryName: "repoB",
		DataUpdateTime: now, LatestDataRepositoryAddressUpdatedTime: now,
	})

	result, err := dao.GetLatestDataRepositoryAddressesByRepName(ctx, "repoA")
	if err != nil {
		t.Fatalf("GetByRepName: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("got %d, want 1", len(result))
	}
}

func TestDeleteLatestDataRepositoryAddress(t *testing.T) {
	db := openTestDB(t)
	mu := &sync.RWMutex{}
	ctx := context.Background()

	dao, err := NewLatestDataRepositoryAddressSQLite3Impl("testuser", db, mu)
	if err != nil {
		t.Fatalf("NewLatestDataRepositoryAddressSQLite3Impl: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	addr := LatestDataRepositoryAddress{
		TargetID: "del-1", LatestDataRepositoryName: "repo",
		DataUpdateTime: now, LatestDataRepositoryAddressUpdatedTime: now,
	}
	_, _ = dao.AddOrUpdateLatestDataRepositoryAddress(ctx, addr)

	ok, err := dao.DeleteLatestDataRepositoryAddress(ctx, &addr)
	if err != nil {
		t.Fatalf("DeleteLatestDataRepositoryAddress: %v", err)
	}
	if !ok {
		t.Error("expected true from Delete")
	}

	got, err := dao.GetLatestDataRepositoryAddress(ctx, "del-1")
	if err != nil {
		t.Fatalf("GetLatestDataRepositoryAddress after delete: %v", err)
	}
	// After delete, should either be nil or marked as deleted
	if got != nil && !got.IsDeleted {
		t.Error("expected nil or IsDeleted=true after deletion")
	}
}

func TestDeleteAllLatestDataRepositoryAddress(t *testing.T) {
	db := openTestDB(t)
	mu := &sync.RWMutex{}
	ctx := context.Background()

	dao, err := NewLatestDataRepositoryAddressSQLite3Impl("testuser", db, mu)
	if err != nil {
		t.Fatalf("NewLatestDataRepositoryAddressSQLite3Impl: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	for _, id := range []string{"x1", "x2"} {
		_, _ = dao.AddOrUpdateLatestDataRepositoryAddress(ctx, LatestDataRepositoryAddress{
			TargetID: id, LatestDataRepositoryName: "repo",
			DataUpdateTime: now, LatestDataRepositoryAddressUpdatedTime: now,
		})
	}

	ok, err := dao.DeleteAllLatestDataRepositoryAddress(ctx)
	if err != nil {
		t.Fatalf("DeleteAllLatestDataRepositoryAddress: %v", err)
	}
	if !ok {
		t.Error("expected true from DeleteAll")
	}

	all, err := dao.GetAllLatestDataRepositoryAddresses(ctx)
	if err != nil {
		t.Fatalf("GetAll after DeleteAll: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("got %d after DeleteAll, want 0", len(all))
	}
}

func TestClose(t *testing.T) {
	db := openTestDB(t)
	mu := &sync.RWMutex{}
	ctx := context.Background()

	dao, err := NewLatestDataRepositoryAddressSQLite3Impl("testuser", db, mu)
	if err != nil {
		t.Fatalf("NewLatestDataRepositoryAddressSQLite3Impl: %v", err)
	}

	if err := dao.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
