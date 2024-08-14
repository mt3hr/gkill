// ˅
package reps

import "context"

// ˄

type NlogRepository interface {
	FindKyous(ctx context.Context, queryJSON string) []*Kyou

	GetKyou(ctx context.Context, id string) *Kyou

	GetKyouHistories(ctx context.Context, id string) []*Kyou

	GetPath(ctx context.Context, id string) string

	UpdateCache(ctx context.Context)

	GetRepName(ctx context.Context) string

	Close(ctx context.Context)

	FindNlog(ctx context.Context, queryJSON string) []*Nlog

	GetNlog(ctx context.Context, id string) *Nlog

	GetNlogHistories(ctx context.Context, id string) []*Nlog

	AddNlogInfo(ctx context.Context, nlog *Nlog)

	// ˅

	// ˄
}

// ˅

// ˄
