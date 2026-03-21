package rep_cache_updater

import (
	"testing"
)

func TestNewFileRepCacheUpdater(t *testing.T) {
	skip := false
	updater, err := NewFileRepCacheUpdater(&skip)
	if err != nil {
		t.Fatalf("NewFileRepCacheUpdater: %v", err)
	}
	if updater == nil {
		t.Fatal("expected non-nil updater")
	}
}

func TestFileRepCacheUpdater_Close(t *testing.T) {
	skip := false
	updater, err := NewFileRepCacheUpdater(&skip)
	if err != nil {
		t.Fatalf("NewFileRepCacheUpdater: %v", err)
	}
	if err := updater.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestNewWatchTargetEntry(t *testing.T) {
	entry := newWatchTargetEntry("/some/path/file.db", []string{"prefix1", "prefix2"})
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.filename != "/some/path/file.db" {
		t.Errorf("filename = %q, want %q", entry.filename, "/some/path/file.db")
	}
	if len(entry.ignorePrefixes) != 2 {
		t.Errorf("ignorePrefixes len = %d, want 2", len(entry.ignorePrefixes))
	}
}

func TestWatchTargetEntry_ShouldSkipAll(t *testing.T) {
	entry := newWatchTargetEntry("/test", nil)

	// No owners => should skip all
	if !entry.shouldSkipAll() {
		t.Error("expected shouldSkipAll=true with no owners")
	}

	// Add owner with skip=false
	skipFalse := false
	entry.addOwner("owner1", nil, &skipFalse, nil)
	if entry.shouldSkipAll() {
		t.Error("expected shouldSkipAll=false with skip=false owner")
	}
}
