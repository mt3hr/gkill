// ˅
package reps

import "context"

// ˄

type MiRepository interface {
	FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error)

	GetKyou(ctx context.Context, id string) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindMi(ctx context.Context, queryJSON string) ([]*Mi, error)

	GetMi(ctx context.Context, id string) (*Mi, error)

	GetMiHistories(ctx context.Context, id string) ([]*Mi, error)

	AddMiInfo(ctx context.Context, mi *Mi) error

	GetBoardNames(ctx context.Context) ([]string, error)

	// ˅

	// ˄
}

// ˅

// ˄
