// ˅
package reps

import "context"

// ˄

type ReKyouRepository interface {
	FindKyous(ctx context.Context, queryJSON string) []*Kyou

	GetKyou(ctx context.Context, id string) *Kyou

	GetKyouHistories(ctx context.Context, id string) []*Kyou

	GetPath(ctx context.Context, id string) string

	UpdateCache(ctx context.Context)

	GetRepName(ctx context.Context) string

	Close(ctx context.Context)

	FindReKyou(ctx context.Context, queryJSON string) []*ReKyou

	GetReKyou(ctx context.Context, id string) *ReKyou

	GetReKyouHistories(ctx context.Context, id string) []*ReKyou

	AddReKyouInfo(ctx context.Context, rekyou *ReKyou)

	// ˅

	// ˄
}

// ˅

// ˄
