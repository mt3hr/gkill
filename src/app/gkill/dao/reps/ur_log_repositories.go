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

type URLogRepositories []URLogRepository

func (u URLogRepositories) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	matchKyous := map[string][]Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan map[string][]Kyou, len(u))
	errch := make(chan error, len(u))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range u {
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

func (u URLogRepositories) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	var matchKyou *Kyou
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Kyou, len(u))
	errch := make(chan error, len(u))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range u {
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

func (u URLogRepositories) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	kyouHistories := map[string]Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Kyou, len(u))
	errch := make(chan error, len(u))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range u {
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

func (u URLogRepositories) GetPath(ctx context.Context, id string) (string, error) {
	// 並列処理
	matchPaths := []string{}
	trueValue := true
	ids := []string{id}
	for _, rep := range u {
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

func (u URLogRepositories) UpdateCache(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(u))
	defer close(errch)

	// 並列処理
	for _, rep := range u {
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

func (u URLogRepositories) GetRepName(ctx context.Context) (string, error) {
	return "URLogReps", nil
}

func (u URLogRepositories) Close(ctx context.Context) error {
	reps, err := u.UnWrapTyped()
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

func (u URLogRepositories) FindURLog(ctx context.Context, query *find.FindQuery) ([]URLog, error) {
	matchURLogs := map[string]URLog{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []URLog, len(u))
	errch := make(chan error, len(u))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range u {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchURLogsInRep, err := rep.FindURLog(ctx, query)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchURLogsInRep
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
			err = fmt.Errorf("error at find urlog: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// URLog集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchURLogsInRep := <-ch:
			if matchURLogsInRep == nil {
				continue loop
			}
			for _, kyou := range matchURLogsInRep {
				key := kyou.ID
				if query.OnlyLatestData == nil || !*query.OnlyLatestData {
					key += fmt.Sprintf("%d", kyou.UpdateTime.Unix())
				}
				if existURLog, exist := matchURLogs[key]; exist {
					if kyou.UpdateTime.After(existURLog.UpdateTime) {
						matchURLogs[key] = kyou
					}
				} else {
					matchURLogs[key] = kyou
				}
			}
		default:
			break loop
		}
	}

	matchURLogsList := []URLog{}
	for _, kyou := range matchURLogs {

		matchURLogsList = append(matchURLogsList, kyou)
	}
	return matchURLogsList, nil
}

func (u URLogRepositories) GetURLog(ctx context.Context, id string, updateTime *time.Time) (*URLog, error) {
	var matchURLog *URLog
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *URLog, len(u))
	errch := make(chan error, len(u))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range u {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchURLogInRep, err := rep.GetURLog(ctx, id, updateTime)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchURLogInRep
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
			err = fmt.Errorf("error at get urlog: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// URLog集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchURLogInRep := <-ch:
			if matchURLogInRep == nil {
				continue loop
			}
			if matchURLog != nil {
				if matchURLogInRep.UpdateTime.Before(matchURLog.UpdateTime) {
					matchURLog = matchURLogInRep
				}
			} else {
				matchURLog = matchURLogInRep
			}
		default:
			break loop
		}
	}

	return matchURLog, nil
}

func (u URLogRepositories) GetURLogHistories(ctx context.Context, id string) ([]URLog, error) {
	kyouHistories := map[string]URLog{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []URLog, len(u))
	errch := make(chan error, len(u))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range u {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchURLogsInRep, err := rep.GetURLogHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchURLogsInRep
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
			err = fmt.Errorf("error at find get urlog histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// URLog集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchURLogsInRep := <-ch:
			if matchURLogsInRep == nil {
				continue loop
			}
			for _, kyou := range matchURLogsInRep {
				if existURLog, exist := kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if kyou.UpdateTime.After(existURLog.UpdateTime) {
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

	kyouHistoriesList := []URLog{}
	for _, kyou := range kyouHistories {

		kyouHistoriesList = append(kyouHistoriesList, kyou)
	}

	sort.Slice(kyouHistoriesList, func(i, j int) bool {
		return kyouHistoriesList[i].UpdateTime.After(kyouHistoriesList[j].UpdateTime)
	})

	return kyouHistoriesList, nil
}

func (u URLogRepositories) GetURLogHistoriesByRepName(ctx context.Context, id string, repName *string) ([]URLog, error) {
	kyouHistories := map[string]URLog{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []URLog, len(u))
	errch := make(chan error, len(u))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range u {
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

			matchURLogsInRep, err := rep.GetURLogHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchURLogsInRep
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
			err = fmt.Errorf("error at find get urlog histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// URLog集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchURLogsInRep := <-ch:
			if matchURLogsInRep == nil {
				continue loop
			}
			for _, kyou := range matchURLogsInRep {
				if existURLog, exist := kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if kyou.UpdateTime.After(existURLog.UpdateTime) {
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

	kyouHistoriesList := []URLog{}
	for _, kyou := range kyouHistories {

		kyouHistoriesList = append(kyouHistoriesList, kyou)
	}

	sort.Slice(kyouHistoriesList, func(i, j int) bool {
		return kyouHistoriesList[i].UpdateTime.After(kyouHistoriesList[j].UpdateTime)
	})

	return kyouHistoriesList, nil
}

func (u URLogRepositories) AddURLogInfo(ctx context.Context, urlog URLog) error {
	err := fmt.Errorf("not implements URLogReps.AddURLogInfo")
	return err
}

func (u URLogRepositories) UnWrapTyped() ([]URLogRepository, error) {
	unwraped := []URLogRepository{}
	for _, rep := range u {
		u, err := rep.UnWrapTyped()
		if err != nil {
			return nil, err
		}
		unwraped = append(unwraped, u...)
	}
	return unwraped, nil
}

func (u URLogRepositories) UnWrap() ([]Repository, error) {
	repositories := []Repository{}
	for _, rep := range u {
		unwraped, err := rep.UnWrap()
		if err != nil {
			return nil, err
		}
		repositories = append(repositories, unwraped...)
	}
	return repositories, nil
}
