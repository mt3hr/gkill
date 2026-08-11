package main

import (
	"os"
	"path/filepath"
	"strings"
)

// cacheDBPath はキャッシュDBの置き場所を返す。
//
// gkillの他の派生キャッシュ(thumb_cache, git_commit_log_cache など)と同じく
// gkillのキャッシュディレクトリ配下に置く:
//
//	$GKILL_HOME/caches/plugin_cache/{userID}/{pluginName}/cache.db
//
// 置き場所を解決できないとき(gkill以外から手動起動したときなど)は、
// 従来どおりプラグインフォルダ直下にフォールバックする。
func cacheDBPath(pluginDir string) string {
	dir := pluginCacheDir(pluginDir)
	if dir == "" {
		return filepath.Join(pluginDir, "cache.db")
	}
	// SQLiteのコネクション取得は親ディレクトリを作ってくれないのでここで作る
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return filepath.Join(pluginDir, "cache.db")
	}
	return filepath.Join(dir, "cache.db")
}

// pluginCacheDir は $GKILL_HOME/caches/plugin_cache/{userID}/{pluginName} を返す。
//
// pluginDir は $GKILL_HOME/plugins/{userID}/{pluginName} の形をしているので、
// 末尾2要素からユーザIDとプラグイン名を取り出す。
// gkillの本体は起動時に環境変数 GKILL_HOME を設定し、プラグインのプロセスは
// その環境を引き継ぐので、キャッシュルートは環境変数から取れる。
// 環境変数が無いときだけ pluginDir から遡って推定する。
// 想定と違う構成なら "" を返す(呼び出し側がフォールバックする)。
func pluginCacheDir(pluginDir string) string {
	if pluginDir == "" {
		return ""
	}
	cleaned := filepath.Clean(pluginDir)
	pluginName := filepath.Base(cleaned)
	userDir := filepath.Dir(cleaned)
	userID := filepath.Base(userDir)

	home := os.Getenv("GKILL_HOME")
	if home == "" {
		pluginsDir := filepath.Dir(userDir)
		if filepath.Base(pluginsDir) != "plugins" {
			return ""
		}
		home = filepath.Dir(pluginsDir)
	}
	if home == "" || !isSafePathElement(userID) || !isSafePathElement(pluginName) {
		return ""
	}
	return filepath.Join(filepath.Clean(home), "caches", "plugin_cache", userID, pluginName)
}

// isSafePathElement は値を単一のパス要素として使ってよいか検証する。
// 区切り文字・親ディレクトリ参照・空文字を含むものを拒否する。
func isSafePathElement(element string) bool {
	if element == "" || element == "." || element == ".." {
		return false
	}
	if strings.ContainsAny(element, `/\`) {
		return false
	}
	return filepath.Clean(element) == element
}
