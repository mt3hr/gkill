package reps

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/threads"
)

type KCRepositories []KCRepository

func (k KCRepositories) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	matchKyous := map[string][]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan map[string][]*Kyou, len(k))
	errch := make(chan error, len(k))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		wg.Add(1)

		done := threads.AllocateThread()
		go func(rep KCRepository) {
			defer done()
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
			for _, kyous := range matchKyousInRep {
				for _, kyou := range kyous {
					if _, exist := matchKyous[kyou.ID]; !exist {
						matchKyous[kyou.ID] = []*Kyou{}
					}
					matchKyous[kyou.ID] = append(matchKyous[kyou.ID], kyou)
				}
			}
		default:
			break loop
		}
	}
	return matchKyous, nil
}

func (k KCRepositories) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	var matchKyou *Kyou
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

		done := threads.AllocateThread()
		go func(rep KCRepository) {
			defer done()
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

func (k KCRepositories) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
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

		done := threads.AllocateThread()
		go func(rep KCRepository) {
			defer done()
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

func (k KCRepositories) GetPath(ctx context.Context, id string) (string, error) {
	err := fmt.Errorf("not implements KCReps.GetPath")
	return "", err
}

func (k KCRepositories) UpdateCache(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(k))
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		wg.Add(1)

		done := threads.AllocateThread()
		go func(rep KCRepository) {
			defer done()
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

func (k KCRepositories) GetRepName(ctx context.Context) (string, error) {
	return "KCReps", nil
}

func (k KCRepositories) Close(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(k))
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		wg.Add(1)

		done := threads.AllocateThread()
		go func(rep KCRepository) {
			defer done()
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

func (k KCRepositories) FindKC(ctx context.Context, query *find.FindQuery) ([]*KC, error) {
	matchKCs := map[string]*KC{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*KC, len(k))
	errch := make(chan error, len(k))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		wg.Add(1)

		done := threads.AllocateThread()
		go func(rep KCRepository) {
			defer done()
			defer wg.Done()
			matchKCsInRep, err := rep.FindKC(ctx, query)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKCsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find kc: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// KC集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchKCsInRep := <-ch:
			if matchKCsInRep == nil {
				continue loop
			}
			for _, kyou := range matchKCsInRep {
				if existKC, exist := matchKCs[kyou.ID]; exist {
					if kyou.UpdateTime.After(existKC.UpdateTime) {
						matchKCs[kyou.ID] = kyou
					}
				} else {
					matchKCs[kyou.ID] = kyou
				}
			}
		default:
			break loop
		}
	}

	matchKCsList := []*KC{}
	for _, kyou := range matchKCs {
		if kyou == nil {
			continue
		}
		matchKCsList = append(matchKCsList, kyou)
	}

	sort.Slice(matchKCsList, func(i, j int) bool {
		return matchKCsList[i].RelatedTime.After(matchKCsList[j].RelatedTime)
	})
	return matchKCsList, nil
}

func (k KCRepositories) GetKC(ctx context.Context, id string, updateTime *time.Time) (*KC, error) {
	var matchKC *KC
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *KC, len(k))
	errch := make(chan error, len(k))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		wg.Add(1)

		done := threads.AllocateThread()
		go func(rep KCRepository) {
			defer done()
			defer wg.Done()
			matchKCInRep, err := rep.GetKC(ctx, id, updateTime)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKCInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get kc: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// KC集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchKCInRep := <-ch:
			if matchKCInRep == nil {
				continue loop
			}
			if matchKC != nil {
				if matchKCInRep.UpdateTime.Before(matchKC.UpdateTime) {
					matchKC = matchKCInRep
				}
			} else {
				matchKC = matchKCInRep
			}
		default:
			break loop
		}
	}

	return matchKC, nil
}

func (k KCRepositories) GetKCHistories(ctx context.Context, id string) ([]*KC, error) {
	kyouHistories := map[string]*KC{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*KC, len(k))
	errch := make(chan error, len(k))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		wg.Add(1)

		done := threads.AllocateThread()
		go func(rep KCRepository) {
			defer done()
			defer wg.Done()
			matchKCsInRep, err := rep.GetKCHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKCsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get kc histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// KC集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchKCsInRep := <-ch:
			if matchKCsInRep == nil {
				continue loop
			}
			for _, kyou := range matchKCsInRep {
				if existKC, exist := kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if kyou.UpdateTime.After(existKC.UpdateTime) {
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

	kyouHistoriesList := []*KC{}
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

func (k KCRepositories) GetKCHistoriesByRepName(ctx context.Context, id string, repName *string) ([]*KC, error) {
	kyouHistories := map[string]*KC{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*KC, len(k))
	errch := make(chan error, len(k))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		wg.Add(1)

		done := threads.AllocateThread()
		go func(rep KCRepository) {
			defer done()
			defer wg.Done()

			if repName != nil {
				// repNameが一致しない場合はスキップ
				repNameInRep, err := rep.GetRepName(ctx)
				if err != nil {
					errch <- fmt.Errorf("error at get rep name: %w", err)
					return
				}
				if repNameInRep != *repName {
					return
				}
			}

			matchKCsInRep, err := rep.GetKCHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKCsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get kc histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// KC集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchKCsInRep := <-ch:
			if matchKCsInRep == nil {
				continue loop
			}
			for _, kyou := range matchKCsInRep {
				if existKC, exist := kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if kyou.UpdateTime.After(existKC.UpdateTime) {
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

	kyouHistoriesList := []*KC{}
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

func (k KCRepositories) AddKCInfo(ctx context.Context, kc *KC) error {
	err := fmt.Errorf("not implements KCReps.AddKCInfo")
	return err
}
