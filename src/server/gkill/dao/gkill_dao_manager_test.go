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
	if c.AppllicationConfigDAO != nil {
		t.Error("expected nil AppllicationConfigDAO")
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
