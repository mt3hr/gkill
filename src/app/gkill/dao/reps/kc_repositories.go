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

func (k KCRepositories) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	matchKyous := map[string][]Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan map[string][]Kyou, len(k))
	errch := make(chan error, len(k))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchKyousInRep, err := rep.FindKyous(ctx, query)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKyousInRep
		})
		if err != nil {
			errch <- err
		}
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
					key := kyou.ID
					if query.OnlyLatestData == nil || !*query.OnlyLatestData {
						key += fmt.Sprintf("%d", kyou.UpdateTime.Unix())
					}
					if _, exist := matchKyous[key]; !exist {
						matchKyous[key] = []Kyou{}
					}
					matchKyous[key] = append(matchKyous[key], kyou)
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
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchKyouInRep, err := rep.GetKyou(ctx, id, updateTime)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKyouInRep
		})
		if err != nil {
			errch <- err
		}
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

func (k KCRepositories) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	kyouHistories := map[string]Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Kyou, len(k))
	errch := make(chan error, len(k))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchKyousInRep, err := rep.GetKyouHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKyousInRep
		})
		if err != nil {
			errch <- err
		}
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

	kyouHistoriesList := []Kyou{}
	for _, kyou := range kyouHistories {

		kyouHistoriesList = append(kyouHistoriesList, kyou)
	}

	sort.Slice(kyouHistoriesList, func(i, j int) bool {
		return kyouHistoriesList[i].UpdateTime.After(kyouHistoriesList[j].UpdateTime)
	})

	return kyouHistoriesList, nil
}

func (k KCRepositories) GetPath(ctx context.Context, id string) (string, error) {
	// 並列処理
	matchPaths := []string{}
	trueValue := true
	ids := []string{id}
	for _, rep := range k {
		query := &find.FindQuery{
			IDs:    &ids,
			UseIDs: &trueValue,
		}
		kyous, err := rep.FindKyous(ctx, query)
		if len(kyous) == 0 || err != nil {
			continue
		}
		matchPathInRep, err := rep.GetPath(ctx, id)
		if err != nil {
			continue
		}
		matchPaths = append(matchPaths, matchPathInRep)
	}
	if len(matchPaths) == 0 {
		return "", fmt.Errorf("not found path for id: %s", id)
	}
	return matchPaths[0], nil
}

func (k KCRepositories) UpdateCache(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(k))
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			err = rep.UpdateCache(ctx)
			if err != nil {
				errch <- err
				return
			}
		})
		if err != nil {
			errch <- err
		}
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
	reps, err := k.UnWrapTyped()
	if err != nil {
		return err
	}

	existErr := false
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(reps))
	defer close(errch)

	// 並列処理
	for _, rep := range reps {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			err = rep.Close(ctx)
			if err != nil {
				errch <- err
				return
			}
		})
		if err != nil {
			errch <- err
		}
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

func (k KCRepositories) FindKC(ctx context.Context, query *find.FindQuery) ([]KC, error) {
	matchKCs := map[string]KC{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []KC, len(k))
	errch := make(chan error, len(k))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchKCsInRep, err := rep.FindKC(ctx, query)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKCsInRep
		})
		if err != nil {
			errch <- err
		}
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
				key := kyou.ID
				if query.OnlyLatestData == nil || !*query.OnlyLatestData {
					key += fmt.Sprintf("%d", kyou.UpdateTime.Unix())
				}
				if existKC, exist := matchKCs[key]; exist {
					if kyou.UpdateTime.After(existKC.UpdateTime) {
						matchKCs[key] = kyou
					}
				} else {
					matchKCs[key] = kyou
				}
			}
		default:
			break loop
		}
	}

	matchKCsList := []KC{}
	for _, kyou := range matchKCs {

		matchKCsList = append(matchKCsList, kyou)
	}
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
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchKCInRep, err := rep.GetKC(ctx, id, updateTime)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKCInRep
		})
		if err != nil {
			errch <- err
		}
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

func (k KCRepositories) GetKCHistories(ctx context.Context, id string) ([]KC, error) {
	kyouHistories := map[string]KC{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []KC, len(k))
	errch := make(chan error, len(k))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchKCsInRep, err := rep.GetKCHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKCsInRep
		})
		if err != nil {
			errch <- err
		}
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

	kyouHistoriesList := []KC{}
	for _, kyou := range kyouHistories {

		kyouHistoriesList = append(kyouHistoriesList, kyou)
	}

	sort.Slice(kyouHistoriesList, func(i, j int) bool {
		return kyouHistoriesList[i].UpdateTime.After(kyouHistoriesList[j].UpdateTime)
	})

	return kyouHistoriesList, nil
}

func (k KCRepositories) GetKCHistoriesByRepName(ctx context.Context, id string, repName *string) ([]KC, error) {
	kyouHistories := map[string]KC{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []KC, len(k))
	errch := make(chan error, len(k))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		rep := rep
		err := threads.Go(ctx, wg, func() {
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
		})
		if err != nil {
			errch <- err
		}
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

	kyouHistoriesList := []KC{}
	for _, kyou := range kyouHistories {

		kyouHistoriesList = append(kyouHistoriesList, kyou)
	}

	sort.Slice(kyouHistoriesList, func(i, j int) bool {
		return kyouHistoriesList[i].UpdateTime.After(kyouHistoriesList[j].UpdateTime)
	})

	return kyouHistoriesList, nil
}

func (k KCRepositories) AddKCInfo(ctx context.Context, kc KC) error {
	err := fmt.Errorf("not implements KCReps.AddKCInfo")
	return err
}

func (k KCRepositories) UnWrapTyped() ([]KCRepository, error) {
	unwraped := []KCRepository{}
	for _, rep := range k {
		u, err := rep.UnWrapTyped()
		if err != nil {
			return nil, err
		}
		unwraped = append(unwraped, u...)
	}
	return unwraped, nil
}

func (k KCRepositories) UnWrap() ([]Repository, error) {
	repositories := []Repository{}
	for _, rep := range k {
		unwraped, err := rep.UnWrap()
		if err != nil {
			return nil, err
		}
		repositories = append(repositories, unwraped...)
	}
	return repositories, nil
}
