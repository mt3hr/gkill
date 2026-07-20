package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

func TestAppNameDefault(t *testing.T) {
	if AppName == "" {
		t.Error("AppName should not be empty")
	}
}

func TestIDFCmdNotNil(t *testing.T) {
	if IDFCmd == nil {
		t.Fatal("IDFCmd should not be nil")
	}
	if IDFCmd.Use != "idf" {
		t.Errorf("IDFCmd.Use = %q, want %q", IDFCmd.Use, "idf")
	}
}

func TestDVNFCmdNotNil(t *testing.T) {
	if DVNFCmd == nil {
		t.Fatal("DVNFCmd should not be nil")
	}
}

func TestVersionCommandNotNil(t *testing.T) {
	if VersionCommand == nil {
		t.Fatal("VersionCommand should not be nil")
	}
	if VersionCommand.Use != "version" {
		t.Errorf("VersionCommand.Use = %q, want %q", VersionCommand.Use, "version")
	}
}

func TestGenerateThumbCacheCmdNotNil(t *testing.T) {
	if GenerateThumbCacheCmd == nil {
		t.Fatal("GenerateThumbCacheCmd should not be nil")
	}
}

func TestGenerateVideoCacheCmdNotNil(t *testing.T) {
	if GenerateVideoCacheCmd == nil {
		t.Fatal("GenerateVideoCacheCmd should not be nil")
	}
}

func TestOptimizeCmdNotNil(t *testing.T) {
	if OptimizeCmd == nil {
		t.Fatal("OptimizeCmd should not be nil")
	}
}

func TestUpdateCacheCmdNotNil(t *testing.T) {
	if UpdateCacheCmd == nil {
		t.Fatal("UpdateCacheCmd should not be nil")
	}
}

func TestClearCacheCmdNotNil(t *testing.T) {
	if ClearCacheCmd == nil {
		t.Fatal("ClearCacheCmd should not be nil")
	}
	if ClearCacheCmd.Use != "clear_cache" {
		t.Errorf("ClearCacheCmd.Use = %q, want %q", ClearCacheCmd.Use, "clear_cache")
	}
}

// clear_cache all はディスク上の派生キャッシュ3種を全削除する
func TestClearCacheCmd_All_RemovesAllDirs(t *testing.T) {
	origCacheDir := gkill_options.CacheDir
	t.Cleanup(func() { gkill_options.CacheDir = origCacheDir })

	tmpDir := t.TempDir()
	gkill_options.CacheDir = tmpDir

	cacheNames := []string{"thumb_cache", "video_cache", "zip_cache"}
	for _, name := range cacheNames {
		dir := filepath.Join(tmpDir, name)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "dummy"), []byte("x"), os.ModePerm); err != nil {
			t.Fatalf("write dummy in %s: %v", dir, err)
		}
	}

	ClearCacheCmd.Run(ClearCacheCmd, []string{"all", "all"})

	for _, name := range cacheNames {
		dir := filepath.Join(tmpDir, name)
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("expected %s removed, stat err = %v", dir, err)
		}
	}
}

// clear_cache <mode> は指定した1種のみ削除し、他は残す
func TestClearCacheCmd_SingleMode_LeavesOthers(t *testing.T) {
	origCacheDir := gkill_options.CacheDir
	t.Cleanup(func() { gkill_options.CacheDir = origCacheDir })

	tmpDir := t.TempDir()
	gkill_options.CacheDir = tmpDir

	cacheNames := []string{"thumb_cache", "video_cache", "zip_cache"}
	for _, name := range cacheNames {
		if err := os.MkdirAll(filepath.Join(tmpDir, name), os.ModePerm); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
	}

	ClearCacheCmd.Run(ClearCacheCmd, []string{"thumb", "all"})

	if _, err := os.Stat(filepath.Join(tmpDir, "thumb_cache")); !os.IsNotExist(err) {
		t.Errorf("expected thumb_cache removed, stat err = %v", err)
	}
	for _, name := range []string{"video_cache", "zip_cache"} {
		if _, err := os.Stat(filepath.Join(tmpDir, name)); err != nil {
			t.Errorf("expected %s to remain, stat err = %v", name, err)
		}
	}
}
