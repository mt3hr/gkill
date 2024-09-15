package reps

import "context"

type KmemoRepository interface {
	FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error)

	GetKyou(ctx context.Context, id string) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindKmemo(ctx context.Context, queryJSON string) ([]*Kmemo, error)

	GetKmemo(ctx context.Context, id string) (*Kmemo, error)

	GetKmemoHistories(ctx context.Context, id string) ([]*Kmemo, error)

	AddKmemoInfo(ctx context.Context, kmemo *Kmemo) error
}
