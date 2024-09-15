package reps

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
)

type IDFKyouRepositories []IDFKyouRepository

func (i IDFKyouRepositories) FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error) {
	matchKyous := map[string]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kyou, len(i))
	errch := make(chan error, len(i))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range i {
		wg.Add(1)
		rep := rep
		go func(rep IDFKyouRepository) {
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

func (i IDFKyouRepositories) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	matchKyou := &Kyou{}
	matchKyou = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Kyou, len(i))
	errch := make(chan error, len(i))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range i {
		wg.Add(1)
		rep := rep
		go func(rep IDFKyouRepository) {
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

func (i IDFKyouRepositories) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	kyouHistories := map[string]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kyou, len(i))
	errch := make(chan error, len(i))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range i {
		wg.Add(1)
		rep := rep
		go func(rep IDFKyouRepository) {
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

func (i IDFKyouRepositories) GetPath(ctx context.Context, id string) (string, error) {
	err := fmt.Errorf("not implements IDFKyouReps.GetPath")
	return "", err
}

func (i IDFKyouRepositories) UpdateCache(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(i))
	defer close(errch)

	// 並列処理
	for _, rep := range i {
		wg.Add(1)
		rep := rep
		go func(rep IDFKyouRepository) {
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

func (i IDFKyouRepositories) GetRepName(ctx context.Context) (string, error) {
	return "IDFKyouReps", nil
}

func (i IDFKyouRepositories) Close(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(i))
	defer close(errch)

	// 並列処理
	for _, rep := range i {
		wg.Add(1)
		rep := rep
		go func(rep IDFKyouRepository) {
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

func (i IDFKyouRepositories) FindIDFKyou(ctx context.Context, queryJSON string) ([]*IDFKyou, error) {
	matchIDFKyous := map[string]*IDFKyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*IDFKyou, len(i))
	errch := make(chan error, len(i))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range i {
		wg.Add(1)
		rep := rep
		go func(rep IDFKyouRepository) {
			defer wg.Done()
			matchIDFKyousInRep, err := rep.FindIDFKyou(ctx, queryJSON)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchIDFKyousInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find idfkyou: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// IDFKyou集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchIDFKyousInRep := <-ch:
			if matchIDFKyousInRep == nil {
				continue loop
			}
			for _, kyou := range matchIDFKyousInRep {
				if existIDFKyou, exist := matchIDFKyous[kyou.ID]; exist {
					if kyou.UpdateTime.Before(existIDFKyou.UpdateTime) {
						matchIDFKyous[kyou.ID] = kyou
					}
				} else {
					matchIDFKyous[kyou.ID] = kyou
				}
			}
		default:
			break loop
		}
	}

	matchIDFKyousList := []*IDFKyou{}
	for _, kyou := range matchIDFKyous {
		if kyou == nil {
			continue
		}
		matchIDFKyousList = append(matchIDFKyousList, kyou)
	}

	sort.Slice(matchIDFKyousList, func(i, j int) bool {
		return matchIDFKyousList[i].RelatedTime.After(matchIDFKyousList[j].RelatedTime)
	})
	return matchIDFKyousList, nil
}

func (i IDFKyouRepositories) GetIDFKyou(ctx context.Context, id string) (*IDFKyou, error) {
	matchIDFKyou := &IDFKyou{}
	matchIDFKyou = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *IDFKyou, len(i))
	errch := make(chan error, len(i))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range i {
		wg.Add(1)
		rep := rep
		go func(rep IDFKyouRepository) {
			defer wg.Done()
			matchIDFKyouInRep, err := rep.GetIDFKyou(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchIDFKyouInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get idfkyou: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// IDFKyou集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchIDFKyouInRep := <-ch:
			if matchIDFKyouInRep == nil {
				continue loop
			}
			if matchIDFKyou != nil {
				if matchIDFKyouInRep.UpdateTime.Before(matchIDFKyou.UpdateTime) {
					matchIDFKyou = matchIDFKyouInRep
				}
			} else {
				matchIDFKyou = matchIDFKyouInRep
			}
		default:
			break loop
		}
	}

	return matchIDFKyou, nil
}

func (i IDFKyouRepositories) GetIDFKyouHistories(ctx context.Context, id string) ([]*IDFKyou, error) {
	kyouHistories := map[string]*IDFKyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*IDFKyou, len(i))
	errch := make(chan error, len(i))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range i {
		wg.Add(1)
		rep := rep
		go func(rep IDFKyouRepository) {
			defer wg.Done()
			matchIDFKyousInRep, err := rep.GetIDFKyouHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchIDFKyousInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get idfkyou histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// IDFKyou集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchIDFKyousInRep := <-ch:
			if matchIDFKyousInRep == nil {
				continue loop
			}
			for _, kyou := range matchIDFKyousInRep {
				if existIDFKyou, exist := kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if kyou.UpdateTime.Before(existIDFKyou.UpdateTime) {
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

	kyouHistoriesList := []*IDFKyou{}
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

func (i IDFKyouRepositories) IDF(ctx context.Context) error {
	err := fmt.Errorf("not implements IDFKyouReps.IDF")
	return err
}

func (i IDFKyouRepositories) AddIDFKyouInfo(ctx context.Context, idfKyou *IDFKyou) error {
	err := fmt.Errorf("not implements IDFKyouReps.AddIDFKyouInfo")
	return err
}
