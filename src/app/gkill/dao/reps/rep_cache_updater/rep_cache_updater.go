package rep_cache_updater

type FileRepCacheUpdater interface {
	RegisterWatchFileRep(rep CacheUpdatable, filename string, ignoreFilePrefixes []string, userID string) error
	RemoveWatchFileRep(filename string, userID string) error
	Close() error
}
