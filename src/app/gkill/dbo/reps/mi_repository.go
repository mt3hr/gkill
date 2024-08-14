// ˅
package reps

import "context"

// ˄

type MiRepository interface {
	FindKyous(ctx context.Context, queryJSON string) []*Kyou

	GetKyou(ctx context.Context, id string) *Kyou

	GetKyouHistories(ctx context.Context, id string) []*Kyou

	GetPath(ctx context.Context, id string) string

	UpdateCache(ctx context.Context)

	GetRepName(ctx context.Context) string

	Close(ctx context.Context)

	FindMi(ctx context.Context, queryJSON string) []*Mi

	GetMi(ctx context.Context, id string) *Mi

	GetMiHistories(ctx context.Context, id string) []*Mi

	AddMiInfo(ctx context.Context, mi *Mi)

	// ˅

	// ˄
}

// ˅

// ˄
