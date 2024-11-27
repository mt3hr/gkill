package reps

import (
	"context"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
)

type IDFKyouRepository interface {
	FindKyous(ctx context.Context, query *find.FindQuery) ([]*Kyou, error)

	GetKyou(ctx context.Context, id string) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindIDFKyou(ctx context.Context, query *find.FindQuery) ([]*IDFKyou, error)

	GetIDFKyou(ctx context.Context, id string) (*IDFKyou, error)

	GetIDFKyouHistories(ctx context.Context, id string) ([]*IDFKyou, error)

	IDF(ctx context.Context) error

	AddIDFKyouInfo(ctx context.Context, idfKyou *IDFKyou) error
}
