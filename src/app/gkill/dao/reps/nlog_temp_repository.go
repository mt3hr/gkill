package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
)

type NlogTempRepository interface {
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error)

	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindNlog(ctx context.Context, query *find.FindQuery) ([]*Nlog, error)

	GetNlog(ctx context.Context, id string, updateTime *time.Time) (*Nlog, error)

	GetNlogHistories(ctx context.Context, id string) ([]*Nlog, error)

	AddNlogInfo(ctx context.Context, nlog *Nlog, txID string, userID string, device string) error

	GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]*Kyou, error)

	GetNlogsByTXID(ctx context.Context, txID string, userID string, device string) ([]*Nlog, error)

	DeleteByTXID(ctx context.Context, txID string, userID string, device string) error
}
