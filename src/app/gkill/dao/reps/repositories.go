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

type Repositories []Repository

func (r Repositories) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	matchKyous := map[string][]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan map[string][]*Kyou, len(r))
	errch := make(chan error, len(r))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range r {
		_ = threads.Go(ctx, wg, func() {
			func(rep Repository) {
				matchKyousInRep, err := rep.FindKyous(ctx, query)
				if err != nil {
					repName, _ := rep.GetRepName(ctx)
					err = fmt.Errorf("error at %s: %w", repName, err)
					errch <- err
					return
				}
				ch <- matchKyousInRep
			}(rep)
		})
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
						matchKyous[key] = []*Kyou{}
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

func (r Repositories) Close(ctx context.Context) error {
	reps, err := r.UnWrap()
	if err != nil {
		return err
	}

	existErr := false
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(reps))
	defer close(errch)

	// 並列処理
	for _, rep := range reps {
		_ = threads.Go(ctx, wg, func() {
			func(rep Repository) {
				err = rep.Close(ctx)
				if err != nil {
					errch <- err
					return
				}
			}(rep)
		})
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

func (r Repositories) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	var matchKyou *Kyou
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Kyou, len(r))
	errch := make(chan error, len(r))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range r {
		_ = threads.Go(ctx, wg, func() {
			func(rep Repository) {
				matchKyouInRep, err := rep.GetKyou(ctx, id, updateTime)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchKyouInRep
			}(rep)
		})
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

func (r Repositories) UpdateCache(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(r))
	defer close(errch)

	// UpdateCache並列処理
	for _, rep := range r {
		func(rep Repository) {
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

func (r Repositories) GetPath(ctx context.Context, id string) (string, error) {
	// 並列処理
	matchPaths := []string{}
	trueValue := true
	ids := []string{id}
	for _, rep := range r {
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

func (r Repositories) GetRepName(ctx context.Context) (string, error) {
	return "Reps", nil
}

func (r Repositories) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	kyouHistories := map[string]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kyou, len(r))
	errch := make(chan error, len(r))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range r {
		_ = threads.Go(ctx, wg, func() {
			func(rep Repository) {
				matchKyousInRep, err := rep.GetKyouHistories(ctx, id)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchKyousInRep
			}(rep)
		})
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
			for _, kyou := range matchKyousInRep {
				if kyou == nil {
					continue
				}
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

func (r Repositories) GetKyouHistoriesByRepName(ctx context.Context, id string, repName *string) ([]*Kyou, error) {
	kyouHistories := map[string]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kyou, len(r))
	errch := make(chan error, len(r))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range r {
		_ = threads.Go(ctx, wg, func() {
			func(rep Repository) {
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

				matchKyousInRep, err := rep.GetKyouHistories(ctx, id)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchKyousInRep
			}(rep)
		})
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
			for _, kyou := range matchKyousInRep {
				if kyou == nil {
					continue
				}
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

func (r Repositories) UnWrap() ([]Repository, error) {
	repositories := []Repository{}
	for _, rep := range r {
		unwraped, err := rep.UnWrap()
		if err != nil {
			return nil, err
		}
		repositories = append(repositories, unwraped...)
	}
	return repositories, nil
}
