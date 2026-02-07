package reps

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/threads"
)

type IDFKyouRepositories []IDFKyouRepository

func (i IDFKyouRepositories) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	matchKyous := map[string][]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan map[string][]*Kyou, len(i))
	errch := make(chan error, len(i))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range i {
		_ = threads.Go(ctx, wg, func() {
			func(rep IDFKyouRepository) {
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

func (i IDFKyouRepositories) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	var matchKyou *Kyou
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Kyou, len(i))
	errch := make(chan error, len(i))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range i {
		_ = threads.Go(ctx, wg, func() {
			func(rep IDFKyouRepository) {
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
		_ = threads.Go(ctx, wg, func() {
			func(rep IDFKyouRepository) {
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
	return kyouHistoriesList, nil
}

func (i IDFKyouRepositories) GetPath(ctx context.Context, id string) (string, error) {
	// 並列処理
	matchPaths := []string{}
	trueValue := true
	ids := []string{id}
	for _, rep := range i {
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

func (i IDFKyouRepositories) UpdateCache(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(i))
	defer close(errch)

	// 並列処理
	for _, rep := range i {
		_ = threads.Go(ctx, wg, func() {
			func(rep IDFKyouRepository) {
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

func (i IDFKyouRepositories) GetRepName(ctx context.Context) (string, error) {
	return "IDFKyouReps", nil
}

func (i IDFKyouRepositories) Close(ctx context.Context) error {
	reps, err := i.UnWrapTyped()
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
			func(rep IDFKyouRepository) {
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

func (i IDFKyouRepositories) FindIDFKyou(ctx context.Context, query *find.FindQuery) ([]*IDFKyou, error) {
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
		_ = threads.Go(ctx, wg, func() {
			func(rep IDFKyouRepository) {
				matchIDFKyousInRep, err := rep.FindIDFKyou(ctx, query)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchIDFKyousInRep
			}(rep)
		})
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
				key := kyou.ID
				if query.OnlyLatestData == nil || !*query.OnlyLatestData {
					key += fmt.Sprintf("%d", kyou.UpdateTime.Unix())
				}
				if existIDFKyou, exist := matchIDFKyous[key]; exist {
					if kyou.UpdateTime.After(existIDFKyou.UpdateTime) {
						matchIDFKyous[key] = kyou
					}
				} else {
					matchIDFKyous[key] = kyou
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
	return matchIDFKyousList, nil
}

func (i IDFKyouRepositories) GetIDFKyou(ctx context.Context, id string, updateTime *time.Time) (*IDFKyou, error) {
	var matchIDFKyou *IDFKyou
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *IDFKyou, len(i))
	errch := make(chan error, len(i))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range i {
		_ = threads.Go(ctx, wg, func() {
			func(rep IDFKyouRepository) {
				matchIDFKyouInRep, err := rep.GetIDFKyou(ctx, id, updateTime)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchIDFKyouInRep
			}(rep)
		})
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
		_ = threads.Go(ctx, wg, func() {
			func(rep IDFKyouRepository) {
				matchIDFKyousInRep, err := rep.GetIDFKyouHistories(ctx, id)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchIDFKyousInRep
			}(rep)
		})
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
					if kyou.UpdateTime.After(existIDFKyou.UpdateTime) {
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

func (i IDFKyouRepositories) GetIDFKyouHistoriesByRepName(ctx context.Context, id string, repName *string) ([]*IDFKyou, error) {
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
		_ = threads.Go(ctx, wg, func() {
			func(rep IDFKyouRepository) {
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

				matchIDFKyousInRep, err := rep.GetIDFKyouHistories(ctx, id)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchIDFKyousInRep
			}(rep)
		})
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
					if kyou.UpdateTime.After(existIDFKyou.UpdateTime) {
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

func (i IDFKyouRepositories) HandleFileServe(w http.ResponseWriter, r *http.Request) {
	http.NotFoundHandler().ServeHTTP(w, r)
}

func (i IDFKyouRepositories) IDF(ctx context.Context) error {
	err := fmt.Errorf("not implements IDFKyouReps.IDF")
	return err
}

func (i IDFKyouRepositories) AddIDFKyouInfo(ctx context.Context, idfKyou *IDFKyou) error {
	err := fmt.Errorf("not implements IDFKyouReps.AddIDFKyouInfo")
	return err
}

func (i IDFKyouRepositories) GenerateThumbCache(ctx context.Context) error {
	unwrapedReps, err := i.UnWrapTyped()
	if err != nil {
		err = fmt.Errorf("error at generate thumb cache at idf kyou repositories: %w", err)
		return err
	}

	wg := &sync.WaitGroup{}
	for _, unwrapedRep := range unwrapedReps {
		_ = threads.Go(ctx, wg, func() {
			func(unwrapedRep IDFKyouRepository) {
				err := unwrapedRep.GenerateThumbCache(ctx)
				if err != nil {
					err = fmt.Errorf("error at generate thumb cache at idf kyou repositories in rep: %w", err)
					slog.Log(ctx, gkill_log.Error, "error", "error", err)
				}
			}(unwrapedRep)
		})
	}
	wg.Wait()
	return nil
}

func (i IDFKyouRepositories) ClearThumbCache() error {
	unwrapedReps, err := i.UnWrapTyped()
	if err != nil {
		err = fmt.Errorf("error at clear thumb cache at idf kyou repositories: %w", err)
		return err
	}
	for _, unwrapedRep := range unwrapedReps {
		err := unwrapedRep.ClearThumbCache()
		if err != nil {
			err = fmt.Errorf("error at clear thumb cache at idf kyou repositories in rep: %w", err)
			return err
		}
	}
	return err
}

func (i IDFKyouRepositories) GenerateVideoCache(ctx context.Context) error {
	unwrapedReps, err := i.UnWrapTyped()
	if err != nil {
		err = fmt.Errorf("error at generate video cache at idf kyou repositories: %w", err)
		return err
	}

	wg := &sync.WaitGroup{}
	for _, unwrapedRep := range unwrapedReps {
		_ = threads.Go(ctx, wg, func() {
			func(unwrapedRep IDFKyouRepository) {
				err := unwrapedRep.GenerateVideoCache(ctx)
				if err != nil {
					err = fmt.Errorf("error at generate video cache at idf kyou repositories in rep: %w", err)
					slog.Log(ctx, gkill_log.Error, "error", "error", err)
				}
			}(unwrapedRep)
		})
	}
	wg.Wait()
	return nil
}

func (i IDFKyouRepositories) ClearVideoCache() error {
	unwrapedReps, err := i.UnWrapTyped()
	if err != nil {
		err = fmt.Errorf("error at clear video cache at idf kyou repositories: %w", err)
		return err
	}
	for _, unwrapedRep := range unwrapedReps {
		err := unwrapedRep.ClearVideoCache()
		if err != nil {
			err = fmt.Errorf("error at clear video cache at idf kyou repositories in rep: %w", err)
			return err
		}
	}
	return err
}

func (i IDFKyouRepositories) UnWrapTyped() ([]IDFKyouRepository, error) {
	unwraped := []IDFKyouRepository{}
	for _, rep := range i {
		u, err := rep.UnWrapTyped()
		if err != nil {
			return nil, err
		}
		unwraped = append(unwraped, u...)
	}
	return unwraped, nil
}

func (i IDFKyouRepositories) UnWrap() ([]Repository, error) {
	repositories := []Repository{}
	for _, rep := range i {
		unwraped, err := rep.UnWrap()
		if err != nil {
			return nil, err
		}
		repositories = append(repositories, unwraped...)
	}
	return repositories, nil
}
