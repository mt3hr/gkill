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

	// Add owner with skip=true, remove first
	entry.removeOwner("owner1")
	skipTrue := true
	entry.addOwner("owner2", nil, &skipTrue, nil)
	if !entry.shouldSkipAll() {
		t.Error("expected shouldSkipAll=true when all owners skip")
	}
}

func TestWatchTargetEntry_RemoveOwner(t *testing.T) {
	entry := newWatchTargetEntry("/test", nil)
	skipFalse := false
	entry.addOwner("owner1", nil, &skipFalse, nil)
	entry.addOwner("owner2", nil, &skipFalse, nil)

	empty := entry.removeOwner("owner1")
	if empty {
		t.Error("expected non-empty after removing one of two owners")
	}

	empty = entry.removeOwner("owner2")
	if !empty {
		t.Error("expected empty after removing last owner")
	}
}

func TestNormalizeKey(t *testing.T) {
	// normalizeKey cleans and converts to slash form
	// Test that path cleaning works (.. removal)
	got := normalizeKey("some/path/../path/file")
	if got != "some/path/file" {
		t.Errorf("normalizeKey(\"some/path/../path/file\") = %q, want %q", got, "some/path/file")
	}

	// Test idempotence
	input := "a/b/c"
	if normalizeKey(input) != normalizeKey(normalizeKey(input)) {
		t.Error("normalizeKey is not idempotent")
	}
}
