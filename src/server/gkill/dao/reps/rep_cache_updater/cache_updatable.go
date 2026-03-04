package rep_cache_updater

import "context"

type CacheUpdatable interface {
	UpdateCache(ctx context.Context) error
	GetRepName(ctx context.Context) (string, error)
	GetPath(ctx context.Context, id string) (string, error)
}
