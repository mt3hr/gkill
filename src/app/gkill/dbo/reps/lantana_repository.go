// ˅
package reps

import "context"

// ˄

type LantanaRepository interface {
	FindKyous(ctx context.Context, queryJSON string) []*Kyou

	GetKyou(ctx context.Context, id string) *Kyou

	GetKyouHistories(ctx context.Context, id string) []*Kyou

	GetPath(ctx context.Context, id string) string

	UpdateCache(ctx context.Context)

	GetRepName(ctx context.Context) string

	Close(ctx context.Context)

	FindLantana(ctx context.Context, queryJSON string) []*Lantana

	GetLantana(ctx context.Context, id string) *Lantana

	GetLantanaHistories(ctx context.Context, id string) []*Lantana

	AddLantanaInfo(ctx context.Context, lantana *Lantana)

	// ˅

	// ˄
}

// ˅

// ˄
