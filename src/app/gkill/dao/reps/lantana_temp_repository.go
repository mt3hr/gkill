package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/app/gkill/dao/reps/cache"
)

type LantanaTempRepository interface {
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error)

	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindLantana(ctx context.Context, query *find.FindQuery) ([]*Lantana, error)

	GetLantana(ctx context.Context, id string, updateTime *time.Time) (*Lantana, error)

	GetLantanaHistories(ctx context.Context, id string) ([]*Lantana, error)

	AddLantanaInfo(ctx context.Context, lantana *Lantana, txID string, userID string, device string) error

	GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]*Kyou, error)

	GetLantanasByTXID(ctx context.Context, txID string, userID string, device string) ([]*Lantana, error)

	DeleteByTXID(ctx context.Context, txID string, userID string, device string) error

	UnWrapTyped() ([]LantanaTempRepository, error)

	UnWrap() ([]Repository, error)

	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]*gkill_cache.LatestDataRepositoryAddress, error)
}
