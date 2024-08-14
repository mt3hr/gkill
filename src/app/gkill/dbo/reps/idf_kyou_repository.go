// ˅
package reps

import "context"

// ˄

type IDFKyouRepository interface {
	FindKyous(ctx context.Context, queryJSON string) []*Kyou

	GetKyou(ctx context.Context, id string) *Kyou

	GetKyouHistories(ctx context.Context, id string) []*Kyou

	GetPath(ctx context.Context, id string) string

	UpdateCache(ctx context.Context)

	GetRepName(ctx context.Context) string

	Close(ctx context.Context)

	FindIDFKyou(ctx context.Context, queryJSON string) []*IDFKyou

	GetIDFKyou(ctx context.Context, id string) *IDFKyou

	GetIDFKyouHistories(ctx context.Context, id string) []*IDFKyou

	// ˅

	// ˄
}

// ˅

// ˄
