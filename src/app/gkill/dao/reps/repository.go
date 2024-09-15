package reps

import "context"

type Repository interface {
	FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error)

	GetKyou(ctx context.Context, id string) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	GetRepName(ctx context.Context) (string, error)

	UpdateCache(ctx context.Context) error

	Close(ctx context.Context) error
}
