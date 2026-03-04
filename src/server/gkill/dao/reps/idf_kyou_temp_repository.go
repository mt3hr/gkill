package reps

import (
	"context"
	"net/http"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
)

type IDFKyouTempRepository interface {
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error)

	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindIDFKyou(ctx context.Context, query *find.FindQuery) ([]IDFKyou, error)

	GetIDFKyou(ctx context.Context, id string, updateTime *time.Time) (*IDFKyou, error)

	GetIDFKyouHistories(ctx context.Context, id string) ([]IDFKyou, error)

	IDF(ctx context.Context) error

	AddIDFKyouInfo(ctx context.Context, idfKyou IDFKyou, txID string, userID string, device string) error

	HandleFileServe(w http.ResponseWriter, r *http.Request)

	GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]Kyou, error)

	GetIDFKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]IDFKyou, error)

	DeleteByTXID(ctx context.Context, txID string, userID string, device string) error

	GenerateThumbCache(ctx context.Context) error

	ClearThumbCache() error

	GenerateVideoCache(ctx context.Context) error

	ClearVideoCache() error

	UnWrapTyped() ([]IDFKyouTempRepository, error)

	UnWrap() ([]Repository, error)
}
