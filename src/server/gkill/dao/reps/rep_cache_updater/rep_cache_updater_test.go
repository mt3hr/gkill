package rep_cache_updater

import (
	"sync/atomic"
	"testing"
)

func TestNewFileRepCacheUpdater(t *testing.T) {
	skip := &atomic.Int64{}
	updater, err := NewFileRepCacheUpdater(skip)
	if err != nil {
		t.Fatalf("NewFileRepCacheUpdater: %v", err)
	}
	if updater == nil {
		t.Fatal("expected non-nil updater")
	}
}

func TestFileRepCacheUpdater_Close(t *testing.T) {
	skip := &atomic.Int64{}
	updater, err := NewFileRepCacheUpdater(skip)
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

	// Add owner with skip count 0 (=not skipping)
	skipZero := &atomic.Int64{}
	entry.addOwner("owner1", nil, skipZero, nil)
	if entry.shouldSkipAll() {
		t.Error("expected shouldSkipAll=false with skip count 0 owner")
	}

	// Add owner with skip count >0, remove first
	entry.removeOwner("owner1")
	skipPositive := &atomic.Int64{}
	skipPositive.Store(1)
	entry.addOwner("owner2", nil, skipPositive, nil)
	if !entry.shouldSkipAll() {
		t.Error("expected shouldSkipAll=true when all owners skip")
	}
}

func TestWatchTargetEntry_RemoveOwner(t *testing.T) {
	entry := newWatchTargetEntry("/test", nil)
	skipZero := &atomic.Int64{}
	entry.addOwner("owner1", nil, skipZero, nil)
	entry.addOwner("owner2", nil, skipZero, nil)

	empty := entry.removeOwner("owner1")
	if empty {
		t.Error("expected non-empty after removing one of two owners")
	}

	empty = entry.removeOwner("owner2")
	if !empty {
		t.Error("expected empty after removing last owner")
	}
}

// TestWatchTargetEntry_SharedSkipCounterOverlap は、共有の参照カウンタで
// 重なる Pause/Resume を表したとき、外側が生きている間は skip 継続、
// 全 Resume でのみ再開されることを entry レベルで確認する。
func TestWatchTargetEntry_SharedSkipCounterOverlap(t *testing.T) {
	entry := newWatchTargetEntry("/test", nil)
	skip := &atomic.Int64{}
	entry.addOwner("owner1", nil, skip, nil)

	// カウント0: 再開中
	if entry.shouldSkipAll() {
		t.Error("expected shouldSkipAll=false at count 0")
	}

	// 外側 Pause
	skip.Add(1)
	if !entry.shouldSkipAll() {
		t.Error("expected shouldSkipAll=true after first pause")
	}

	// 内側 Pause (重なる)
	skip.Add(1)
	// 内側 Resume: まだ外側が生きている
	skip.Add(-1)
	if !entry.shouldSkipAll() {
		t.Error("expected shouldSkipAll=true while outer pause still alive")
	}

	// 外側 Resume: 全 Resume で再開
	skip.Add(-1)
	if entry.shouldSkipAll() {
		t.Error("expected shouldSkipAll=false after all resume")
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
