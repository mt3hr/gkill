package main

import (
	"os"
	"path/filepath"
	"testing"
)

// GKILL_HOME があれば、gkillのキャッシュディレクトリ配下に置く
func TestCacheDBPath_UsesGkillCacheDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GKILL_HOME", home)

	pluginDir := filepath.Join(home, "plugins", "testuser", "gkill_plugin_google_locationhistory")
	got := cacheDBPath(pluginDir)

	want := filepath.Join(home, "caches", "plugin_cache", "testuser", "gkill_plugin_google_locationhistory", "cache.db")
	if got != want {
		t.Errorf("cacheDBPath = %q, want %q", got, want)
	}
	// SQLiteのコネクション取得は親ディレクトリを作らないので、ここで作られている必要がある
	if _, err := os.Stat(filepath.Dir(got)); err != nil {
		t.Errorf("expected cache dir created, stat err = %v", err)
	}
}

// GKILL_HOME が無くても plugins/{userID}/{pluginName} の形なら遡って推定する
func TestCacheDBPath_FallsBackToPluginDirLayout(t *testing.T) {
	t.Setenv("GKILL_HOME", "")

	home := t.TempDir()
	pluginDir := filepath.Join(home, "plugins", "testuser", "gkill_plugin_google_locationhistory")
	got := cacheDBPath(pluginDir)

	want := filepath.Join(home, "caches", "plugin_cache", "testuser", "gkill_plugin_google_locationhistory", "cache.db")
	if got != want {
		t.Errorf("cacheDBPath = %q, want %q", got, want)
	}
}

// GKILL_HOME が無く、想定外の構成なら従来どおりプラグインフォルダ直下に置く
func TestCacheDBPath_FallsBackToPluginDir(t *testing.T) {
	t.Setenv("GKILL_HOME", "")

	pluginDir := filepath.Join(t.TempDir(), "somewhere", "gkill_plugin_google_locationhistory")
	got := cacheDBPath(pluginDir)

	want := filepath.Join(pluginDir, "cache.db")
	if got != want {
		t.Errorf("cacheDBPath = %q, want %q", got, want)
	}
}

// pluginDir が空(手動起動)ならカレントディレクトリの cache.db
func TestCacheDBPath_EmptyPluginDir(t *testing.T) {
	t.Setenv("GKILL_HOME", t.TempDir())

	if got := cacheDBPath(""); got != "cache.db" {
		t.Errorf("cacheDBPath(\"\") = %q, want %q", got, "cache.db")
	}
}
