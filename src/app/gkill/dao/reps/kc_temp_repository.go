package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/app/gkill/dao/reps/cache"
)

type KCTempRepository interface {
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error)

	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindKC(ctx context.Context, query *find.FindQuery) ([]*KC, error)

	GetKC(ctx context.Context, id string, updateTime *time.Time) (*KC, error)

	GetKCHistories(ctx context.Context, id string) ([]*KC, error)

	AddKCInfo(ctx context.Context, kc *KC, txID string, userID string, device string) error

	GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]*Kyou, error)

	GetKCsByTXID(ctx context.Context, txID string, userID string, device string) ([]*KC, error)

	DeleteByTXID(ctx context.Context, txID string, userID string, device string) error

	UnWrapTyped() ([]KCTempRepository, error)

	UnWrap() ([]Repository, error)

	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]*gkill_cache.LatestDataRepositoryAddress, error)
}
