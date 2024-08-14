// ˅
package reps

import "context"

// ˄

type URLogRepository interface {
	FindKyous(ctx context.Context, queryJSON string) []*Kyou

	GetKyou(ctx context.Context, id string) *Kyou

	GetKyouHistories(ctx context.Context, id string) []*Kyou

	GetPath(ctx context.Context, id string) string

	UpdateCache(ctx context.Context)

	GetRepName(ctx context.Context) string

	Close(ctx context.Context)

	FindURLog(ctx context.Context, queryJSON string) []*URLog

	GetURLog(ctx context.Context, id string) *URLog

	GetURLogHistories(ctx context.Context, id string) []*URLog

	AddURLogInfo(ctx context.Context, urlog *URLog)

	// ˅

	// ˄
}

// ˅

// ˄
