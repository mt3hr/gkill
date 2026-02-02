package reps

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/app/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/threads"
)

type LantanaRepositories []LantanaRepository

func (l LantanaRepositories) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	matchKyous := map[string][]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan map[string][]*Kyou, len(l))
	errch := make(chan error, len(l))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range l {
		_ = threads.Go(ctx, wg, func() {
			func(rep LantanaRepository) {
				matchKyousInRep, err := rep.FindKyous(ctx, query)
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

func (l LantanaRepositories) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	var matchKyou *Kyou
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Kyou, len(l))
	errch := make(chan error, len(l))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range l {
		_ = threads.Go(ctx, wg, func() {
			func(rep LantanaRepository) {
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

func (l LantanaRepositories) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	kyouHistories := map[string]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kyou, len(l))
	errch := make(chan error, len(l))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range l {
		_ = threads.Go(ctx, wg, func() {
			func(rep LantanaRepository) {
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

func (l LantanaRepositories) GetPath(ctx context.Context, id string) (string, error) {
	// 並列処理
	matchPaths := []string{}
	trueValue := true
	ids := []string{id}
	for _, rep := range l {
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

func (l LantanaRepositories) UpdateCache(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(l))
	defer close(errch)

	// 並列処理
	for _, rep := range l {
		_ = threads.Go(ctx, wg, func() {
			func(rep LantanaRepository) {
				err = rep.UpdateCache(ctx)
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

func (l LantanaRepositories) GetRepName(ctx context.Context) (string, error) {
	return "LantanaReps", nil
}

func (l LantanaRepositories) Close(ctx context.Context) error {
	reps, err := l.UnWrapTyped()
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
			func(rep LantanaRepository) {
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

func (l LantanaRepositories) FindLantana(ctx context.Context, query *find.FindQuery) ([]*Lantana, error) {
	matchLantanas := map[string]*Lantana{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Lantana, len(l))
	errch := make(chan error, len(l))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range l {
		_ = threads.Go(ctx, wg, func() {
			func(rep LantanaRepository) {
				matchLantanasInRep, err := rep.FindLantana(ctx, query)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchLantanasInRep
			}(rep)
		})
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find lantana: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Lantana集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchLantanasInRep := <-ch:
			if matchLantanasInRep == nil {
				continue loop
			}
			for _, kyou := range matchLantanasInRep {
				key := kyou.ID
				if query.OnlyLatestData == nil || !*query.OnlyLatestData {
					key += fmt.Sprintf("%d", kyou.UpdateTime.Unix())
				}
				if existLantana, exist := matchLantanas[key]; exist {
					if kyou.UpdateTime.After(existLantana.UpdateTime) {
						matchLantanas[key] = kyou
					}
				} else {
					matchLantanas[key] = kyou
				}
			}
		default:
			break loop
		}
	}

	matchLantanasList := []*Lantana{}
	for _, kyou := range matchLantanas {
		if kyou == nil {
			continue
		}
		matchLantanasList = append(matchLantanasList, kyou)
	}
	return matchLantanasList, nil
}

func (l LantanaRepositories) GetLantana(ctx context.Context, id string, updateTime *time.Time) (*Lantana, error) {
	var matchLantana *Lantana
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Lantana, len(l))
	errch := make(chan error, len(l))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range l {
		_ = threads.Go(ctx, wg, func() {
			func(rep LantanaRepository) {
				matchLantanaInRep, err := rep.GetLantana(ctx, id, updateTime)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchLantanaInRep
			}(rep)
		})
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get lantana: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Lantana集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchLantanaInRep := <-ch:
			if matchLantanaInRep == nil {
				continue loop
			}
			if matchLantana != nil {
				if matchLantanaInRep.UpdateTime.Before(matchLantana.UpdateTime) {
					matchLantana = matchLantanaInRep
				}
			} else {
				matchLantana = matchLantanaInRep
			}
		default:
			break loop
		}
	}

	return matchLantana, nil
}

func (l LantanaRepositories) GetLantanaHistories(ctx context.Context, id string) ([]*Lantana, error) {
	kyouHistories := map[string]*Lantana{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Lantana, len(l))
	errch := make(chan error, len(l))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range l {
		_ = threads.Go(ctx, wg, func() {
			func(rep LantanaRepository) {
				matchLantanasInRep, err := rep.GetLantanaHistories(ctx, id)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchLantanasInRep
			}(rep)
		})
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get lantana histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Lantana集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchLantanasInRep := <-ch:
			if matchLantanasInRep == nil {
				continue loop
			}
			for _, kyou := range matchLantanasInRep {
				if existLantana, exist := kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if kyou.UpdateTime.After(existLantana.UpdateTime) {
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

	kyouHistoriesList := []*Lantana{}
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

func (l LantanaRepositories) GetLantanaHistoriesByRepName(ctx context.Context, id string, repName *string) ([]*Lantana, error) {
	kyouHistories := map[string]*Lantana{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Lantana, len(l))
	errch := make(chan error, len(l))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range l {
		_ = threads.Go(ctx, wg, func() {
			func(rep LantanaRepository) {
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

				matchLantanasInRep, err := rep.GetLantanaHistories(ctx, id)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchLantanasInRep
			}(rep)
		})
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get lantana histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Lantana集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchLantanasInRep := <-ch:
			if matchLantanasInRep == nil {
				continue loop
			}
			for _, kyou := range matchLantanasInRep {
				if existLantana, exist := kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if kyou.UpdateTime.After(existLantana.UpdateTime) {
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

	kyouHistoriesList := []*Lantana{}
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

func (l LantanaRepositories) AddLantanaInfo(ctx context.Context, lantana *Lantana) error {
	err := fmt.Errorf("not implements LantanaReps.AddLantanaInfo")
	return err
}

func (l LantanaRepositories) UnWrapTyped() ([]LantanaRepository, error) {
	unwraped := []LantanaRepository{}
	for _, rep := range l {
		u, err := rep.UnWrapTyped()
		if err != nil {
			return nil, err
		}
		unwraped = append(unwraped, u...)
	}
	return unwraped, nil
}

func (l LantanaRepositories) UnWrap() ([]Repository, error) {
	repositories := []Repository{}
	for _, rep := range l {
		unwraped, err := rep.UnWrap()
		if err != nil {
			return nil, err
		}
		repositories = append(repositories, unwraped...)
	}
	return repositories, nil
}

func (l LantanaRepositories) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]*gkill_cache.LatestDataRepositoryAddress, error) {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	latestDataRepositoryAddressCh := make(chan []*gkill_cache.LatestDataRepositoryAddress, len(l))
	errch := make(chan error, len(l))
	defer close(latestDataRepositoryAddressCh)
	defer close(errch)

	// 並列処理
	for _, rep := range l {
		_ = threads.Go(ctx, wg, func() {
			func(rep LantanaRepository) {
				latestDataRepositoryAddresses, err := rep.GetLatestDataRepositoryAddress(ctx, updateCache)
				if err != nil {
					errch <- err
					return
				}
				latestDataRepositoryAddressCh <- latestDataRepositoryAddresses
			}(rep)
		})
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get latest data repository address: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	latestDataRepositoryAddresses := []*gkill_cache.LatestDataRepositoryAddress{}
loop:
	for {
		select {
		case latestDataRepositoryAddressInRep := <-latestDataRepositoryAddressCh:
			latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, latestDataRepositoryAddressInRep...)
		default:
			break loop
		}
	}

	return latestDataRepositoryAddresses, nil
}
