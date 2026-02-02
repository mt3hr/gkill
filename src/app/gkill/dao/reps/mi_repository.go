package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/app/gkill/dao/reps/cache"
)

type MiRepository interface {
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error)

	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindMi(ctx context.Context, query *find.FindQuery) ([]*Mi, error)

	GetMi(ctx context.Context, id string, updateTime *time.Time) (*Mi, error)

	GetMiHistories(ctx context.Context, id string) ([]*Mi, error)

	AddMiInfo(ctx context.Context, mi *Mi) error

	GetBoardNames(ctx context.Context) ([]string, error)

	UnWrapTyped() ([]MiRepository, error)

	UnWrap() ([]Repository, error)

	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]*gkill_cache.LatestDataRepositoryAddress, error)
}
