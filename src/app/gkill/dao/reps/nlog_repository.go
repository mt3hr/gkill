package reps

import "context"

type NlogRepository interface {
	FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error)

	GetKyou(ctx context.Context, id string) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindNlog(ctx context.Context, queryJSON string) ([]*Nlog, error)

	GetNlog(ctx context.Context, id string) (*Nlog, error)

	GetNlogHistories(ctx context.Context, id string) ([]*Nlog, error)

	AddNlogInfo(ctx context.Context, nlog *Nlog) error
}
