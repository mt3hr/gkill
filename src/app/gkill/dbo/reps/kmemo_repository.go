// ˅
package reps

import "context"

// ˄

type KmemoRepository interface {
	FindKyous(ctx context.Context, queryJSON string) []*Kyou

	GetKyou(ctx context.Context, id string) *Kyou

	GetKyouHistories(ctx context.Context, id string) []*Kyou

	GetPath(ctx context.Context, id string) string

	UpdateCache(ctx context.Context)

	GetRepName(ctx context.Context) string

	Close(ctx context.Context)

	FindKmemo(ctx context.Context, queryJSON string) []*Kmemo

	GetKmemo(ctx context.Context, id string) *Kmemo

	GetKmemoHistories(ctx context.Context, id string) []*Kmemo

	AddKmemoInfo(ctx context.Context, kmemo *Kmemo)

	// ˅

	// ˄
}

// ˅

// ˄
