package reps

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
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
func (g *gitCommitLogRepositoryLocalImpl) FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error) {
	var err error

	// jsonからパースする
	queryMap := map[string]string{}
	err = json.Unmarshal([]byte(queryJSON), &queryMap)
	if err != nil {
		err = fmt.Errorf("error at parse query json at kmemo %s: %w", queryJSON, err)
		return nil, err
	}

	// update_cacheであればキャッシュを更新する
	if queryMap["update_cache"] == fmt.Sprintf("%t", true) {
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
		if queryMap["is_deleted"] == fmt.Sprintf("%t", true) {
			match = false
			if !match {
				return nil
			}
		}

		// id検索である場合のSQL追記
		if queryMap["use_ids"] == fmt.Sprintf("%t", true) {
			ids := []string{}
			err := json.Unmarshal([]byte(queryMap["ids"]), ids)
			if err != nil {
				err = fmt.Errorf("error at parse ids %s: %w", ids, err)
				return nil
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
		if queryMap["use_word"] == fmt.Sprintf("%t", true) {
			match = false
			// ワードを解析
			words := []string{}
			err = json.Unmarshal([]byte(queryMap["words"]), &words)
			if err != nil {
				err = fmt.Errorf("error at parse query word %s: %w", queryMap["words"], err)
				return err
			}
			notWords := []string{}
			err = json.Unmarshal([]byte(queryMap["not_words"]), &words)
			if err != nil {
				err = fmt.Errorf("error at parse query not word %s: %w", queryMap["not_words"], err)
				return err
			}

			if queryMap["words_and"] == fmt.Sprintf("%t", true) {
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

func (g *gitCommitLogRepositoryLocalImpl) GetKyou(ctx context.Context, id string) (*Kyou, error) {
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
	kyou, err := g.GetKyou(ctx, id)
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

func (g *gitCommitLogRepositoryLocalImpl) FindGitCommitLog(ctx context.Context, queryJSON string) ([]*GitCommitLog, error) {
	var err error

	// jsonからパースする
	queryMap := map[string]string{}
	err = json.Unmarshal([]byte(queryJSON), &queryMap)
	if err != nil {
		err = fmt.Errorf("error at parse query json at kmemo %s: %w", queryJSON, err)
		return nil, err
	}

	// update_cacheであればキャッシュを更新する
	if queryMap["update_cache"] == fmt.Sprintf("%t", true) {
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
		if queryMap["use_ids"] == fmt.Sprintf("%t", true) {
			ids := []string{}
			err := json.Unmarshal([]byte(queryMap["ids"]), ids)
			if err != nil {
				err = fmt.Errorf("error at parse ids %s: %w", ids, err)
				return nil
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
		if queryMap["use_word"] == fmt.Sprintf("%t", true) {
			match = false
			// ワードを解析
			words := []string{}
			err = json.Unmarshal([]byte(queryMap["words"]), &words)
			if err != nil {
				err = fmt.Errorf("error at parse query word %s: %w", queryMap["words"], err)
				return err
			}
			notWords := []string{}
			err = json.Unmarshal([]byte(queryMap["not_words"]), &words)
			if err != nil {
				err = fmt.Errorf("error at parse query not word %s: %w", queryMap["not_words"], err)
				return err
			}

			if queryMap["words_and"] == fmt.Sprintf("%t", true) {
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

func (g *gitCommitLogRepositoryLocalImpl) GetGitCommitLog(ctx context.Context, id string) (*GitCommitLog, error) {
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
