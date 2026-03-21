package hide_files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHideFolderExistingDir(t *testing.T) {
	dir := t.TempDir()
	err := HideFolder(dir)
	if err != nil {
		t.Fatalf("HideFolder on existing dir returned error: %v", err)
	}
}

func TestUnhideFolderExistingDir(t *testing.T) {
	dir := t.TempDir()
	err := UnhideFolder(dir)
	if err != nil {
		t.Fatalf("UnhideFolder on existing dir returned error: %v", err)
	}
}

func TestHideAndUnhideRoundTrip(t *testing.T) {
	dir := t.TempDir()

	err := HideFolder(dir)
	if err != nil {
		t.Fatalf("HideFolder error: %v", err)
	}

	err = UnhideFolder(dir)
	if err != nil {
		t.Fatalf("UnhideFolder error: %v", err)
	}

	// Directory should still be accessible after round-trip.
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat after round-trip error: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory after round-trip")
	}
}

func TestHideFolderNonExistentPath(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "does_not_exist")
	err := HideFolder(nonExistent)
	// On Windows, this should return an error (path does not exist).
	// On non-Windows, hideFolder is a no-op and returns nil.
	if err != nil {
		// This is expected on Windows.
		t.Logf("HideFolder on non-existent path returned expected error: %v", err)
	}
}

func TestUnhideFolderNonExistentPath(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "does_not_exist")
	err := UnhideFolder(nonExistent)
	// On Windows, this should return an error.
	// On non-Windows, unhideFolder is a no-op and returns nil.
	if err != nil {
		t.Logf("UnhideFolder on non-existent path returned expected error: %v", err)
	}
}

func TestHideUnhideIdempotent(t *testing.T) {
	dir := t.TempDir()

	// Hide twice should not error.
	if err := HideFolder(dir); err != nil {
		t.Fatalf("first HideFolder error: %v", err)
	}
	if err := HideFolder(dir); err != nil {
		t.Fatalf("second HideFolder error: %v", err)
	}

	// Unhide twice should not error.
	if err := UnhideFolder(dir); err != nil {
		t.Fatalf("first UnhideFolder error: %v", err)
	}
	if err := UnhideFolder(dir); err != nil {
		t.Fatalf("second UnhideFolder error: %v", err)
	}
}
