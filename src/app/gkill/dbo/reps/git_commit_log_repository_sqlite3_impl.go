// ˅
package reps

import "context"

// ˄

type gitCommitLogRepositoryLocalImpl struct {
	// ˅

	// ˄
}

// ˅
func (g *gitCommitLogRepositoryLocalImpl) FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error) {
	panic("notImplements")
}

func (g *gitCommitLogRepositoryLocalImpl) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	panic("notImplements")
}

func (g *gitCommitLogRepositoryLocalImpl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	panic("notImplements")
}

func (g *gitCommitLogRepositoryLocalImpl) GetPath(ctx context.Context, id string) (string, error) {
	panic("notImplements")
}

func (g *gitCommitLogRepositoryLocalImpl) UpdateCache(ctx context.Context) error {
	panic("notImplements")
}

func (g *gitCommitLogRepositoryLocalImpl) GetRepName(ctx context.Context) (string, error) {
	panic("notImplements")
}

func (g *gitCommitLogRepositoryLocalImpl) Close(ctx context.Context) error {
	panic("notImplements")
}

func (g *gitCommitLogRepositoryLocalImpl) FindGitCommitLog(ctx context.Context, queryJSON string) ([]*GitCommitLog, error) {
	panic("notImplements")
}

func (g *gitCommitLogRepositoryLocalImpl) GetGitCommitLog(ctx context.Context, id string) (GitCommitLog, error) {
	panic("notImplements")
}

// ˄
