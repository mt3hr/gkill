package dao

import (
	"os"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

func TestConfigDAOsStructFields(t *testing.T) {
	c := &ConfigDAOs{}
	// Verify fields exist and are nil-valued (interface fields default to nil)
	if c.AccountDAO != nil {
		t.Error("expected nil AccountDAO")
	}
	if c.LoginSessionDAO != nil {
		t.Error("expected nil LoginSessionDAO")
	}
	if c.FileUploadHistoryDAO != nil {
		t.Error("expected nil FileUploadHistoryDAO")
	}
	if c.ShareKyouInfoDAO != nil {
		t.Error("expected nil ShareKyouInfoDAO")
	}
	if c.ServerConfigDAO != nil {
		t.Error("expected nil ServerConfigDAO")
	}
	if c.ApplicationConfigDAO != nil {
		t.Error("expected nil ApplicationConfigDAO")
	}
	if c.RepositoryDAO != nil {
		t.Error("expected nil RepositoryDAO")
	}
	if c.GkillNotificationTargetDAO != nil {
		t.Error("expected nil GkillNotificationTargetDAO")
	}
}

func TestNewGkillDAOManager(t *testing.T) {
	// Use os.MkdirTemp instead of t.TempDir() because the DAO manager
	// opens SQLite databases that hold file handles, and t.TempDir()
	// auto-cleanup fails on Windows when files are still locked.
	tmpDir, err := os.MkdirTemp("", "gkill_dao_test_*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	// Best-effort cleanup; may fail on Windows due to locked SQLite files.
	defer os.RemoveAll(tmpDir)

	// Override gkill_options to use temp directory
	origConfigDir := gkill_options.ConfigDir
	gkill_options.ConfigDir = tmpDir
	defer func() { gkill_options.ConfigDir = origConfigDir }()

	manager, err := NewGkillDAOManager()
	if err != nil {
		t.Fatalf("NewGkillDAOManager: %v", err)
	}
	if manager == nil {
		t.Fatal("expected non-nil manager")
	}
	if manager.ConfigDAOs == nil {
		t.Error("expected non-nil ConfigDAOs")
	}
}

// TestSetSkipIDF_RefCount は SetSkipIDF が参照カウントとして働くことを検証する。
// 重なる Pause/Resume で、内側の Resume 後も外側が生きている間は skip を継続し、
// 全 Resume 後に初めて再開されること、二重 Resume で負数にならないことを確かめる。
func TestSetSkipIDF_RefCount(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gkill_dao_skipidf_test_*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origConfigDir := gkill_options.ConfigDir
	gkill_options.ConfigDir = tmpDir
	defer func() { gkill_options.ConfigDir = origConfigDir }()

	manager, err := NewGkillDAOManager()
	if err != nil {
		t.Fatalf("NewGkillDAOManager: %v", err)
	}

	// 初期状態は再開中(カウント0)
	if got := manager.skipUpdateCache.Load(); got != 0 {
		t.Fatalf("initial skip count = %d, want 0", got)
	}

	// 外側 Pause
	manager.SetSkipIDF(true)
	if got := manager.skipUpdateCache.Load(); got != 1 {
		t.Fatalf("after outer pause skip count = %d, want 1", got)
	}

	// 内側 Pause (重なる)
	manager.SetSkipIDF(true)
	if got := manager.skipUpdateCache.Load(); got != 2 {
		t.Fatalf("after inner pause skip count = %d, want 2", got)
	}

	// 内側 Resume。外側がまだ生きているので skip 継続(カウント>0)
	manager.SetSkipIDF(false)
	if got := manager.skipUpdateCache.Load(); got != 1 {
		t.Fatalf("after inner resume skip count = %d, want 1 (still skipping)", got)
	}

	// 外側 Resume。全 Resume したので再開(カウント0)
	manager.SetSkipIDF(false)
	if got := manager.skipUpdateCache.Load(); got != 0 {
		t.Fatalf("after outer resume skip count = %d, want 0 (resumed)", got)
	}

	// 対応する Pause の無い二重 Resume では負数にしない
	manager.SetSkipIDF(false)
	if got := manager.skipUpdateCache.Load(); got != 0 {
		t.Fatalf("after extra resume skip count = %d, want 0 (never negative)", got)
	}
}
