package reps

import (
	"context"
	"net/http"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/app/gkill/dao/reps/cache"
)

type IDFKyouRepository interface {
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error)

	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindIDFKyou(ctx context.Context, query *find.FindQuery) ([]*IDFKyou, error)

	GetIDFKyou(ctx context.Context, id string, updateTime *time.Time) (*IDFKyou, error)

	GetIDFKyouHistories(ctx context.Context, id string) ([]*IDFKyou, error)

	IDF(ctx context.Context) error

	AddIDFKyouInfo(ctx context.Context, idfKyou *IDFKyou) error

	HandleFileServe(w http.ResponseWriter, r *http.Request)

	GenerateThumbCache(ctx context.Context) error

	ClearThumbCache() error

	UnWrapTyped() ([]IDFKyouRepository, error)

	UnWrap() ([]Repository, error)

	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]*gkill_cache.LatestDataRepositoryAddress, error)
}
