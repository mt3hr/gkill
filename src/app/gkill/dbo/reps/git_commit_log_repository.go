// ˅
package reps

import "context"

// ˄

type GitCommitLogRepository interface {
	FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error)

	GetKyou(ctx context.Context, id string) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindGitCommitLog(ctx context.Context, queryJSON string) ([]*GitCommitLog, error)

	GetGitCommitLog(ctx context.Context, id string) (GitCommitLog, error)

	// ˅

	// ˄
}

// ˅

// ˄
