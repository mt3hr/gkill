// ˅
package reps

import "context"

// ˄

type LantanaRepository interface {
	FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error)

	GetKyou(ctx context.Context, id string) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindLantana(ctx context.Context, queryJSON string) ([]*Lantana, error)

	GetLantana(ctx context.Context, id string) (*Lantana, error)

	GetLantanaHistories(ctx context.Context, id string) ([]*Lantana, error)

	AddLantanaInfo(ctx context.Context, lantana *Lantana) error

	// ˅

	// ˄
}

// ˅

// ˄
