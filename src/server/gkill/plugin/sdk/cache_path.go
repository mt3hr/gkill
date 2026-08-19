package sdk

import (
	"os"
	"path/filepath"
	"strings"
)

// CacheDBPath はプラグインのキャッシュDBの置き場所を返す。
//
// gkillの他の派生キャッシュ(thumb_cache, git_commit_log_cache など)と同じく
// gkillのキャッシュディレクトリ配下に置く:
//
//	$GKILL_HOME/caches/plugin_cache/{userID}/{pluginName}/cache.db
//
// ここに置くことで、gkill の `clear_cache plugin` がまとめて消せる。
//
// 置き場所を解決できないとき(gkill以外から手動起動したときなど)は、
// プラグインフォルダ直下にフォールバックする。
//
// 同梱プラグイン6本が1文字違わず同じものを持っていたのでSDKへ移した。
func CacheDBPath(pluginDir string) string {
	dir := PluginCacheDir(pluginDir)
	if dir == "" {
		return filepath.Join(pluginDir, "cache.db")
	}
	// SQLiteのコネクション取得は親ディレクトリを作ってくれないのでここで作る
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return filepath.Join(pluginDir, "cache.db")
	}
	return filepath.Join(dir, "cache.db")
}

// PluginCacheDir は $GKILL_HOME/caches/plugin_cache/{userID}/{pluginName} を返す。
//
// pluginDir は $GKILL_HOME/plugins/{userID}/{pluginName} の形をしているので、
// 末尾2要素からユーザIDとプラグイン名を取り出す。
// gkillの本体は起動時に環境変数 GKILL_HOME を設定し、プラグインのプロセスは
// その環境を引き継ぐので、キャッシュルートは環境変数から取れる。
// 環境変数が無いときだけ pluginDir から遡って推定する。
// 想定と違う構成なら "" を返す(呼び出し側がフォールバックする)。
func PluginCacheDir(pluginDir string) string {
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
	if home == "" || !IsSafePathElement(userID) || !IsSafePathElement(pluginName) {
		return ""
	}
	return filepath.Join(filepath.Clean(home), "caches", "plugin_cache", userID, pluginName)
}

// IsSafePathElement は値を単一のパス要素として使ってよいか検証する。
// 区切り文字・親ディレクトリ参照・空文字を含むものを拒否する。
//
// gkill本体側にも同じ判定が dao/plugin_manager.go の isSingleSafePathElement としてある。
// あちらは別モジュール(src/server)からプラグインの置き場所を組み立てるためのもので、
// SDKへの依存を本体に持ち込みたくないので意図的に分けてある。直すときは両方。
func IsSafePathElement(element string) bool {
	if element == "" || element == "." || element == ".." {
		return false
	}
	if strings.ContainsAny(element, `/\`) {
		return false
	}
	return filepath.Clean(element) == element
}
