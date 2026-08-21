// Package rep_cache_updater はリポジトリキャッシュの定期更新処理(代替パス)。
package rep_cache_updater

type FileRepCacheUpdater interface {
	RegisterWatchFileRep(rep CacheUpdatable, filename string, ignoreFilePrefixes []string, userID string) error
	RemoveWatchFileRep(filename string, userID string) error
	// KickAllPendingUpdates は、skip(一時停止)中に握りつぶしたファイル変更の更新走査を再開時に1回だけキックする。
	// 取りこぼし防止のcatch-up用で、SetSkipIDFがカウント0へ戻ったときに呼ぶ。
	KickAllPendingUpdates()
	Close() error
}
