package reps

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
)

type GitCommitLogRepositories []GitCommitLogRepository

func (g GitCommitLogRepositories) FindKyous(ctx context.Context, query *find.FindQuery) ([]*Kyou, error) {
	matchKyous := map[string]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kyou, len(g))
	errch := make(chan error, len(g))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g {
		wg.Add(1)

		go func(rep GitCommitLogRepository) {
			defer wg.Done()
			matchKyousInRep, err := rep.FindKyous(ctx, query)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKyousInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find kyous: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Kyou集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchKyousInRep := <-ch:
			if matchKyousInRep == nil {
				continue loop
			}
			for _, kyou := range matchKyousInRep {
				if existKyou, exist := matchKyous[kyou.ID]; exist {
					if kyou.UpdateTime.After(existKyou.UpdateTime) {
						matchKyous[kyou.ID] = kyou
					}
				} else {
					matchKyous[kyou.ID] = kyou
				}
			}
		default:
			break loop
		}
	}

	matchKyousList := []*Kyou{}
	for _, kyou := range matchKyous {
		if kyou == nil {
			continue
		}
		matchKyousList = append(matchKyousList, kyou)
	}

	sort.Slice(matchKyousList, func(i, j int) bool {
		return matchKyousList[i].RelatedTime.After(matchKyousList[j].RelatedTime)
	})
	return matchKyousList, nil
}

func (g GitCommitLogRepositories) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	matchKyou := &Kyou{}
	matchKyou = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Kyou, len(g))
	errch := make(chan error, len(g))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g {
		wg.Add(1)

		go func(rep GitCommitLogRepository) {
			defer wg.Done()
			matchKyouInRep, err := rep.GetKyou(ctx, id, updateTime)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKyouInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get kyou: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Kyou集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchKyouInRep := <-ch:
			if matchKyouInRep == nil {
				continue loop
			}
			if matchKyou != nil {
				if matchKyouInRep.UpdateTime.Before(matchKyou.UpdateTime) {
					matchKyou = matchKyouInRep
				}
			} else {
				matchKyou = matchKyouInRep
			}
		default:
			break loop
		}
	}

	return matchKyou, nil
}

func (g GitCommitLogRepositories) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	kyouHistories := map[string]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kyou, len(g))
	errch := make(chan error, len(g))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g {
		wg.Add(1)

		go func(rep GitCommitLogRepository) {
			defer wg.Done()
			matchKyousInRep, err := rep.GetKyouHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKyousInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get kyou histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Kyou集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchKyousInRep := <-ch:
			if matchKyousInRep == nil {
				continue loop
			}
			for _, kyou := range matchKyousInRep {
				if existKyou, exist := kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if kyou.UpdateTime.After(existKyou.UpdateTime) {
						kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)] = kyou
					}
				} else {
					kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)] = kyou
				}
			}
		default:
			break loop
		}
	}

	kyouHistoriesList := []*Kyou{}
	for _, kyou := range kyouHistories {
		if kyou == nil {
			continue
		}
		kyouHistoriesList = append(kyouHistoriesList, kyou)
	}

	sort.Slice(kyouHistoriesList, func(i, j int) bool {
		return kyouHistoriesList[i].UpdateTime.After(kyouHistoriesList[j].UpdateTime)
	})

	return kyouHistoriesList, nil
}

func (g GitCommitLogRepositories) GetPath(ctx context.Context, id string) (string, error) {
	err := fmt.Errorf("not implements GitCommitLogReps.GetPath")
	return "", err
}

func (g GitCommitLogRepositories) UpdateCache(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(g))
	defer close(errch)

	// 並列処理
	for _, rep := range g {
		wg.Add(1)

		go func(rep GitCommitLogRepository) {
			defer wg.Done()
			err = rep.UpdateCache(ctx)
			if err != nil {
				errch <- err
				return
			}
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at update cache: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return err
	}

	return nil
}

func (g GitCommitLogRepositories) GetRepName(ctx context.Context) (string, error) {
	return "GitCommitLogReps", nil
}

func (g GitCommitLogRepositories) Close(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(g))
	defer close(errch)

	// 並列処理
	for _, rep := range g {
		wg.Add(1)

		go func(rep GitCommitLogRepository) {
			defer wg.Done()
			err = rep.Close(ctx)
			if err != nil {
				errch <- err
				return
			}
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at close: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return err
	}

	return nil
}

func (g GitCommitLogRepositories) FindGitCommitLog(ctx context.Context, query *find.FindQuery) ([]*GitCommitLog, error) {
	matchGitCommitLogs := map[string]*GitCommitLog{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*GitCommitLog, len(g))
	errch := make(chan error, len(g))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g {
		wg.Add(1)

		go func(rep GitCommitLogRepository) {
			defer wg.Done()
			matchGitCommitLogsInRep, err := rep.FindGitCommitLog(ctx, query)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchGitCommitLogsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find git commit log: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// GitCommitLog集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchGitCommitLogsInRep := <-ch:
			if matchGitCommitLogsInRep == nil {
				continue loop
			}
			for _, kyou := range matchGitCommitLogsInRep {
				if existGitCommitLog, exist := matchGitCommitLogs[kyou.ID]; exist {
					if kyou.UpdateTime.After(existGitCommitLog.UpdateTime) {
						matchGitCommitLogs[kyou.ID] = kyou
					}
				} else {
					matchGitCommitLogs[kyou.ID] = kyou
				}
			}
		default:
			break loop
		}
	}

	matchGitCommitLogsList := []*GitCommitLog{}
	for _, kyou := range matchGitCommitLogs {
		if kyou == nil {
			continue
		}
		matchGitCommitLogsList = append(matchGitCommitLogsList, kyou)
	}

	sort.Slice(matchGitCommitLogsList, func(i, j int) bool {
		return matchGitCommitLogsList[i].RelatedTime.After(matchGitCommitLogsList[j].RelatedTime)
	})
	return matchGitCommitLogsList, nil
}

func (g GitCommitLogRepositories) GetGitCommitLog(ctx context.Context, id string, updateTime *time.Time) (*GitCommitLog, error) {
	matchGitCommitLog := &GitCommitLog{}
	matchGitCommitLog = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *GitCommitLog, len(g))
	errch := make(chan error, len(g))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g {
		wg.Add(1)

		go func(rep GitCommitLogRepository) {
			defer wg.Done()
			matchGitCommitLogInRep, err := rep.GetGitCommitLog(ctx, id, updateTime)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchGitCommitLogInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get git commit log: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// GitCommitLog集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchGitCommitLogInRep := <-ch:
			if matchGitCommitLogInRep == nil {
				continue loop
			}
			if matchGitCommitLog != nil {
				if matchGitCommitLogInRep.UpdateTime.Before(matchGitCommitLog.UpdateTime) {
					matchGitCommitLog = matchGitCommitLogInRep
				}
			} else {
				matchGitCommitLog = matchGitCommitLogInRep
			}
		default:
			break loop
		}
	}

	return matchGitCommitLog, nil
}
