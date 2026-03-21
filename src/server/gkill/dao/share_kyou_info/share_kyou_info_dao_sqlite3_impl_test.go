package share_kyou_info

import (
	"context"
	"path/filepath"
	"testing"
)

func newTempDAO(t *testing.T) ShareKyouInfoDAO {
	t.Helper()
	dir := t.TempDir()
	dao, err := NewShareKyouInfoDAOSQLite3Impl(context.Background(), filepath.Join(dir, "share.db"))
	if err != nil {
		t.Fatalf("failed to create dao: %v", err)
	}
	t.Cleanup(func() { dao.Close(context.Background()) })
	return dao
}

func makeTestShareKyouInfo(id, shareID, userID, device string) *ShareKyouInfo {
	return &ShareKyouInfo{
		ID:                   id,
		ShareID:              shareID,
		UserID:               userID,
		Device:               device,
		ShareTitle:           "Test Share",
		FindQueryJSON:        JSONString(`{"query":"test"}`),
		ViewType:             "list",
		IsShareTimeOnly:      false,
		IsShareWithTags:      false,
		IsShareWithTexts:     false,
		IsShareWithTimeIss:   false,
		IsShareWithLocations: false,
	}
}

func TestShareKyouInfoAddAndGet(t *testing.T) {
	dao := newTempDAO(t)
	ctx := context.Background()

	info := makeTestShareKyouInfo("id-1", "share-1", "user1", "device1")
	ok, err := dao.AddKyouShareInfo(ctx, info)
	if err != nil {
		t.Fatalf("AddKyouShareInfo failed: %v", err)
	}
	if !ok {
		t.Fatal("AddKyouShareInfo returned false")
	}

	got, err := dao.GetKyouShareInfo(ctx, "share-1")
	if err != nil {
		t.Fatalf("GetKyouShareInfo failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetKyouShareInfo returned nil")
	}
	if got.ShareID != "share-1" {
		t.Errorf("ShareID = %q, want %q", got.ShareID, "share-1")
	}
	if got.UserID != "user1" {
		t.Errorf("UserID = %q, want %q", got.UserID, "user1")
	}
	if got.ShareTitle != "Test Share" {
		t.Errorf("ShareTitle = %q, want %q", got.ShareTitle, "Test Share")
	}
}

func TestShareKyouInfoGetByUserDevice(t *testing.T) {
	dao := newTempDAO(t)
	ctx := context.Background()

	info1 := makeTestShareKyouInfo("id-1", "share-1", "user1", "device1")
	info2 := makeTestShareKyouInfo("id-2", "share-2", "user1", "device1")
	info2.ShareTitle = "Second Share"

	if _, err := dao.AddKyouShareInfo(ctx, info1); err != nil {
		t.Fatalf("AddKyouShareInfo failed: %v", err)
	}
	if _, err := dao.AddKyouShareInfo(ctx, info2); err != nil {
		t.Fatalf("AddKyouShareInfo failed: %v", err)
	}

	got, err := dao.GetKyouShareInfos(ctx, "user1", "device1")
	if err != nil {
		t.Fatalf("GetKyouShareInfos failed: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 entries, got %d", len(got))
	}
}

func TestShareKyouInfoGetAll(t *testing.T) {
	dao := newTempDAO(t)
	ctx := context.Background()

	for i, sid := range []string{"share-1", "share-2", "share-3"} {
		info := makeTestShareKyouInfo("id-"+sid, sid, "user1", "device1")
		info.ShareTitle = "Share " + string(rune('A'+i))
		if _, err := dao.AddKyouShareInfo(ctx, info); err != nil {
			t.Fatalf("AddKyouShareInfo failed: %v", err)
		}
	}

	all, err := dao.GetAllKyouShareInfos(ctx)
	if err != nil {
		t.Fatalf("GetAllKyouShareInfos failed: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 entries, got %d", len(all))
	}
}

func TestShareKyouInfoUpdate(t *testing.T) {
	dao := newTempDAO(t)
	ctx := context.Background()

	info := makeTestShareKyouInfo("id-upd", "share-upd", "user1", "device1")
	if _, err := dao.AddKyouShareInfo(ctx, info); err != nil {
		t.Fatalf("AddKyouShareInfo failed: %v", err)
	}

	info.ShareTitle = "Updated Title"
	ok, err := dao.UpdateKyouShareInfo(ctx, info)
	if err != nil {
		t.Fatalf("UpdateKyouShareInfo failed: %v", err)
	}
	if !ok {
		t.Fatal("UpdateKyouShareInfo returned false")
	}

	got, err := dao.GetKyouShareInfo(ctx, "share-upd")
	if err != nil {
		t.Fatalf("GetKyouShareInfo failed: %v", err)
	}
	if got.ShareTitle != "Updated Title" {
		t.Errorf("ShareTitle = %q, want %q", got.ShareTitle, "Updated Title")
	}
}

func TestShareKyouInfoDelete(t *testing.T) {
	dao := newTempDAO(t)
	ctx := context.Background()

	info := makeTestShareKyouInfo("id-del", "share-del", "user1", "device1")
	if _, err := dao.AddKyouShareInfo(ctx, info); err != nil {
		t.Fatalf("AddKyouShareInfo failed: %v", err)
	}

	ok, err := dao.DeleteKyouShareInfo(ctx, "share-del")
	if err != nil {
		t.Fatalf("DeleteKyouShareInfo failed: %v", err)
	}
	if !ok {
		t.Fatal("DeleteKyouShareInfo returned false")
	}

	got, err := dao.GetKyouShareInfo(ctx, "share-del")
	if err != nil {
		// Not found may return error, which is acceptable
		return
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestShareKyouInfoGetNonExistent(t *testing.T) {
	dao := newTempDAO(t)
	ctx := context.Background()

	got, err := dao.GetKyouShareInfo(ctx, "nonexistent")
	if err != nil {
		// Not found may return error, which is acceptable
		return
	}
	if got != nil {
		t.Error("expected nil for non-existent share info")
	}
}

func TestShareKyouInfoUpdateOptions(t *testing.T) {
	dao := newTempDAO(t)
	ctx := context.Background()

	info := makeTestShareKyouInfo("id-opts", "share-opts", "user1", "device1")
	info.IsShareWithTags = true
	info.IsShareWithTexts = true
	info.IsShareTimeOnly = true
	if _, err := dao.AddKyouShareInfo(ctx, info); err != nil {
		t.Fatalf("AddKyouShareInfo failed: %v", err)
	}

	got, err := dao.GetKyouShareInfo(ctx, "share-opts")
	if err != nil {
		t.Fatalf("GetKyouShareInfo failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetKyouShareInfo returned nil")
	}
	if !got.IsShareWithTags {
		t.Error("IsShareWithTags should be true")
	}
	if !got.IsShareWithTexts {
		t.Error("IsShareWithTexts should be true")
	}
	if !got.IsShareTimeOnly {
		t.Error("IsShareTimeOnly should be true")
	}
	if got.IsShareWithTimeIss {
		t.Error("IsShareWithTimeIss should be false")
	}
	if got.IsShareWithLocations {
		t.Error("IsShareWithLocations should be false")
	}
}

func TestShareKyouInfoEmptyDB(t *testing.T) {
	dao := newTempDAO(t)
	ctx := context.Background()

	all, err := dao.GetAllKyouShareInfos(ctx)
	if err != nil {
		t.Fatalf("GetAllKyouShareInfos failed: %v", err)
	}
	if all == nil {
		t.Fatal("GetAllKyouShareInfos returned nil, want empty slice")
	}
	if len(all) != 0 {
		t.Errorf("expected 0 entries, got %d", len(all))
	}
}
