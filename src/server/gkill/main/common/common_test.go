package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	"github.com/spf13/cobra"
)

// TestSubcommandsUseRunE は、このパッケージが提供する全サブコマンドが Run ではなく RunE を持つことを固定する。
// Run(戻り値なし)だと内部で失敗しても exit code に出ず、SyncDatas 等の呼び出し側が失敗を観測できない。
// RunE で error を返し、cobra の Execute() 経由で log.Fatal(exit 1)へ伝わるようにする。
// あわせて、usageの二重印字(SilenceUsage)とエラーの二重印字(SilenceErrors)を止めていることも確認する。
func TestSubcommandsUseRunE(t *testing.T) {
	cmds := map[string]*cobra.Command{
		"idf":                  IDFCmd,
		"version":              VersionCommand,
		"generate_thumb_cache": GenerateThumbCacheCmd,
		"generate_video_cache": GenerateVideoCacheCmd,
		"clear_cache":          ClearCacheCmd,
		"optimize":             OptimizeCmd,
		"update_cache":         UpdateCacheCmd,
		"reset_password":       ResetPasswordCmd,
		"auto_tag":             AutoTagCmd,
	}
	for name, cmd := range cmds {
		if cmd == nil {
			t.Errorf("%s: command is nil", name)
			continue
		}
		if cmd.RunE == nil {
			t.Errorf("%s: RunE is nil (must return an error so exit code propagates)", name)
		}
		if cmd.Run != nil {
			t.Errorf("%s: Run is set (must use RunE, not Run)", name)
		}
		if !cmd.SilenceUsage {
			t.Errorf("%s: SilenceUsage must be true (avoid reprinting usage on error)", name)
		}
		if !cmd.SilenceErrors {
			t.Errorf("%s: SilenceErrors must be true (main's log.Fatal is the single error printer)", name)
		}
	}
}

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

	cacheNames := []string{"thumb_cache", "video_cache", "zip_cache", "plugin_cache"}
	for _, name := range cacheNames {
		dir := filepath.Join(tmpDir, name)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "dummy"), []byte("x"), os.ModePerm); err != nil {
			t.Fatalf("write dummy in %s: %v", dir, err)
		}
	}

	if err := ClearCacheCmd.RunE(ClearCacheCmd, []string{"all", "all"}); err != nil {
		t.Fatalf("clear_cache all all: %v", err)
	}

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

	cacheNames := []string{"thumb_cache", "video_cache", "zip_cache", "plugin_cache"}
	for _, name := range cacheNames {
		if err := os.MkdirAll(filepath.Join(tmpDir, name), os.ModePerm); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
	}

	if err := ClearCacheCmd.RunE(ClearCacheCmd, []string{"thumb", "all"}); err != nil {
		t.Fatalf("clear_cache thumb all: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "thumb_cache")); !os.IsNotExist(err) {
		t.Errorf("expected thumb_cache removed, stat err = %v", err)
	}
	for _, name := range []string{"video_cache", "zip_cache", "plugin_cache"} {
		if _, err := os.Stat(filepath.Join(tmpDir, name)); err != nil {
			t.Errorf("expected %s to remain, stat err = %v", name, err)
		}
	}
}

// clear_cache plugin all はプラグインキャッシュだけを消し、他は残す
func TestClearCacheCmd_Plugin_RemovesOnlyPluginCache(t *testing.T) {
	origCacheDir := gkill_options.CacheDir
	t.Cleanup(func() { gkill_options.CacheDir = origCacheDir })

	tmpDir := t.TempDir()
	gkill_options.CacheDir = tmpDir

	cacheNames := []string{"thumb_cache", "video_cache", "zip_cache", "plugin_cache"}
	for _, name := range cacheNames {
		if err := os.MkdirAll(filepath.Join(tmpDir, name), os.ModePerm); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
	}

	if err := ClearCacheCmd.RunE(ClearCacheCmd, []string{"plugin", "all"}); err != nil {
		t.Fatalf("clear_cache plugin all: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "plugin_cache")); !os.IsNotExist(err) {
		t.Errorf("expected plugin_cache removed, stat err = %v", err)
	}
	for _, name := range []string{"thumb_cache", "video_cache", "zip_cache"} {
		if _, err := os.Stat(filepath.Join(tmpDir, name)); err != nil {
			t.Errorf("expected %s to remain, stat err = %v", name, err)
		}
	}
}

// clear_cache plugin <user_id> は指定ユーザーのディレクトリだけを消す
func TestClearPluginCache_RemovesOnlyTargetUser(t *testing.T) {
	origCacheDir := gkill_options.CacheDir
	t.Cleanup(func() { gkill_options.CacheDir = origCacheDir })

	tmpDir := t.TempDir()
	gkill_options.CacheDir = tmpDir

	pluginCacheRootDir := filepath.Join(tmpDir, "plugin_cache")
	targetUserDir := filepath.Join(pluginCacheRootDir, "user1", "gkill_plugin_claudecode")
	otherUserDir := filepath.Join(pluginCacheRootDir, "user2", "gkill_plugin_claudecode")
	for _, dir := range []string{targetUserDir, otherUserDir} {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "cache.db"), []byte("x"), os.ModePerm); err != nil {
			t.Fatalf("write cache.db in %s: %v", dir, err)
		}
	}

	if err := ClearPluginCache("user1"); err != nil {
		t.Fatalf("ClearPluginCache: %v", err)
	}

	if _, err := os.Stat(filepath.Join(pluginCacheRootDir, "user1")); !os.IsNotExist(err) {
		t.Errorf("expected user1 plugin cache removed, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(otherUserDir, "cache.db")); err != nil {
		t.Errorf("expected user2 plugin cache to remain, stat err = %v", err)
	}
}

// パス要素として使えないユーザーIDはエラーにする(キャッシュルート外を消させない)
func TestClearPluginCache_RejectsUnsafeUserID(t *testing.T) {
	origCacheDir := gkill_options.CacheDir
	t.Cleanup(func() { gkill_options.CacheDir = origCacheDir })
	gkill_options.CacheDir = t.TempDir()

	for _, userID := range []string{"", ".", "..", "../other", `a\b`, "a/b"} {
		if err := ClearPluginCache(userID); err == nil {
			t.Errorf("ClearPluginCache(%q) = nil, want error", userID)
		}
	}
}
