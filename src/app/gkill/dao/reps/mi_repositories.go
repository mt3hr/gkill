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

type MiRepositories []MiRepository

func (m MiRepositories) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	matchKyous := map[string][]Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan map[string][]Kyou, len(m))
	errch := make(chan error, len(m))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range m {
		_ = threads.Go(ctx, wg, func() {
			func(rep MiRepository) {
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

func (m MiRepositories) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	var matchKyou *Kyou
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Kyou, len(m))
	errch := make(chan error, len(m))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range m {
		_ = threads.Go(ctx, wg, func() {
			func(rep MiRepository) {
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

func (m MiRepositories) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	kyouHistories := map[string]Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Kyou, len(m))
	errch := make(chan error, len(m))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range m {
		_ = threads.Go(ctx, wg, func() {
			func(rep MiRepository) {
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

func (m MiRepositories) GetPath(ctx context.Context, id string) (string, error) {
	// 並列処理
	matchPaths := []string{}
	trueValue := true
	ids := []string{id}
	for _, rep := range m {
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

func (m MiRepositories) UpdateCache(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(m))
	defer close(errch)

	// 並列処理
	for _, rep := range m {
		_ = threads.Go(ctx, wg, func() {
			func(rep MiRepository) {
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

func (m MiRepositories) GetRepName(ctx context.Context) (string, error) {
	return "MiReps", nil
}

func (m MiRepositories) Close(ctx context.Context) error {
	reps, err := m.UnWrapTyped()
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
			func(rep MiRepository) {
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

func (m MiRepositories) FindMi(ctx context.Context, query *find.FindQuery) ([]Mi, error) {
	matchMis := map[string]Mi{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Mi, len(m))
	errch := make(chan error, len(m))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range m {
		_ = threads.Go(ctx, wg, func() {
			func(rep MiRepository) {
				matchMisInRep, err := rep.FindMi(ctx, query)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchMisInRep
			}(rep)
		})
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find mi: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Mi集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchMisInRep := <-ch:
			if matchMisInRep == nil {
				continue loop
			}
			for _, kyou := range matchMisInRep {
				key := kyou.ID
				if query.OnlyLatestData == nil || !*query.OnlyLatestData {
					key += fmt.Sprintf("%d", kyou.UpdateTime.Unix())
				}
				if existMi, exist := matchMis[key]; exist {
					if kyou.UpdateTime.After(existMi.UpdateTime) {
						matchMis[key] = kyou
					}
				} else {
					matchMis[key] = kyou
				}
			}
		default:
			break loop
		}
	}

	matchMisList := []Mi{}
	for _, kyou := range matchMis {

		matchMisList = append(matchMisList, kyou)
	}
	return matchMisList, nil
}

func (m MiRepositories) GetMi(ctx context.Context, id string, updateTime *time.Time) (*Mi, error) {
	var matchMi *Mi
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Mi, len(m))
	errch := make(chan error, len(m))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range m {
		_ = threads.Go(ctx, wg, func() {
			func(rep MiRepository) {
				matchMiInRep, err := rep.GetMi(ctx, id, updateTime)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchMiInRep
			}(rep)
		})
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get mi: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Mi集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchMiInRep := <-ch:
			if matchMiInRep == nil {
				continue loop
			}
			if matchMi != nil {
				if matchMiInRep.UpdateTime.Before(matchMi.UpdateTime) {
					matchMi = matchMiInRep
				}
			} else {
				matchMi = matchMiInRep
			}
		default:
			break loop
		}
	}

	return matchMi, nil
}

func (m MiRepositories) GetMiHistories(ctx context.Context, id string) ([]Mi, error) {
	kyouHistories := map[string]Mi{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Mi, len(m))
	errch := make(chan error, len(m))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range m {
		_ = threads.Go(ctx, wg, func() {
			func(rep MiRepository) {
				matchMisInRep, err := rep.GetMiHistories(ctx, id)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchMisInRep
			}(rep)
		})
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get mi histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Mi集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchMisInRep := <-ch:
			if matchMisInRep == nil {
				continue loop
			}
			for _, kyou := range matchMisInRep {
				if existMi, exist := kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if kyou.UpdateTime.After(existMi.UpdateTime) {
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

	kyouHistoriesList := []Mi{}
	for _, kyou := range kyouHistories {

		kyouHistoriesList = append(kyouHistoriesList, kyou)
	}

	sort.Slice(kyouHistoriesList, func(i, j int) bool {
		return kyouHistoriesList[i].UpdateTime.After(kyouHistoriesList[j].UpdateTime)
	})

	return kyouHistoriesList, nil
}

func (m MiRepositories) GetMiHistoriesByRepName(ctx context.Context, id string, repName *string) ([]Mi, error) {
	kyouHistories := map[string]Mi{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Mi, len(m))
	errch := make(chan error, len(m))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range m {
		_ = threads.Go(ctx, wg, func() {
			func(rep MiRepository) {
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

				matchMisInRep, err := rep.GetMiHistories(ctx, id)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchMisInRep
			}(rep)
		})
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get mi histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Mi集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchMisInRep := <-ch:
			if matchMisInRep == nil {
				continue loop
			}
			for _, kyou := range matchMisInRep {
				if existMi, exist := kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if kyou.UpdateTime.After(existMi.UpdateTime) {
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

	kyouHistoriesList := []Mi{}
	for _, kyou := range kyouHistories {

		kyouHistoriesList = append(kyouHistoriesList, kyou)
	}

	sort.Slice(kyouHistoriesList, func(i, j int) bool {
		return kyouHistoriesList[i].UpdateTime.After(kyouHistoriesList[j].UpdateTime)
	})

	return kyouHistoriesList, nil
}

func (m MiRepositories) AddMiInfo(ctx context.Context, mi Mi) error {
	err := fmt.Errorf("not implements MiReps.AddMiInfo")
	return err
}

func (m MiRepositories) GetBoardNames(ctx context.Context) ([]string, error) {
	boardNames := map[string]interface{}{}

	trueValue := true

	findMiQuery := &find.FindQuery{}
	findMiQuery.OnlyLatestData = &trueValue
	mis, err := m.FindMi(ctx, findMiQuery)
	if err != nil {
		err = fmt.Errorf("error at find mi: %w", err)
		return nil, err
	}
	for _, mi := range mis {
		if mi.IsDeleted {
			continue
		}
		boardNames[mi.BoardName] = struct{}{}
	}

	boardNamesList := []string{}
	for boardName := range boardNames {
		boardNamesList = append(boardNamesList, boardName)
	}
	return boardNamesList, nil
}

func (m MiRepositories) UnWrapTyped() ([]MiRepository, error) {
	unwraped := []MiRepository{}
	for _, rep := range m {
		u, err := rep.UnWrapTyped()
		if err != nil {
			return nil, err
		}
		unwraped = append(unwraped, u...)
	}
	return unwraped, nil
}

func (m MiRepositories) UnWrap() ([]Repository, error) {
	repositories := []Repository{}
	for _, rep := range m {
		unwraped, err := rep.UnWrap()
		if err != nil {
			return nil, err
		}
		repositories = append(repositories, unwraped...)
	}
	return repositories, nil
}
