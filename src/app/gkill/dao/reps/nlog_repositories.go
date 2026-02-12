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

type NlogRepositories []NlogRepository

func (n NlogRepositories) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	matchKyous := map[string][]Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan map[string][]Kyou, len(n))
	errch := make(chan error, len(n))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range n {
		_ = threads.Go(ctx, wg, func() {
			func(rep NlogRepository) {
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

func (n NlogRepositories) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	var matchKyou *Kyou
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Kyou, len(n))
	errch := make(chan error, len(n))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range n {
		_ = threads.Go(ctx, wg, func() {
			func(rep NlogRepository) {
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

func (n NlogRepositories) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	kyouHistories := map[string]Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Kyou, len(n))
	errch := make(chan error, len(n))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range n {
		_ = threads.Go(ctx, wg, func() {
			func(rep NlogRepository) {
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

	kyouHistoriesList := []Kyou{}
	for _, kyou := range kyouHistories {

		kyouHistoriesList = append(kyouHistoriesList, kyou)
	}

	sort.Slice(kyouHistoriesList, func(i, j int) bool {
		return kyouHistoriesList[i].UpdateTime.After(kyouHistoriesList[j].UpdateTime)
	})

	return kyouHistoriesList, nil
}

func (n NlogRepositories) GetPath(ctx context.Context, id string) (string, error) {
	// 並列処理
	matchPaths := []string{}
	trueValue := true
	ids := []string{id}
	for _, rep := range n {
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

func (n NlogRepositories) UpdateCache(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(n))
	defer close(errch)

	// 並列処理
	for _, rep := range n {
		_ = threads.Go(ctx, wg, func() {
			func(rep NlogRepository) {
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

func (n NlogRepositories) GetRepName(ctx context.Context) (string, error) {
	return "NlogReps", nil
}

func (n NlogRepositories) Close(ctx context.Context) error {
	reps, err := n.UnWrapTyped()
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
			func(rep NlogRepository) {
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

func (n NlogRepositories) FindNlog(ctx context.Context, query *find.FindQuery) ([]Nlog, error) {
	matchNlogs := map[string]Nlog{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Nlog, len(n))
	errch := make(chan error, len(n))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range n {
		_ = threads.Go(ctx, wg, func() {
			func(rep NlogRepository) {
				matchNlogsInRep, err := rep.FindNlog(ctx, query)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchNlogsInRep
			}(rep)
		})
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
				key := kyou.ID
				if query.OnlyLatestData == nil || !*query.OnlyLatestData {
					key += fmt.Sprintf("%d", kyou.UpdateTime.Unix())
				}
				if existNlog, exist := matchNlogs[key]; exist {
					if kyou.UpdateTime.After(existNlog.UpdateTime) {
						matchNlogs[key] = kyou
					}
				} else {
					matchNlogs[key] = kyou
				}
			}
		default:
			break loop
		}
	}

	matchNlogsList := []Nlog{}
	for _, kyou := range matchNlogs {

		matchNlogsList = append(matchNlogsList, kyou)
	}
	return matchNlogsList, nil
}

func (n NlogRepositories) GetNlog(ctx context.Context, id string, updateTime *time.Time) (*Nlog, error) {
	var matchNlog *Nlog
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Nlog, len(n))
	errch := make(chan error, len(n))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range n {
		_ = threads.Go(ctx, wg, func() {
			func(rep NlogRepository) {
				matchNlogInRep, err := rep.GetNlog(ctx, id, updateTime)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchNlogInRep
			}(rep)
		})
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

func (n NlogRepositories) GetNlogHistories(ctx context.Context, id string) ([]Nlog, error) {
	kyouHistories := map[string]Nlog{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Nlog, len(n))
	errch := make(chan error, len(n))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range n {
		_ = threads.Go(ctx, wg, func() {
			func(rep NlogRepository) {
				matchNlogsInRep, err := rep.GetNlogHistories(ctx, id)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchNlogsInRep
			}(rep)
		})
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
					if kyou.UpdateTime.After(existNlog.UpdateTime) {
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

	kyouHistoriesList := []Nlog{}
	for _, kyou := range kyouHistories {

		kyouHistoriesList = append(kyouHistoriesList, kyou)
	}

	sort.Slice(kyouHistoriesList, func(i, j int) bool {
		return kyouHistoriesList[i].UpdateTime.After(kyouHistoriesList[j].UpdateTime)
	})

	return kyouHistoriesList, nil
}

func (n NlogRepositories) GetNlogHistoriesByRepName(ctx context.Context, id string, repName *string) ([]Nlog, error) {
	kyouHistories := map[string]Nlog{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Nlog, len(n))
	errch := make(chan error, len(n))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range n {
		_ = threads.Go(ctx, wg, func() {
			func(rep NlogRepository) {
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

				matchNlogsInRep, err := rep.GetNlogHistories(ctx, id)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchNlogsInRep
			}(rep)
		})
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
					if kyou.UpdateTime.After(existNlog.UpdateTime) {
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

	kyouHistoriesList := []Nlog{}
	for _, kyou := range kyouHistories {

		kyouHistoriesList = append(kyouHistoriesList, kyou)
	}

	sort.Slice(kyouHistoriesList, func(i, j int) bool {
		return kyouHistoriesList[i].UpdateTime.After(kyouHistoriesList[j].UpdateTime)
	})

	return kyouHistoriesList, nil
}

func (n NlogRepositories) AddNlogInfo(ctx context.Context, nlog Nlog) error {
	err := fmt.Errorf("not implements NlogReps.AddNlogInfo")
	return err
}

func (n NlogRepositories) UnWrapTyped() ([]NlogRepository, error) {
	unwraped := []NlogRepository{}
	for _, rep := range n {
		u, err := rep.UnWrapTyped()
		if err != nil {
			return nil, err
		}
		unwraped = append(unwraped, u...)
	}
	return unwraped, nil
}

func (n NlogRepositories) UnWrap() ([]Repository, error) {
	repositories := []Repository{}
	for _, rep := range n {
		unwraped, err := rep.UnWrap()
		if err != nil {
			return nil, err
		}
		repositories = append(repositories, unwraped...)
	}
	return repositories, nil
}
