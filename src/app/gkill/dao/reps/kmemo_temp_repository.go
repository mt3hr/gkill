package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
)

type KmemoTempRepository interface {
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error)

	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindKmemo(ctx context.Context, query *find.FindQuery) ([]*Kmemo, error)

	GetKmemo(ctx context.Context, id string, updateTime *time.Time) (*Kmemo, error)

	GetKmemoHistories(ctx context.Context, id string) ([]*Kmemo, error)

	AddKmemoInfo(ctx context.Context, kmemo *Kmemo, txID string, userID string, device string) error

	GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]*Kyou, error)

	GetKmemosByTXID(ctx context.Context, txID string, userID string, device string) ([]*Kmemo, error)

	DeleteByTXID(ctx context.Context, txID string, userID string, device string) error

	UnWrapTyped() ([]KmemoTempRepository, error)

	UnWrap() ([]Repository, error)
}
