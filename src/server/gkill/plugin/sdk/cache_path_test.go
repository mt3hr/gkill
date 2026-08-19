package sdk

import (
	"os"
	"path/filepath"
	"testing"
)

// キャッシュDBは gkill のキャッシュディレクトリ配下に置く。
// ここを外すと `clear_cache plugin` が消せない場所にキャッシュが溜まる。
func TestCacheDBPath_UsesGkillCacheDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GKILL_HOME", home)

	pluginDir := filepath.Join(home, "plugins", "testuser", "gkill_plugin_example")
	got := CacheDBPath(pluginDir)
	want := filepath.Join(home, "caches", "plugin_cache", "testuser", "gkill_plugin_example", "cache.db")
	if got != want {
		t.Errorf("CacheDBPath = %q, want %q", got, want)
	}
	// SQLiteのコネクション取得は親ディレクトリを作ってくれないので、ここで作れていること
	if _, err := os.Stat(filepath.Dir(got)); err != nil {
		t.Errorf("キャッシュディレクトリが作られていない: %v", err)
	}
}

// gkill以外から手動起動したときは環境変数が無い。
// $GKILL_HOME/plugins/{userID}/{pluginName} の形なら遡って推定できる。
func TestCacheDBPath_InfersHomeFromPluginDirLayout(t *testing.T) {
	t.Setenv("GKILL_HOME", "")

	home := t.TempDir()
	pluginDir := filepath.Join(home, "plugins", "testuser", "gkill_plugin_example")
	got := CacheDBPath(pluginDir)
	want := filepath.Join(home, "caches", "plugin_cache", "testuser", "gkill_plugin_example", "cache.db")
	if got != want {
		t.Errorf("CacheDBPath = %q, want %q", got, want)
	}
}

// 想定と違う置き方をされていたらプラグインフォルダ直下へフォールバックする。
func TestCacheDBPath_FallsBackToPluginDir(t *testing.T) {
	t.Setenv("GKILL_HOME", "")

	pluginDir := filepath.Join(t.TempDir(), "どこか", "gkill_plugin_example")
	got := CacheDBPath(pluginDir)
	want := filepath.Join(pluginDir, "cache.db")
	if got != want {
		t.Errorf("CacheDBPath = %q, want %q", got, want)
	}
}

func TestCacheDBPath_EmptyPluginDir(t *testing.T) {
	t.Setenv("GKILL_HOME", "")

	if got := CacheDBPath(""); got != "cache.db" {
		t.Errorf(`CacheDBPath("") = %q, want cache.db`, got)
	}
}

// ユーザIDやプラグイン名はそのままパス要素として連結するので、
// 区切り文字や親ディレクトリ参照が混ざったものは通してはいけない。
func TestIsSafePathElement(t *testing.T) {
	for _, ok := range []string{"user1", "gkill_plugin_example", "a.b"} {
		if !IsSafePathElement(ok) {
			t.Errorf("%q を拒否している", ok)
		}
	}
	for _, ng := range []string{"", ".", "..", "a/b", `a\b`, "a/"} {
		if IsSafePathElement(ng) {
			t.Errorf("%q を通している", ng)
		}
	}
}
