// ˅
package reps

import "context"

// ˄

type TimeIsRepository interface {
	FindKyous(ctx context.Context, queryJSON string) []*Kyou

	GetKyou(ctx context.Context, id string) *Kyou

	GetKyouHistories(ctx context.Context, id string) []*Kyou

	GetPath(ctx context.Context, id string) string

	UpdateCache(ctx context.Context)

	GetRepName(ctx context.Context) string

	Close(ctx context.Context)

	FindTimeIs(ctx context.Context, queryJSON string) []*TimeIs

	GetTimeIs(ctx context.Context, id string) *TimeIs

	GetTimeIsHistories(ctx context.Context, id string) []*TimeIs

	AddTimeIsInfo(ctx context.Context, timeis *TimeIs)

	// ˅

	// ˄
}

// ˅

// ˄
