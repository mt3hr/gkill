package reps

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
)

type gitCommitLogRepositoryLocalImpl struct {
	gitrep   *git.Repository
	filename string
	m        sync.Mutex
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
func (g *gitCommitLogRepositoryLocalImpl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	g.m.Lock()
	defer g.m.Unlock()

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
	kyous := map[string][]*Kyou{}

	var logs object.CommitIter
	if query.UseIDs != nil && *query.UseIDs && len(*query.IDs) == 1 {
		logs, err = g.gitrep.Log(&git.LogOptions{From: plumbing.NewHash((*query.IDs)[0])})
	} else {
		logs, err = g.gitrep.Log(&git.LogOptions{All: true})
	}
	if err != nil {
		return nil, nil
		// return err
	}
	defer logs.Close()
loop:
	for commit, err := logs.Next(); commit != nil; commit, err = logs.Next() {
		if err != nil {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, nil
		default:
			// 判定
			match := true

			// 削除済みであるかどうかの判定
			if query.IsDeleted != nil && *query.IsDeleted {
				match = false
				if !match {
					continue
				}
			}

			// id検索である場合のSQL追記
			if query.UseIDs != nil && *query.UseIDs {
				ids := []string{}
				if query.IDs != nil {
					ids = *query.IDs
				}
				for _, id := range ids {
					match = id == commit.Hash.String()
					if match {
						break
					}
				}
				if !match {
					continue
				}
			}

			words := []string{}
			notWords := []string{}
			if query.Words != nil {
				words = *query.Words
			}
			if query.NotWords != nil {
				notWords = *query.NotWords
			}

			if query.UseWords != nil && *query.UseWords {
				// ワードand検索である場合の判定
				if query.WordsAnd != nil && *query.WordsAnd {
					match = true
					for _, word := range words {
						match = strings.Contains(strings.ToLower(commit.Message), strings.ToLower(word))
						if !match {
							break
						}
					}
					if !match {
						continue
					}
				} else if query.WordsAnd != nil && !(*query.WordsAnd) {
					// ワードor検索である場合の判定
					match = false
					for _, word := range words {
						match = strings.Contains(strings.ToLower(commit.Message), strings.ToLower(word))
						if match {
							break
						}
					}
				}

				// notワードを除外する場合の判定
				for _, notWord := range notWords {
					match = strings.Contains(strings.ToLower(commit.Message), strings.ToLower(notWord))
					if match {
						match = false
						break
					}
				}
			}

			if !match {
				continue
			}

			// 日付範囲指定ありの場合
			useCalendar := false
			calendarStartDate := query.CalendarStartDate
			calendarEndDate := query.CalendarEndDate
			if query.UseCalendar != nil {
				useCalendar = *query.UseCalendar
			}
			if useCalendar {
				if calendarStartDate != nil {
					if !commit.Committer.When.After(*calendarStartDate) {
						continue
					}

				}
				if calendarEndDate != nil {
					if !commit.Committer.When.Before(*calendarEndDate) {
						continue
					}
				}
			}

			kyou := &Kyou{}
			kyou.IsDeleted = false
			kyou.ID = commit.Hash.String()
			kyou.RepName = repName
			kyou.RelatedTime = commit.Committer.When
			kyou.DataType = "git_commit_log"
			kyou.CreateTime = commit.Committer.When
			kyou.CreateApp = "git"
			kyou.CreateDevice = ""
			kyou.CreateUser = commit.Author.Name
			kyou.UpdateTime = commit.Committer.When
			kyou.UpdateApp = "git"
			kyou.UpdateDevice = ""
			kyou.UpdateUser = commit.Author.Name

			if _, exist := kyous[kyou.ID]; !exist {
				kyous[kyou.ID] = []*Kyou{}
			}
			kyous[kyou.ID] = append(kyous[kyou.ID], kyou)

			if query.UseIDs != nil && *query.UseIDs && len(*query.IDs) == 1 {
				break loop
			}
		}
	}
	return kyous, nil
}

func (g *gitCommitLogRepositoryLocalImpl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	g.m.Lock()
	defer g.m.Unlock()

	var err error

	repName, err := g.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	// 判定OKであればKyouを作る
	var matchKyou *Kyou
	logs, err := g.gitrep.Log(&git.LogOptions{From: plumbing.NewHash(id)})
	if err != nil {
		return nil, nil
		// return err
	}
	defer logs.Close()
loop:
	for commit, err := logs.Next(); commit != nil; commit, err = logs.Next() {
		if err != nil {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, nil
		default:
			// 判定
			match := true
			if id == commit.Hash.String() {
				match = true
			}
			if !match {
				continue
			}

			kyou := &Kyou{}
			kyou.IsDeleted = false
			kyou.ID = commit.Hash.String()
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
			break loop
		}
	}
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
	g.m.Lock()
	defer g.m.Unlock()

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
	var logs object.CommitIter
	if query.UseIDs != nil && *query.UseIDs && len(*query.IDs) == 1 {
		logs, err = g.gitrep.Log(&git.LogOptions{From: plumbing.NewHash((*query.IDs)[0])})
	} else {
		logs, err = g.gitrep.Log(&git.LogOptions{All: true})
	}
	if err != nil {
		return nil, nil
		// return err
	}
	defer logs.Close()
loop:
	for commit, err := logs.Next(); commit != nil; commit, err = logs.Next() {
		if err != nil {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, nil
		default:
			// 判定
			match := true

			// id検索である場合のSQL追記
			if query.UseIDs != nil && query.IDs != nil {
				ids := []string{}
				if query.IDs != nil {
					ids = *query.IDs
				}
				for _, id := range ids {
					match = id == commit.Hash.String()
					if match {
						break
					}
				}
				if !match {
					continue
				}
			}

			words := []string{}
			notWords := []string{}
			if query.Words != nil {
				words = *query.Words
			}
			if query.NotWords != nil {
				notWords = *query.NotWords
			}

			if query.UseWords != nil && *query.UseWords {
				// ワードand検索である場合の判定
				if query.WordsAnd != nil && *query.WordsAnd {
					match = true
					for _, word := range words {
						match = strings.Contains(strings.ToLower(commit.Message), strings.ToLower(word))
						if !match {
							break
						}
					}
					if !match {
						continue
					}
				} else if query.WordsAnd != nil && !(*query.WordsAnd) {
					// ワードor検索である場合の判定
					match = false
					for _, word := range words {
						match = strings.Contains(strings.ToLower(commit.Message), strings.ToLower(word))
						if match {
							break
						}
					}
				}
				// notワードを除外する場合の判定
				for _, notWord := range notWords {
					match = strings.Contains(strings.ToLower(commit.Message), strings.ToLower(notWord))
					if match {
						match = false
						break
					}
				}
			}

			if !match {
				continue
			}

			addition, deletion := 0, 0
			stats, err := commit.StatsContext(ctx)
			if err != nil {
				err = fmt.Errorf("error at get stat from commit: %w", err)
				return nil, err
			}

			for _, stat := range stats {
				addition += stat.Addition
				deletion += stat.Deletion
			}

			gitCommitLog := &GitCommitLog{}
			gitCommitLog.IsDeleted = false
			gitCommitLog.ID = commit.Hash.String()
			gitCommitLog.RepName = repName
			gitCommitLog.RelatedTime = commit.Committer.When
			gitCommitLog.DataType = "git_commit_log"
			gitCommitLog.CreateTime = commit.Committer.When
			gitCommitLog.CreateApp = "git"
			gitCommitLog.CreateDevice = ""
			gitCommitLog.CreateUser = commit.Author.Name
			gitCommitLog.UpdateTime = commit.Committer.When
			gitCommitLog.UpdateApp = "git"
			gitCommitLog.UpdateDevice = ""
			gitCommitLog.UpdateUser = commit.Author.Name
			gitCommitLog.CommitMessage = commit.Message
			gitCommitLog.Addition = addition
			gitCommitLog.Deletion = deletion

			gitCommitLogs = append(gitCommitLogs, gitCommitLog)

			if query.UseIDs != nil && *query.UseIDs && len(*query.IDs) == 1 {
				break loop
			}
		}
	}
	return gitCommitLogs, nil
}

func (g *gitCommitLogRepositoryLocalImpl) GetGitCommitLog(ctx context.Context, id string, updateTime *time.Time) (*GitCommitLog, error) {
	g.m.Lock()
	defer g.m.Unlock()

	var err error

	repName, err := g.GetRepName(ctx)
	if err != nil {
		return nil, err
	}

	// 判定OKであればKyouを作る
	var matchGitCommitLog *GitCommitLog
	logs, err := g.gitrep.Log(&git.LogOptions{From: plumbing.NewHash(id)})
	if err != nil {
		return nil, nil
		// return err
	}
	defer logs.Close()

loop:
	for commit, err := logs.Next(); commit != nil; commit, err = logs.Next() {
		if err != nil {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, nil
		default:
			// 判定
			match := false
			if id == commit.Hash.String() {
				match = true
			}
			if updateTime != nil && updateTime.Format(sqlite3impl.TimeLayout) != commit.Committer.When.Format(sqlite3impl.TimeLayout) {
				match = false
			}
			if !match {
				continue
			}

			addition, deletion := 0, 0
			stats, err := commit.StatsContext(ctx)
			if err != nil {
				err = fmt.Errorf("error at get stat from commit: %w", err)
				return nil, err
			}

			for _, stat := range stats {
				addition += stat.Addition
				deletion += stat.Deletion
			}

			gitCommitLog := &GitCommitLog{}
			gitCommitLog.IsDeleted = false
			gitCommitLog.ID = commit.Hash.String()
			gitCommitLog.RepName = repName
			gitCommitLog.RelatedTime = commit.Committer.When
			gitCommitLog.DataType = "git_commit_log"
			gitCommitLog.CreateTime = commit.Committer.When
			gitCommitLog.CreateApp = "git"
			gitCommitLog.CreateDevice = ""
			gitCommitLog.CreateUser = commit.Author.Name
			gitCommitLog.UpdateTime = commit.Committer.When
			gitCommitLog.UpdateApp = "git"
			gitCommitLog.UpdateDevice = ""
			gitCommitLog.UpdateUser = commit.Author.Name
			gitCommitLog.CommitMessage = commit.Message
			gitCommitLog.Addition = addition
			gitCommitLog.Deletion = deletion

			matchGitCommitLog = gitCommitLog
			break loop
		}
	}
	return matchGitCommitLog, nil
}
