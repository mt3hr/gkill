package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/app/gkill/dao/reps/cache"
)

type TextRepository interface {
	FindTexts(ctx context.Context, query *find.FindQuery) ([]*Text, error)

	Close(ctx context.Context) error

	GetText(ctx context.Context, id string, updateTime *time.Time) (*Text, error)

	GetTextsByTargetID(ctx context.Context, target_id string) ([]*Text, error)

	UpdateCache(ctx context.Context) error

	GetPath(ctx context.Context, id string) (string, error)

	GetRepName(ctx context.Context) (string, error)

	GetTextHistories(ctx context.Context, id string) ([]*Text, error)

	AddTextInfo(ctx context.Context, text *Text) error

	UnWrapTyped() ([]TextRepository, error)

	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]*gkill_cache.LatestDataRepositoryAddress, error)
}
