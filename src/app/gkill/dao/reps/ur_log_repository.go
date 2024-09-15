package reps

import "context"

type URLogRepository interface {
	FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error)

	GetKyou(ctx context.Context, id string) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindURLog(ctx context.Context, queryJSON string) ([]*URLog, error)

	GetURLog(ctx context.Context, id string) (*URLog, error)

	GetURLogHistories(ctx context.Context, id string) ([]*URLog, error)

	AddURLogInfo(ctx context.Context, urlog *URLog) error
}
