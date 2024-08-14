// ˅
package reps

import "context"

// ˄

type Repository interface {
	FindKyous(ctx context.Context, queryJSON string) []*Kyou

	GetKyou(ctx context.Context, id string) *Kyou

	GetKyouHistories(ctx context.Context, id string) []*Kyou

	GetPath(ctx context.Context, id string) string

	GetRepName(ctx context.Context) string

	UpdateCache(ctx context.Context)

	Close(ctx context.Context)

	// ˅

	// ˄
}

// ˅

// ˄
