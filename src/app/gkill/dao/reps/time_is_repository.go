package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
)

type TimeIsRepository interface {
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error)

	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindTimeIs(ctx context.Context, query *find.FindQuery) ([]*TimeIs, error)

	GetTimeIs(ctx context.Context, id string, updateTime *time.Time) (*TimeIs, error)

	GetTimeIsHistories(ctx context.Context, id string) ([]*TimeIs, error)

	AddTimeIsInfo(ctx context.Context, timeis *TimeIs) error
}
