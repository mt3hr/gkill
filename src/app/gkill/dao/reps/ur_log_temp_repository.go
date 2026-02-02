package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/app/gkill/dao/reps/cache"
)

type URLogTempRepository interface {
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error)

	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindURLog(ctx context.Context, query *find.FindQuery) ([]*URLog, error)

	GetURLog(ctx context.Context, id string, updateTime *time.Time) (*URLog, error)

	GetURLogHistories(ctx context.Context, id string) ([]*URLog, error)

	AddURLogInfo(ctx context.Context, urlog *URLog, txID string, userID string, device string) error

	GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]*Kyou, error)

	GetURLogsByTXID(ctx context.Context, txID string, userID string, device string) ([]*URLog, error)

	DeleteByTXID(ctx context.Context, txID string, userID string, device string) error

	UnWrapTyped() ([]URLogTempRepository, error)

	UnWrap() ([]Repository, error)

	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]*gkill_cache.LatestDataRepositoryAddress, error)
}
