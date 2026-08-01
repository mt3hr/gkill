package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

type MiReKyouTempRepository interface {
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error)

	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error)

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindMiReKyou(ctx context.Context, query *find.FindQuery) ([]MiReKyou, error)

	GetMiReKyou(ctx context.Context, id string, updateTime *time.Time) (*MiReKyou, error)

	GetMiReKyouHistories(ctx context.Context, id string) ([]MiReKyou, error)

	AddMiReKyouInfo(ctx context.Context, mirekyou MiReKyou, txID string, userID string, device string) error

	GetMiReKyousAllLatest(ctx context.Context) ([]MiReKyou, error)

	GetBoardNames(ctx context.Context) ([]string, error)

	GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]Kyou, error)

	GetMiReKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]MiReKyou, error)

	DeleteByTXID(ctx context.Context, txID string, userID string, device string) error

	UnWrapTyped() ([]MiReKyouTempRepository, error)

	UnWrap() ([]Repository, error)
}
