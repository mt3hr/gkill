// ˅
package reps

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/mt3hr/gkill/src/app/gkill/dbo/sqlite3impl"
)

// ˄

type KmemoRepositories []KmemoRepository

func (k KmemoRepositories) FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error) {
	// ˅
	matchKyous := map[string]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kyou, len(k))
	errch := make(chan error, len(k))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		wg.Add(1)
		rep := rep
		go func(rep KmemoRepository) {
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
	// ˄
}

func (k KmemoRepositories) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	// ˅
	matchKyou := &Kyou{}
	matchKyou = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Kyou, len(k))
	errch := make(chan error, len(k))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		wg.Add(1)
		rep := rep
		go func(rep KmemoRepository) {
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
	// ˄
}

func (k KmemoRepositories) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	// ˅
	kyouHistories := map[string]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kyou, len(k))
	errch := make(chan error, len(k))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		wg.Add(1)
		rep := rep
		go func(rep KmemoRepository) {
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
	// ˄
}

func (k KmemoRepositories) GetPath(ctx context.Context, id string) (string, error) {
	// ˅
	err := fmt.Errorf("not implements KmemoReps.GetPath")
	return "", err
	// ˄
}

func (k KmemoRepositories) UpdateCache(ctx context.Context) error {
	// ˅
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(k))
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		wg.Add(1)
		rep := rep
		go func(rep KmemoRepository) {
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
	// ˄
}

func (k KmemoRepositories) GetRepName(ctx context.Context) (string, error) {
	// ˅
	return "KmemoReps", nil
	// ˄
}

func (k KmemoRepositories) Close(ctx context.Context) error {
	// ˅
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(k))
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		wg.Add(1)
		rep := rep
		go func(rep KmemoRepository) {
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
	// ˄
}

func (k KmemoRepositories) FindKmemo(ctx context.Context, queryJSON string) ([]*Kmemo, error) {
	// ˅
	matchKmemos := map[string]*Kmemo{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kmemo, len(k))
	errch := make(chan error, len(k))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		wg.Add(1)
		rep := rep
		go func(rep KmemoRepository) {
			defer wg.Done()
			matchKmemosInRep, err := rep.FindKmemo(ctx, queryJSON)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKmemosInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find kmemo: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Kmemo集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchKmemosInRep := <-ch:
			if matchKmemosInRep == nil {
				continue loop
			}
			for _, kyou := range matchKmemosInRep {
				if existKmemo, exist := matchKmemos[kyou.ID]; exist {
					if kyou.UpdateTime.Before(existKmemo.UpdateTime) {
						matchKmemos[kyou.ID] = kyou
					}
				} else {
					matchKmemos[kyou.ID] = kyou
				}
			}
		default:
			break loop
		}
	}

	matchKmemosList := []*Kmemo{}
	for _, kyou := range matchKmemos {
		if kyou == nil {
			continue
		}
		matchKmemosList = append(matchKmemosList, kyou)
	}

	sort.Slice(matchKmemosList, func(i, j int) bool {
		return matchKmemosList[i].RelatedTime.After(matchKmemosList[j].RelatedTime)
	})
	return matchKmemosList, nil
	// ˄
}

func (k KmemoRepositories) GetKmemo(ctx context.Context, id string) (*Kmemo, error) {
	// ˅
	matchKmemo := &Kmemo{}
	matchKmemo = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Kmemo, len(k))
	errch := make(chan error, len(k))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		wg.Add(1)
		rep := rep
		go func(rep KmemoRepository) {
			defer wg.Done()
			matchKmemoInRep, err := rep.GetKmemo(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKmemoInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get kmemo: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Kmemo集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchKmemoInRep := <-ch:
			if matchKmemoInRep == nil {
				continue loop
			}
			if matchKmemo != nil {
				if matchKmemoInRep.UpdateTime.Before(matchKmemo.UpdateTime) {
					matchKmemo = matchKmemoInRep
				}
			} else {
				matchKmemo = matchKmemoInRep
			}
		default:
			break loop
		}
	}

	return matchKmemo, nil
	// ˄
}

func (k KmemoRepositories) GetKmemoHistories(ctx context.Context, id string) ([]*Kmemo, error) {
	// ˅
	kyouHistories := map[string]*Kmemo{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kmemo, len(k))
	errch := make(chan error, len(k))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		wg.Add(1)
		rep := rep
		go func(rep KmemoRepository) {
			defer wg.Done()
			matchKmemosInRep, err := rep.GetKmemoHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKmemosInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get kmemo histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Kmemo集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchKmemosInRep := <-ch:
			if matchKmemosInRep == nil {
				continue loop
			}
			for _, kyou := range matchKmemosInRep {
				if existKmemo, exist := kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if kyou.UpdateTime.Before(existKmemo.UpdateTime) {
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

	kyouHistoriesList := []*Kmemo{}
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
	// ˄
}

func (k KmemoRepositories) AddKmemoInfo(ctx context.Context, kmemo *Kmemo) error {
	// ˅
	err := fmt.Errorf("not implements KmemoReps.AddKmemoInfo")
	return err
	// ˄
}

// ˅

// ˄
