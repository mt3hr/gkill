package reps

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
)

type gitCommitLogRepositoryLocalImpl struct {
	gitrep   *git.Repository
	filename string
}

func NewGitRep(reppath string) (GitCommitLogRepository, error) {
	gitrep, err := git.PlainOpen(reppath)
	if err != nil {
		return nil, err
	}

	return &gitCommitLogRepositoryLocalImpl{
		gitrep:   gitrep,
		filename: reppath,
	}, nil
}
func (g *gitCommitLogRepositoryLocalImpl) FindKyous(ctx context.Context, query *find.FindQuery) ([]*Kyou, error) {
	var err error
	// update_cacheであればキャッシュを更新する
	if query.UpdateCache != nil && *query.UpdateCache {
		err = g.UpdateCache(ctx)
		if err != nil {
			repName, _ := g.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}

	repName, err := g.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	// 判定OKであればKyouを作る
	kyous := []*Kyou{}
	logs, err := g.gitrep.Log(&git.LogOptions{All: true})
	defer logs.Close()
	logs.ForEach(func(commit *object.Commit) error {
		// 判定
		match := true

		// 削除済みであるかどうかの判定
		if query.IsDeleted != nil && *query.IsDeleted {
			match = false
			if !match {
				return nil
			}
		}

		// id検索である場合のSQL追記
		if query.UseIDs != nil && *query.UseIDs {
			ids := []string{}
			if query.IDs != nil {
				ids = *query.IDs
			}
			for _, id := range ids {
				match = id == fmt.Sprintf("%s", commit.Hash)
				if match {
					break
				}
			}
			if !match {
				return nil
			}
		}

		// ワードand検索である場合の判定
		if query.WordsAnd != nil && *query.WordsAnd {
			match = false
			words := []string{}
			if query.Words != nil {
				words = *query.Words
			}
			notWords := []string{}
			if query.NotWords != nil {
				notWords = *query.NotWords
			}

			if query.WordsAnd != nil && *query.WordsAnd {
				for _, word := range words {
					match = strings.Contains(fmt.Sprintf("%s", commit.Message), word)
					if !match {
						return nil
					}
				}
			} else {
				// ワードor検索である場合の判定
				for _, word := range words {
					match = strings.Contains(fmt.Sprintf("%s", commit.Message), word)
					if match {
						break
					}
				}
			}

			// notワードを除外する場合の判定
			for _, notWord := range notWords {
				match = strings.Contains(fmt.Sprintf("%s", commit.Message), notWord)
				if match {
					return nil
				}
			}
		}

		kyou := &Kyou{}
		kyou.IsDeleted = false
		kyou.ID = fmt.Sprintf("%s", commit.Hash)
		kyou.RepName = repName
		kyou.RelatedTime = commit.Committer.When
		kyou.DataType = "git_commit_log"
		kyou.CreateTime = commit.Committer.When
		kyou.CreateApp = "git"
		kyou.CreateDevice = ""
		kyou.CreateUser = fmt.Sprintf("%s", commit.Author)
		kyou.UpdateTime = commit.Committer.When
		kyou.UpdateApp = "git"
		kyou.UpdateDevice = ""
		kyou.UpdateUser = fmt.Sprintf("%s", commit.Author)

		kyous = append(kyous, kyou)
		return nil
	})
	return kyous, nil
}

func (g *gitCommitLogRepositoryLocalImpl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	var err error

	repName, err := g.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	// 判定OKであればKyouを作る
	matchKyou := &Kyou{}
	matchKyou = nil
	logs, err := g.gitrep.Log(&git.LogOptions{All: true})
	defer logs.Close()
	logs.ForEach(func(commit *object.Commit) error {
		// 判定
		match := true
		if id == fmt.Sprintf("%s", commit.Hash) {
			match = true
		}
		if !match {
			return nil
		}

		kyou := &Kyou{}
		kyou.IsDeleted = false
		kyou.ID = fmt.Sprintf("%s", commit.Hash)
		kyou.RepName = repName
		kyou.RelatedTime = commit.Committer.When
		kyou.DataType = "git_commit_log"
		kyou.CreateTime = commit.Committer.When
		kyou.CreateApp = "git"
		kyou.CreateDevice = ""
		kyou.CreateUser = fmt.Sprintf("%s", commit.Author)
		kyou.UpdateTime = commit.Committer.When
		kyou.UpdateApp = "git"
		kyou.UpdateDevice = ""
		kyou.UpdateUser = fmt.Sprintf("%s", commit.Author)

		matchKyou = kyou
		return nil
	})
	return matchKyou, nil
}

func (g *gitCommitLogRepositoryLocalImpl) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	kyou, err := g.GetKyou(ctx, id, nil)
	if err != nil {
		err = fmt.Errorf("error at get kyou histories git commit log repositories %s: %w", id, err)
		return nil, err
	}
	return []*Kyou{kyou}, nil
}

func (g *gitCommitLogRepositoryLocalImpl) GetPath(ctx context.Context, id string) (string, error) {
	return filepath.Abs(g.filename)
}

func (g *gitCommitLogRepositoryLocalImpl) UpdateCache(ctx context.Context) error {
	return nil
}

func (g *gitCommitLogRepositoryLocalImpl) GetRepName(ctx context.Context) (string, error) {
	return filepath.Base(g.filename), nil
}

func (g *gitCommitLogRepositoryLocalImpl) Close(ctx context.Context) error {
	return nil
}

func (g *gitCommitLogRepositoryLocalImpl) FindGitCommitLog(ctx context.Context, query *find.FindQuery) ([]*GitCommitLog, error) {
	var err error

	// update_cacheであればキャッシュを更新する
	if query.UpdateCache != nil && *query.UpdateCache {
		err = g.UpdateCache(ctx)
		if err != nil {
			repName, _ := g.GetRepName(ctx)
			err = fmt.Errorf("error at update cache %s: %w", repName, err)
			return nil, err
		}
	}

	repName, err := g.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	// 判定OKであればGitCommitLogを作る
	gitCommitLogs := []*GitCommitLog{}
	logs, err := g.gitrep.Log(&git.LogOptions{All: true})
	defer logs.Close()
	logs.ForEach(func(commit *object.Commit) error {
		// 判定
		match := true

		// id検索である場合のSQL追記
		if query.UseIDs != nil && query.IDs != nil {
			ids := []string{}
			if query.IDs != nil {
				ids = *query.IDs
			}
			for _, id := range ids {
				match = id == fmt.Sprintf("%s", commit.Hash)
				if match {
					break
				}
			}
			if !match {
				return nil
			}
		}

		// ワードand検索である場合の判定
		if query.WordsAnd != nil {
			match = false
			// ワードを解析
			if query.WordsAnd != nil && *query.WordsAnd {
				words := []string{}
				if query.Words != nil {
					words = *query.Words
				}
				for _, word := range words {
					match = strings.Contains(fmt.Sprintf("%s", commit.Message), word)
					if !match {
						return nil
					}
				}
			} else {
				words := []string{}
				if query.Words != nil {
					words = *query.Words
				}
				// ワードor検索である場合の判定
				for _, word := range words {
					match = strings.Contains(fmt.Sprintf("%s", commit.Message), word)
					if match {
						break
					}
				}
			}

			notWords := []string{}
			if query.NotWords != nil {
				notWords = *query.NotWords
			}
			// notワードを除外する場合の判定
			for _, notWord := range notWords {
				match = strings.Contains(fmt.Sprintf("%s", commit.Message), notWord)
				if match {
					return nil
				}
			}
		}

		gitCommitLog := &GitCommitLog{}
		gitCommitLog.IsDeleted = false
		gitCommitLog.ID = fmt.Sprintf("%s", commit.Hash)
		gitCommitLog.RepName = repName
		gitCommitLog.RelatedTime = commit.Committer.When
		gitCommitLog.DataType = "git_commit_log"
		gitCommitLog.CreateTime = commit.Committer.When
		gitCommitLog.CreateApp = "git"
		gitCommitLog.CreateDevice = ""
		gitCommitLog.CreateUser = fmt.Sprintf("%s", commit.Author)
		gitCommitLog.UpdateTime = commit.Committer.When
		gitCommitLog.UpdateApp = "git"
		gitCommitLog.UpdateDevice = ""
		gitCommitLog.UpdateUser = fmt.Sprintf("%s", commit.Author)
		gitCommitLog.CommitMessage = fmt.Sprintf("%s", commit.Message)

		gitCommitLogs = append(gitCommitLogs, gitCommitLog)
		return nil
	})
	return gitCommitLogs, nil
}

func (g *gitCommitLogRepositoryLocalImpl) GetGitCommitLog(ctx context.Context, id string, updateTime *time.Time) (*GitCommitLog, error) {
	var err error

	repName, err := g.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	// 判定OKであればKyouを作る
	matchGitCommitLog := &GitCommitLog{}
	matchGitCommitLog = nil
	logs, err := g.gitrep.Log(&git.LogOptions{All: true})
	defer logs.Close()
	logs.ForEach(func(commit *object.Commit) error {
		// 判定
		match := true
		if id == fmt.Sprintf("%s", commit.Hash) {
			match = true
		}
		if updateTime != nil && updateTime.Format(sqlite3impl.TimeLayout) != commit.Committer.When.Format(sqlite3impl.TimeLayout) {
			match = false
		}
		if !match {
			return nil
		}

		gitCommitLog := &GitCommitLog{}
		gitCommitLog.IsDeleted = false
		gitCommitLog.ID = fmt.Sprintf("%s", commit.Hash)
		gitCommitLog.RepName = repName
		gitCommitLog.RelatedTime = commit.Committer.When
		gitCommitLog.DataType = "git_commit_log"
		gitCommitLog.CreateTime = commit.Committer.When
		gitCommitLog.CreateApp = "git"
		gitCommitLog.CreateDevice = ""
		gitCommitLog.CreateUser = fmt.Sprintf("%s", commit.Author)
		gitCommitLog.UpdateTime = commit.Committer.When
		gitCommitLog.UpdateApp = "git"
		gitCommitLog.UpdateDevice = ""
		gitCommitLog.UpdateUser = fmt.Sprintf("%s", commit.Author)
		gitCommitLog.CommitMessage = fmt.Sprintf("%s", commit.Message)

		matchGitCommitLog = gitCommitLog
		return nil
	})
	return matchGitCommitLog, nil
}
