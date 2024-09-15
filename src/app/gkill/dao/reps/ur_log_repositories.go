package reps

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
)

type URLogRepositories []URLogRepository

func (u URLogRepositories) FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error) {
	matchKyous := map[string]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kyou, len(u))
	errch := make(chan error, len(u))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range u {
		wg.Add(1)
		rep := rep
		go func(rep URLogRepository) {
			defer wg.Done()
			matchKyousInRep, err := rep.FindKyous(ctx, queryJSON)
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
					if kyou.UpdateTime.Before(existKyou.UpdateTime) {
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

func (u URLogRepositories) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	matchKyou := &Kyou{}
	matchKyou = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Kyou, len(u))
	errch := make(chan error, len(u))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range u {
		wg.Add(1)
		rep := rep
		go func(rep URLogRepository) {
			defer wg.Done()
			matchKyouInRep, err := rep.GetKyou(ctx, id)
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

func (u URLogRepositories) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	kyouHistories := map[string]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kyou, len(u))
	errch := make(chan error, len(u))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range u {
		wg.Add(1)
		rep := rep
		go func(rep URLogRepository) {
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
					if kyou.UpdateTime.Before(existKyou.UpdateTime) {
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

func (u URLogRepositories) GetPath(ctx context.Context, id string) (string, error) {
	err := fmt.Errorf("not implements URLogReps.GetPath")
	return "", err
}

func (u URLogRepositories) UpdateCache(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(u))
	defer close(errch)

	// 並列処理
	for _, rep := range u {
		wg.Add(1)
		rep := rep
		go func(rep URLogRepository) {
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

func (u URLogRepositories) GetRepName(ctx context.Context) (string, error) {
	return "URLogReps", nil
}

func (u URLogRepositories) Close(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(u))
	defer close(errch)

	// 並列処理
	for _, rep := range u {
		wg.Add(1)
		rep := rep
		go func(rep URLogRepository) {
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

func (u URLogRepositories) FindURLog(ctx context.Context, queryJSON string) ([]*URLog, error) {
	matchURLogs := map[string]*URLog{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*URLog, len(u))
	errch := make(chan error, len(u))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range u {
		wg.Add(1)
		rep := rep
		go func(rep URLogRepository) {
			defer wg.Done()
			matchURLogsInRep, err := rep.FindURLog(ctx, queryJSON)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchURLogsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find urlog: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// URLog集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchURLogsInRep := <-ch:
			if matchURLogsInRep == nil {
				continue loop
			}
			for _, kyou := range matchURLogsInRep {
				if existURLog, exist := matchURLogs[kyou.ID]; exist {
					if kyou.UpdateTime.Before(existURLog.UpdateTime) {
						matchURLogs[kyou.ID] = kyou
					}
				} else {
					matchURLogs[kyou.ID] = kyou
				}
			}
		default:
			break loop
		}
	}

	matchURLogsList := []*URLog{}
	for _, kyou := range matchURLogs {
		if kyou == nil {
			continue
		}
		matchURLogsList = append(matchURLogsList, kyou)
	}

	sort.Slice(matchURLogsList, func(i, j int) bool {
		return matchURLogsList[i].RelatedTime.After(matchURLogsList[j].RelatedTime)
	})
	return matchURLogsList, nil
}

func (u URLogRepositories) GetURLog(ctx context.Context, id string) (*URLog, error) {
	matchURLog := &URLog{}
	matchURLog = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *URLog, len(u))
	errch := make(chan error, len(u))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range u {
		wg.Add(1)
		rep := rep
		go func(rep URLogRepository) {
			defer wg.Done()
			matchURLogInRep, err := rep.GetURLog(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchURLogInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get urlog: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// URLog集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchURLogInRep := <-ch:
			if matchURLogInRep == nil {
				continue loop
			}
			if matchURLog != nil {
				if matchURLogInRep.UpdateTime.Before(matchURLog.UpdateTime) {
					matchURLog = matchURLogInRep
				}
			} else {
				matchURLog = matchURLogInRep
			}
		default:
			break loop
		}
	}

	return matchURLog, nil
}

func (u URLogRepositories) GetURLogHistories(ctx context.Context, id string) ([]*URLog, error) {
	kyouHistories := map[string]*URLog{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*URLog, len(u))
	errch := make(chan error, len(u))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range u {
		wg.Add(1)
		rep := rep
		go func(rep URLogRepository) {
			defer wg.Done()
			matchURLogsInRep, err := rep.GetURLogHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchURLogsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get urlog histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// URLog集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchURLogsInRep := <-ch:
			if matchURLogsInRep == nil {
				continue loop
			}
			for _, kyou := range matchURLogsInRep {
				if existURLog, exist := kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if kyou.UpdateTime.Before(existURLog.UpdateTime) {
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

	kyouHistoriesList := []*URLog{}
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

func (u URLogRepositories) AddURLogInfo(ctx context.Context, urlog *URLog) error {
	err := fmt.Errorf("not implements URLogReps.AddURLogInfo")
	return err
}
