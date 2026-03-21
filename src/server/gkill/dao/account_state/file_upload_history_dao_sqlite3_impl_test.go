package account_state

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func newTempFileUploadHistoryDAO(t *testing.T) FileUploadHistoryDAO {
	t.Helper()
	dir := t.TempDir()
	dao, err := NewFileUploadHistoryDAOSQLite3Impl(context.Background(), filepath.Join(dir, "file_upload_history.db"))
	if err != nil {
		t.Fatalf("failed to create file upload history dao: %v", err)
	}
	t.Cleanup(func() { dao.Close(context.Background()) })
	return dao
}

func makeTestFileUploadHistory(id, userID, device, fileName string) *FileUploadHistory {
	return &FileUploadHistory{
		ID:            id,
		UserID:        userID,
		Device:        device,
		FileName:      fileName,
		FileSizeByte:  "1024",
		Successed:     true,
		SourceAddress: "127.0.0.1",
		UploadTime:    time.Now().Truncate(time.Second),
	}
}

func TestFileUploadHistoryAddAndGetAll(t *testing.T) {
	dao := newTempFileUploadHistoryDAO(t)
	ctx := context.Background()

	h := makeTestFileUploadHistory("fuh-001", "user1", "device1", "test.txt")
	ok, err := dao.AddFileUploadHistory(ctx, h)
	if err != nil {
		t.Fatalf("AddFileUploadHistory failed: %v", err)
	}
	if !ok {
		t.Fatal("AddFileUploadHistory returned false")
	}

	all, err := dao.GetAllFileUploadHistories(ctx)
	if err != nil {
		t.Fatalf("GetAllFileUploadHistories failed: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 history, got %d", len(all))
	}
	if all[0].ID != "fuh-001" {
		t.Errorf("ID = %q, want %q", all[0].ID, "fuh-001")
	}
	if all[0].FileName != "test.txt" {
		t.Errorf("FileName = %q, want %q", all[0].FileName, "test.txt")
	}
	if all[0].SourceAddress != "127.0.0.1" {
		t.Errorf("SourceAddress = %q, want %q", all[0].SourceAddress, "127.0.0.1")
	}
}

func TestFileUploadHistoryGetByUserDevice(t *testing.T) {
	dao := newTempFileUploadHistoryDAO(t)
	ctx := context.Background()

	h1 := makeTestFileUploadHistory("fuh-1", "user1", "device1", "a.txt")
	h2 := makeTestFileUploadHistory("fuh-2", "user1", "device1", "b.txt")
	h3 := makeTestFileUploadHistory("fuh-3", "user2", "device2", "c.txt")

	for _, h := range []*FileUploadHistory{h1, h2, h3} {
		if _, err := dao.AddFileUploadHistory(ctx, h); err != nil {
			t.Fatalf("AddFileUploadHistory failed: %v", err)
		}
	}

	histories, err := dao.GetFileUploadHistories(ctx, "user1", "device1")
	if err != nil {
		t.Fatalf("GetFileUploadHistories failed: %v", err)
	}
	if len(histories) != 2 {
		t.Errorf("expected 2 histories for user1/device1, got %d", len(histories))
	}

	histories2, err := dao.GetFileUploadHistories(ctx, "user2", "device2")
	if err != nil {
		t.Fatalf("GetFileUploadHistories failed: %v", err)
	}
	if len(histories2) != 1 {
		t.Errorf("expected 1 history for user2/device2, got %d", len(histories2))
	}
}

func TestFileUploadHistoryUpdate(t *testing.T) {
	dao := newTempFileUploadHistoryDAO(t)
	ctx := context.Background()

	h := makeTestFileUploadHistory("fuh-upd", "user1", "device1", "old.txt")
	if _, err := dao.AddFileUploadHistory(ctx, h); err != nil {
		t.Fatalf("AddFileUploadHistory failed: %v", err)
	}

	h.FileName = "new.txt"
	h.FileSizeByte = "2048"
	ok, err := dao.UpdateFileUploadHistory(ctx, h)
	if err != nil {
		t.Fatalf("UpdateFileUploadHistory failed: %v", err)
	}
	if !ok {
		t.Fatal("UpdateFileUploadHistory returned false")
	}

	all, err := dao.GetAllFileUploadHistories(ctx)
	if err != nil {
		t.Fatalf("GetAllFileUploadHistories failed: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 history after update, got %d", len(all))
	}
	if all[0].FileName != "new.txt" {
		t.Errorf("FileName = %q, want %q", all[0].FileName, "new.txt")
	}
	if all[0].FileSizeByte != "2048" {
		t.Errorf("FileSizeByte = %q, want %q", all[0].FileSizeByte, "2048")
	}
}

func TestFileUploadHistoryDelete(t *testing.T) {
	dao := newTempFileUploadHistoryDAO(t)
	ctx := context.Background()

	h := makeTestFileUploadHistory("fuh-del", "user1", "device1", "delete_me.txt")
	if _, err := dao.AddFileUploadHistory(ctx, h); err != nil {
		t.Fatalf("AddFileUploadHistory failed: %v", err)
	}

	ok, err := dao.DeleteFileUploadHistory(ctx, "fuh-del")
	if err != nil {
		t.Fatalf("DeleteFileUploadHistory failed: %v", err)
	}
	if !ok {
		t.Fatal("DeleteFileUploadHistory returned false")
	}

	all, err := dao.GetAllFileUploadHistories(ctx)
	if err != nil {
		t.Fatalf("GetAllFileUploadHistories failed: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 histories after delete, got %d", len(all))
	}
}

func TestFileUploadHistoryGetEmpty(t *testing.T) {
	dao := newTempFileUploadHistoryDAO(t)
	ctx := context.Background()

	all, err := dao.GetAllFileUploadHistories(ctx)
	if err != nil {
		t.Fatalf("GetAllFileUploadHistories failed: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 histories on empty db, got %d", len(all))
	}

	histories, err := dao.GetFileUploadHistories(ctx, "nouser", "nodevice")
	if err != nil {
		t.Fatalf("GetFileUploadHistories failed: %v", err)
	}
	if len(histories) != 0 {
		t.Errorf("expected 0 histories for nonexistent user, got %d", len(histories))
	}
}
