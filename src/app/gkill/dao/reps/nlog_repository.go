package reps

import (
	"context"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
)

type NlogRepository interface {
	FindKyous(ctx context.Context, query *find.FindQuery) ([]*Kyou, error)

	GetKyou(ctx context.Context, id string) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindNlog(ctx context.Context, query *find.FindQuery) ([]*Nlog, error)

	GetNlog(ctx context.Context, id string) (*Nlog, error)

	GetNlogHistories(ctx context.Context, id string) ([]*Nlog, error)

	AddNlogInfo(ctx context.Context, nlog *Nlog) error
}
