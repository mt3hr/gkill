// ˅
package reps

import "context"

// ˄

type IDFKyouRepository interface {
	FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error)

	GetKyou(ctx context.Context, id string) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindIDFKyou(ctx context.Context, queryJSON string) ([]*IDFKyou, error)

	GetIDFKyou(ctx context.Context, id string) (*IDFKyou, error)

	GetIDFKyouHistories(ctx context.Context, id string) ([]*IDFKyou, error)

	// ˅

	IDF(ctx context.Context) error

	AddIDFKyouInfo(ctx context.Context, idfKyou *IDFKyou) error

	// ˄
}

// ˅

// ˄
