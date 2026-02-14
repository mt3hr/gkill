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

type KmemoRepositories []KmemoRepository

func (k KmemoRepositories) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
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
			return nil, err
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

func (k KmemoRepositories) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
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
			return nil, err
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

func (k KmemoRepositories) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
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
			return nil, err
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

func (k KmemoRepositories) GetPath(ctx context.Context, id string) (string, error) {
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

func (k KmemoRepositories) UpdateCache(ctx context.Context) error {
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
			return err
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

func (k KmemoRepositories) GetRepName(ctx context.Context) (string, error) {
	return "KmemoReps", nil
}

func (k KmemoRepositories) Close(ctx context.Context) error {
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
			return err
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

func (k KmemoRepositories) FindKmemo(ctx context.Context, query *find.FindQuery) ([]Kmemo, error) {
	matchKmemos := map[string]Kmemo{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Kmemo, len(k))
	errch := make(chan error, len(k))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchKmemosInRep, err := rep.FindKmemo(ctx, query)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKmemosInRep
		})
		if err != nil {
			return nil, err
		}
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
				key := kyou.ID
				if query.OnlyLatestData == nil || !*query.OnlyLatestData {
					key += fmt.Sprintf("%d", kyou.UpdateTime.Unix())
				}

				if existKmemo, exist := matchKmemos[key]; exist {
					if kyou.UpdateTime.After(existKmemo.UpdateTime) {
						matchKmemos[key] = kyou
					}
				} else {
					matchKmemos[key] = kyou
				}
			}
		default:
			break loop
		}
	}

	matchKmemosList := []Kmemo{}
	for _, kyou := range matchKmemos {

		matchKmemosList = append(matchKmemosList, kyou)
	}

	return matchKmemosList, nil
}

func (k KmemoRepositories) GetKmemo(ctx context.Context, id string, updateTime *time.Time) (*Kmemo, error) {
	var matchKmemo *Kmemo
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Kmemo, len(k))
	errch := make(chan error, len(k))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchKmemoInRep, err := rep.GetKmemo(ctx, id, updateTime)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKmemoInRep
		})
		if err != nil {
			return nil, err
		}
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
}

func (k KmemoRepositories) GetKmemoHistories(ctx context.Context, id string) ([]Kmemo, error) {
	kyouHistories := map[string]Kmemo{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Kmemo, len(k))
	errch := make(chan error, len(k))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range k {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchKmemosInRep, err := rep.GetKmemoHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKmemosInRep
		})
		if err != nil {
			return nil, err
		}
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
					if kyou.UpdateTime.After(existKmemo.UpdateTime) {
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

	kyouHistoriesList := []Kmemo{}
	for _, kyou := range kyouHistories {

		kyouHistoriesList = append(kyouHistoriesList, kyou)
	}

	sort.Slice(kyouHistoriesList, func(i, j int) bool {
		return kyouHistoriesList[i].UpdateTime.After(kyouHistoriesList[j].UpdateTime)
	})

	return kyouHistoriesList, nil
}

func (k KmemoRepositories) GetKmemoHistoriesByRepName(ctx context.Context, id string, repName *string) ([]Kmemo, error) {
	kyouHistories := map[string]Kmemo{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Kmemo, len(k))
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

			matchKmemosInRep, err := rep.GetKmemoHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKmemosInRep
		})
		if err != nil {
			return nil, err
		}
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
					if kyou.UpdateTime.After(existKmemo.UpdateTime) {
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

	kyouHistoriesList := []Kmemo{}
	for _, kyou := range kyouHistories {

		kyouHistoriesList = append(kyouHistoriesList, kyou)
	}

	sort.Slice(kyouHistoriesList, func(i, j int) bool {
		return kyouHistoriesList[i].UpdateTime.After(kyouHistoriesList[j].UpdateTime)
	})

	return kyouHistoriesList, nil
}

func (k KmemoRepositories) AddKmemoInfo(ctx context.Context, kmemo Kmemo) error {
	err := fmt.Errorf("not implements KmemoReps.AddKmemoInfo")
	return err
}

func (k KmemoRepositories) UnWrapTyped() ([]KmemoRepository, error) {
	unwraped := []KmemoRepository{}
	for _, rep := range k {
		u, err := rep.UnWrapTyped()
		if err != nil {
			return nil, err
		}
		unwraped = append(unwraped, u...)
	}
	return unwraped, nil
}

func (k KmemoRepositories) UnWrap() ([]Repository, error) {
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
