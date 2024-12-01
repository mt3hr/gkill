package reps

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
)

type TimeIsRepositories []TimeIsRepository

func (t TimeIsRepositories) FindKyous(ctx context.Context, query *find.FindQuery) ([]*Kyou, error) {
	matchKyous := map[string]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kyou, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)
		
		go func(rep TimeIsRepository) {
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

func (t TimeIsRepositories) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	matchKyou := &Kyou{}
	matchKyou = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Kyou, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)
		
		go func(rep TimeIsRepository) {
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

func (t TimeIsRepositories) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	kyouHistories := map[string]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kyou, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)
		
		go func(rep TimeIsRepository) {
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

func (t TimeIsRepositories) GetPath(ctx context.Context, id string) (string, error) {
	err := fmt.Errorf("not implements TimeIsReps.GetPath")
	return "", err
}

func (t TimeIsRepositories) UpdateCache(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(t))
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)
		
		go func(rep TimeIsRepository) {
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

func (t TimeIsRepositories) GetRepName(ctx context.Context) (string, error) {
	return "TimeIsReps", nil
}

func (t TimeIsRepositories) Close(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(t))
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)
		
		go func(rep TimeIsRepository) {
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

func (t TimeIsRepositories) FindTimeIs(ctx context.Context, query *find.FindQuery) ([]*TimeIs, error) {
	matchTimeIss := map[string]*TimeIs{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*TimeIs, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)
		
		go func(rep TimeIsRepository) {
			defer wg.Done()
			matchTimeIssInRep, err := rep.FindTimeIs(ctx, query)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTimeIssInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find timeis: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// TimeIs集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchTimeIssInRep := <-ch:
			if matchTimeIssInRep == nil {
				continue loop
			}
			for _, kyou := range matchTimeIssInRep {
				if existTimeIs, exist := matchTimeIss[kyou.ID]; exist {
					if kyou.UpdateTime.Before(existTimeIs.UpdateTime) {
						matchTimeIss[kyou.ID] = kyou
					}
				} else {
					matchTimeIss[kyou.ID] = kyou
				}
			}
		default:
			break loop
		}
	}

	matchTimeIssList := []*TimeIs{}
	for _, kyou := range matchTimeIss {
		if kyou == nil {
			continue
		}
		matchTimeIssList = append(matchTimeIssList, kyou)
	}

	sort.Slice(matchTimeIssList, func(i, j int) bool {
		return matchTimeIssList[i].StartTime.After(matchTimeIssList[j].StartTime)
	})
	return matchTimeIssList, nil
}

func (t TimeIsRepositories) GetTimeIs(ctx context.Context, id string) (*TimeIs, error) {
	matchTimeIs := &TimeIs{}
	matchTimeIs = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *TimeIs, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)
		
		go func(rep TimeIsRepository) {
			defer wg.Done()
			matchTimeIsInRep, err := rep.GetTimeIs(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTimeIsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get timeis: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// TimeIs集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchTimeIsInRep := <-ch:
			if matchTimeIsInRep == nil {
				continue loop
			}
			if matchTimeIs != nil {
				if matchTimeIsInRep.UpdateTime.Before(matchTimeIs.UpdateTime) {
					matchTimeIs = matchTimeIsInRep
				}
			} else {
				matchTimeIs = matchTimeIsInRep
			}
		default:
			break loop
		}
	}

	return matchTimeIs, nil
}

func (t TimeIsRepositories) GetTimeIsHistories(ctx context.Context, id string) ([]*TimeIs, error) {
	kyouHistories := map[string]*TimeIs{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*TimeIs, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)
		
		go func(rep TimeIsRepository) {
			defer wg.Done()
			matchTimeIssInRep, err := rep.GetTimeIsHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTimeIssInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get timeis histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// TimeIs集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchTimeIssInRep := <-ch:
			if matchTimeIssInRep == nil {
				continue loop
			}
			for _, kyou := range matchTimeIssInRep {
				if existTimeIs, exist := kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if kyou.UpdateTime.Before(existTimeIs.UpdateTime) {
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

	kyouHistoriesList := []*TimeIs{}
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

func (t TimeIsRepositories) AddTimeIsInfo(ctx context.Context, timeis *TimeIs) error {
	err := fmt.Errorf("not implements TimeIsReps.AddTimeIsInfo")
	return err
}
