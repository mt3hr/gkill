package reps

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
)

type NlogRepositories []NlogRepository

func (n NlogRepositories) FindKyous(ctx context.Context, query *find.FindQuery) ([]*Kyou, error) {
	matchKyous := map[string]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kyou, len(n))
	errch := make(chan error, len(n))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range n {
		wg.Add(1)
		
		go func(rep NlogRepository) {
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

func (n NlogRepositories) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	matchKyou := &Kyou{}
	matchKyou = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Kyou, len(n))
	errch := make(chan error, len(n))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range n {
		wg.Add(1)
		
		go func(rep NlogRepository) {
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

func (n NlogRepositories) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	kyouHistories := map[string]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kyou, len(n))
	errch := make(chan error, len(n))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range n {
		wg.Add(1)
		
		go func(rep NlogRepository) {
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

func (n NlogRepositories) GetPath(ctx context.Context, id string) (string, error) {
	err := fmt.Errorf("not implements NlogReps.GetPath")
	return "", err
}

func (n NlogRepositories) UpdateCache(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(n))
	defer close(errch)

	// 並列処理
	for _, rep := range n {
		wg.Add(1)
		
		go func(rep NlogRepository) {
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

func (n NlogRepositories) GetRepName(ctx context.Context) (string, error) {
	return "NlogReps", nil
}

func (n NlogRepositories) Close(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(n))
	defer close(errch)

	// 並列処理
	for _, rep := range n {
		wg.Add(1)
		
		go func(rep NlogRepository) {
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

func (n NlogRepositories) FindNlog(ctx context.Context, query *find.FindQuery) ([]*Nlog, error) {
	matchNlogs := map[string]*Nlog{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Nlog, len(n))
	errch := make(chan error, len(n))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range n {
		wg.Add(1)
		
		go func(rep NlogRepository) {
			defer wg.Done()
			matchNlogsInRep, err := rep.FindNlog(ctx, query)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchNlogsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find nlog: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Nlog集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchNlogsInRep := <-ch:
			if matchNlogsInRep == nil {
				continue loop
			}
			for _, kyou := range matchNlogsInRep {
				if existNlog, exist := matchNlogs[kyou.ID]; exist {
					if kyou.UpdateTime.Before(existNlog.UpdateTime) {
						matchNlogs[kyou.ID] = kyou
					}
				} else {
					matchNlogs[kyou.ID] = kyou
				}
			}
		default:
			break loop
		}
	}

	matchNlogsList := []*Nlog{}
	for _, kyou := range matchNlogs {
		if kyou == nil {
			continue
		}
		matchNlogsList = append(matchNlogsList, kyou)
	}

	sort.Slice(matchNlogsList, func(i, j int) bool {
		return matchNlogsList[i].RelatedTime.After(matchNlogsList[j].RelatedTime)
	})
	return matchNlogsList, nil
}

func (n NlogRepositories) GetNlog(ctx context.Context, id string) (*Nlog, error) {
	matchNlog := &Nlog{}
	matchNlog = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Nlog, len(n))
	errch := make(chan error, len(n))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range n {
		wg.Add(1)
		
		go func(rep NlogRepository) {
			defer wg.Done()
			matchNlogInRep, err := rep.GetNlog(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchNlogInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get nlog: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Nlog集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchNlogInRep := <-ch:
			if matchNlogInRep == nil {
				continue loop
			}
			if matchNlog != nil {
				if matchNlogInRep.UpdateTime.Before(matchNlog.UpdateTime) {
					matchNlog = matchNlogInRep
				}
			} else {
				matchNlog = matchNlogInRep
			}
		default:
			break loop
		}
	}

	return matchNlog, nil
}

func (n NlogRepositories) GetNlogHistories(ctx context.Context, id string) ([]*Nlog, error) {
	kyouHistories := map[string]*Nlog{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Nlog, len(n))
	errch := make(chan error, len(n))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range n {
		wg.Add(1)
		
		go func(rep NlogRepository) {
			defer wg.Done()
			matchNlogsInRep, err := rep.GetNlogHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchNlogsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get nlog histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Nlog集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchNlogsInRep := <-ch:
			if matchNlogsInRep == nil {
				continue loop
			}
			for _, kyou := range matchNlogsInRep {
				if existNlog, exist := kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if kyou.UpdateTime.Before(existNlog.UpdateTime) {
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

	kyouHistoriesList := []*Nlog{}
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

func (n NlogRepositories) AddNlogInfo(ctx context.Context, nlog *Nlog) error {
	err := fmt.Errorf("not implements NlogReps.AddNlogInfo")
	return err
}
