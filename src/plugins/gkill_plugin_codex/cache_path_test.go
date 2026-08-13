package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCacheDBPathUsesGkillHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GKILL_HOME", home)

	pluginDir := filepath.Join(home, "plugins", "testuser", "gkill_plugin_codex")
	got := cacheDBPath(pluginDir)
	want := filepath.Join(home, "caches", "plugin_cache", "testuser", "gkill_plugin_codex", "cache.db")
	if got != want {
		t.Errorf("cacheDBPath = %q, want %q", got, want)
	}
	// SQLiteのコネクション取得は親ディレクトリを作ってくれないのでここで作る
	if _, err := os.Stat(filepath.Dir(got)); err != nil {
		t.Errorf("キャッシュディレクトリが作られていない: %v", err)
	}
}

func TestCacheDBPathInfersHomeFromPluginDir(t *testing.T) {
	// gkill以外から手動起動したときは環境変数が無い。
	// $GKILL_HOME/plugins/{userID}/{pluginName} の形なら遡って推定できる。
	t.Setenv("GKILL_HOME", "")

	home := t.TempDir()
	pluginDir := filepath.Join(home, "plugins", "testuser", "gkill_plugin_codex")
	got := cacheDBPath(pluginDir)
	want := filepath.Join(home, "caches", "plugin_cache", "testuser", "gkill_plugin_codex", "cache.db")
	if got != want {
		t.Errorf("cacheDBPath = %q, want %q", got, want)
	}
}

func TestCacheDBPathFallsBackOnUnexpectedLayout(t *testing.T) {
	t.Setenv("GKILL_HOME", "")

	pluginDir := filepath.Join(t.TempDir(), "どこか", "gkill_plugin_codex")
	got := cacheDBPath(pluginDir)
	want := filepath.Join(pluginDir, "cache.db")
	if got != want {
		t.Errorf("cacheDBPath = %q, want %q", got, want)
	}
}

func TestCacheDBPathWithEmptyPluginDir(t *testing.T) {
	t.Setenv("GKILL_HOME", "")

	if got := cacheDBPath(""); got != "cache.db" {
		t.Errorf("cacheDBPath(\"\") = %q, want cache.db", got)
	}
}

func TestIsSafePathElement(t *testing.T) {
	for _, ok := range []string{"user1", "gkill_plugin_codex", "a.b"} {
		if !isSafePathElement(ok) {
			t.Errorf("%q を拒否している", ok)
		}
	}
	for _, ng := range []string{"", ".", "..", "a/b", `a\b`, "a/"} {
		if isSafePathElement(ng) {
			t.Errorf("%q を通している", ng)
		}
	}
}
