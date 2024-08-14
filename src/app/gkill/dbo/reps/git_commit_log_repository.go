// ˅
package reps

import "context"

// ˄

type GitCommitLogRepository interface {
	FindKyous(ctx context.Context, queryJSON string) []*Kyou

	GetKyou(ctx context.Context, id string) *Kyou

	GetKyouHistories(ctx context.Context, id string) []*Kyou

	GetPath(ctx context.Context, id string) string

	UpdateCache(ctx context.Context)

	GetRepName(ctx context.Context) string

	Close(ctx context.Context)

	FindGitCommitLog(ctx context.Context, queryJSON string) []*GitCommitLog

	GetGitCommitLog(ctx context.Context, id string) GitCommitLog

	// ˅

	// ˄
}

// ˅

// ˄
