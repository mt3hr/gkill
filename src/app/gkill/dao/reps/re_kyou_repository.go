// ˅
package reps

import "context"

// ˄

type ReKyouRepository interface {
	FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error)

	GetKyou(ctx context.Context, id string) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindReKyou(ctx context.Context, queryJSON string) ([]*ReKyou, error)

	GetReKyou(ctx context.Context, id string) (*ReKyou, error)

	GetReKyouHistories(ctx context.Context, id string) ([]*ReKyou, error)

	AddReKyouInfo(ctx context.Context, rekyou *ReKyou) error

	GetReKyousAllLatest(ctx context.Context) ([]*ReKyou, error)

	GetRepositories(ctx context.Context) (*GkillRepositories, error)

	// ˅

	// ˄
}

// ˅

// ˄
