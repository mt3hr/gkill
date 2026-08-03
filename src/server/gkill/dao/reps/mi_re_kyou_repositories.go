package reps

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/threads"
)

// MiReKyouRepositories はMiReKyouリポジトリの集約です。
// ターゲット解決のためにGkillRepositoriesを参照する必要があるため、
// ReKyouRepositoriesと同じく構造体になっています。
type MiReKyouRepositories struct {
	MiReKyouRepositories []MiReKyouRepository
	GkillRepositories    *GkillRepositories
}

// cloneRepositoriesWithoutMiReKyou はMiReKyou自身を除いたGkillRepositoriesを作ります。
// MiReKyouのターゲット検索で自己参照による無限再帰が起きないようにするためのものです。
//
// ここへReKyouRepsを足してはいけません。
// ReKyou側（collectNonReKyouRepositories）はMiReKyouを含んでいるので、
// 両方向に含めるとReKyou→MiReKyou→ReKyou→…で無限再帰します。この非対称は意図的です。
func cloneRepositoriesWithoutMiReKyou(original *GkillRepositories, withoutMiReKyouReps Repositories) *GkillRepositories {
	cloned := *original
	cloned.Reps = append(Repositories(nil), withoutMiReKyouReps...)
	cloned.MiReKyouReps = MiReKyouRepositories{
		MiReKyouRepositories: nil,
		GkillRepositories:    &cloned,
	}
	return &cloned
}

func (m *MiReKyouRepositories) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	matchKyous := map[string][]Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan map[string][]Kyou, len(m.MiReKyouRepositories))
	errch := make(chan error, len(m.MiReKyouRepositories))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range m.MiReKyouRepositories {
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

	// Kyou集約
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
					if !query.OnlyLatestData {
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

func (m *MiReKyouRepositories) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	var matchKyou *Kyou
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Kyou, len(m.MiReKyouRepositories))
	errch := make(chan error, len(m.MiReKyouRepositories))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range m.MiReKyouRepositories {
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
				if matchKyouInRep.UpdateTime.After(matchKyou.UpdateTime) {
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

func (m *MiReKyouRepositories) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	kyouHistories := map[string]Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Kyou, len(m.MiReKyouRepositories))
	errch := make(chan error, len(m.MiReKyouRepositories))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range m.MiReKyouRepositories {
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

	slices.SortFunc(kyouHistoriesList, func(a, b Kyou) int {
		return b.UpdateTime.Compare(a.UpdateTime)
	})

	return kyouHistoriesList, nil
}

func (m *MiReKyouRepositories) GetPath(ctx context.Context, id string) (string, error) {
	// 並列処理
	matchPaths := []string{}

	ids := []string{id}
	for _, rep := range m.MiReKyouRepositories {
		query := &find.FindQuery{
			IDs:    ids,
			UseIDs: true,
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

func (m *MiReKyouRepositories) UpdateCache(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(m.MiReKyouRepositories))
	defer close(errch)

	for _, rep := range m.MiReKyouRepositories {
		rep := rep
		if e := threads.Go(ctx, wg, func() {
			if e := rep.UpdateCache(ctx); e != nil {
				errch <- e
			}
		}); e != nil {
			errch <- e
		}
	}
	wg.Wait()

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

func (m *MiReKyouRepositories) LastUpdateCacheChanged() bool {
	for _, rep := range m.MiReKyouRepositories {
		if rep.LastUpdateCacheChanged() {
			return true
		}
	}
	return false
}

func (m *MiReKyouRepositories) GetRepName(ctx context.Context) (string, error) {
	return "MiReKyou", nil
}

func (m *MiReKyouRepositories) Close(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(m.MiReKyouRepositories))
	defer close(errch)

	// 並列処理
	for _, rep := range m.MiReKyouRepositories {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			err := rep.Close(ctx)
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

func (m *MiReKyouRepositories) FindMiReKyou(ctx context.Context, query *find.FindQuery) ([]MiReKyou, error) {
	matchMiReKyous := map[string]MiReKyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []MiReKyou, len(m.MiReKyouRepositories))
	errch := make(chan error, len(m.MiReKyouRepositories))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range m.MiReKyouRepositories {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchMiReKyousInRep, err := rep.FindMiReKyou(ctx, query)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchMiReKyousInRep
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
			err = fmt.Errorf("error at find mirekyou: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// MiReKyou集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchMiReKyousInRep := <-ch:
			if matchMiReKyousInRep == nil {
				continue loop
			}
			for _, mirekyou := range matchMiReKyousInRep {
				key := mirekyou.ID
				if !query.OnlyLatestData {
					key += fmt.Sprintf("%d", mirekyou.UpdateTime.Unix())
				}
				if existMiReKyou, exist := matchMiReKyous[key]; exist {
					if mirekyou.UpdateTime.After(existMiReKyou.UpdateTime) {
						matchMiReKyous[key] = mirekyou
					}
				} else {
					matchMiReKyous[key] = mirekyou
				}
			}
		default:
			break loop
		}
	}

	matchMiReKyousList := []MiReKyou{}
	for _, mirekyou := range matchMiReKyous {
		matchMiReKyousList = append(matchMiReKyousList, mirekyou)
	}
	return matchMiReKyousList, nil
}

func (m *MiReKyouRepositories) GetMiReKyou(ctx context.Context, id string, updateTime *time.Time) (*MiReKyou, error) {
	var matchMiReKyou *MiReKyou
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *MiReKyou, len(m.MiReKyouRepositories))
	errch := make(chan error, len(m.MiReKyouRepositories))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range m.MiReKyouRepositories {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchMiReKyouInRep, err := rep.GetMiReKyou(ctx, id, updateTime)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchMiReKyouInRep
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
			err = fmt.Errorf("error at get mirekyou: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// MiReKyou集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchMiReKyouInRep := <-ch:
			if matchMiReKyouInRep == nil {
				continue loop
			}
			if matchMiReKyou != nil {
				if matchMiReKyouInRep.UpdateTime.After(matchMiReKyou.UpdateTime) {
					matchMiReKyou = matchMiReKyouInRep
				}
			} else {
				matchMiReKyou = matchMiReKyouInRep
			}
		default:
			break loop
		}
	}

	return matchMiReKyou, nil
}

func (m *MiReKyouRepositories) GetMiReKyouHistories(ctx context.Context, id string) ([]MiReKyou, error) {
	return m.GetMiReKyouHistoriesByRepName(ctx, id, nil)
}

func (m *MiReKyouRepositories) GetMiReKyouHistoriesByRepName(ctx context.Context, id string, repName *string) ([]MiReKyou, error) {
	repImpls, err := m.UnWrapTyped()
	if err != nil {
		return nil, err
	}

	mirekyouHistories := map[string]MiReKyou{}
	existErr := false
	wg := &sync.WaitGroup{}
	ch := make(chan []MiReKyou, len(repImpls))
	errch := make(chan error, len(repImpls))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range repImpls {
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

			matchMiReKyousInRep, err := rep.GetMiReKyouHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchMiReKyousInRep
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
			err = fmt.Errorf("error at find get mirekyou histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// MiReKyou集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchMiReKyousInRep := <-ch:
			if matchMiReKyousInRep == nil {
				continue loop
			}
			for _, mirekyou := range matchMiReKyousInRep {
				if existMiReKyou, exist := mirekyouHistories[mirekyou.ID+mirekyou.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if mirekyou.UpdateTime.After(existMiReKyou.UpdateTime) {
						mirekyouHistories[mirekyou.ID+mirekyou.UpdateTime.Format(sqlite3impl.TimeLayout)] = mirekyou
					}
				} else {
					mirekyouHistories[mirekyou.ID+mirekyou.UpdateTime.Format(sqlite3impl.TimeLayout)] = mirekyou
				}
			}
		default:
			break loop
		}
	}

	mirekyouHistoriesList := []MiReKyou{}
	for _, mirekyou := range mirekyouHistories {
		mirekyouHistoriesList = append(mirekyouHistoriesList, mirekyou)
	}

	slices.SortFunc(mirekyouHistoriesList, func(a, b MiReKyou) int {
		return b.UpdateTime.Compare(a.UpdateTime)
	})

	return mirekyouHistoriesList, nil
}

func (m *MiReKyouRepositories) AddMiReKyouInfo(ctx context.Context, mirekyou MiReKyou) error {
	err := fmt.Errorf("not implements MiReKyouReps.AddMiReKyouInfo")
	return err
}

func (m *MiReKyouRepositories) GetMiReKyousAllLatest(ctx context.Context) ([]MiReKyou, error) {
	matchMiReKyous := map[string]MiReKyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []MiReKyou, len(m.MiReKyouRepositories))
	errch := make(chan error, len(m.MiReKyouRepositories))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range m.MiReKyouRepositories {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchMiReKyousInRep, err := rep.GetMiReKyousAllLatest(ctx)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchMiReKyousInRep
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
			err = fmt.Errorf("error at find mirekyou: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// MiReKyou集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchMiReKyousInRep := <-ch:
			if matchMiReKyousInRep == nil {
				continue loop
			}
			for _, mirekyou := range matchMiReKyousInRep {
				if existMiReKyou, exist := matchMiReKyous[mirekyou.ID]; exist {
					if mirekyou.UpdateTime.After(existMiReKyou.UpdateTime) {
						matchMiReKyous[mirekyou.ID] = mirekyou
					}
				} else {
					matchMiReKyous[mirekyou.ID] = mirekyou
				}
			}
		default:
			break loop
		}
	}

	matchMiReKyousList := []MiReKyou{}
	for _, mirekyou := range matchMiReKyous {
		matchMiReKyousList = append(matchMiReKyousList, mirekyou)
	}

	slices.SortFunc(matchMiReKyousList, func(a, b MiReKyou) int {
		return b.CreateTime.Compare(a.CreateTime)
	})
	return matchMiReKyousList, nil
}

func (m *MiReKyouRepositories) GetBoardNames(ctx context.Context) ([]string, error) {
	boardNames := map[string]any{}

	findMiReKyouQuery := &find.FindQuery{}
	findMiReKyouQuery.OnlyLatestData = true
	findMiReKyouQuery.IncludeCreateMi = true
	findMiReKyouQuery.IncludeCheckMi = true
	findMiReKyouQuery.IncludeLimitMi = true
	findMiReKyouQuery.IncludeStartMi = true
	findMiReKyouQuery.IncludeEndMi = true
	mirekyous, err := m.FindMiReKyou(ctx, findMiReKyouQuery)
	if err != nil {
		err = fmt.Errorf("error at find mirekyou: %w", err)
		return nil, err
	}
	for _, mirekyou := range mirekyous {
		if mirekyou.IsDeleted {
			continue
		}
		boardNames[mirekyou.BoardName] = struct{}{}
	}

	boardNamesList := []string{}
	for boardName := range boardNames {
		boardNamesList = append(boardNamesList, boardName)
	}
	return boardNamesList, nil
}

// GetRepositoriesWithoutMiReKyouRep はMiReKyou自身とReKyouを除いたリポジトリ群を返します。
// MiReKyouのターゲット解決で相互再帰が起きないようにするためのものです。
func (m *MiReKyouRepositories) GetRepositoriesWithoutMiReKyouRep(ctx context.Context) (*GkillRepositories, error) {
	if m.GkillRepositories == nil {
		// TX中の一時リポジトリなどリポジトリ群を持たない場合はnilを返す。
		// 呼び出し側(newMiReKyouTargetFilter)はターゲット解決を行わない。
		return nil, nil
	}

	withoutMiReKyouReps := Repositories{}
	for _, rep := range m.GkillRepositories.KmemoReps {
		withoutMiReKyouReps = append(withoutMiReKyouReps, rep)
	}
	for _, rep := range m.GkillRepositories.KCReps {
		withoutMiReKyouReps = append(withoutMiReKyouReps, rep)
	}
	for _, rep := range m.GkillRepositories.URLogReps {
		withoutMiReKyouReps = append(withoutMiReKyouReps, rep)
	}
	for _, rep := range m.GkillRepositories.NlogReps {
		withoutMiReKyouReps = append(withoutMiReKyouReps, rep)
	}
	for _, rep := range m.GkillRepositories.TimeIsReps {
		withoutMiReKyouReps = append(withoutMiReKyouReps, rep)
	}
	for _, rep := range m.GkillRepositories.MiReps {
		withoutMiReKyouReps = append(withoutMiReKyouReps, rep)
	}
	for _, rep := range m.GkillRepositories.LantanaReps {
		withoutMiReKyouReps = append(withoutMiReKyouReps, rep)
	}
	for _, rep := range m.GkillRepositories.IDFKyouReps {
		withoutMiReKyouReps = append(withoutMiReKyouReps, rep)
	}
	for _, rep := range m.GkillRepositories.GitCommitLogReps {
		withoutMiReKyouReps = append(withoutMiReKyouReps, rep)
	}

	return cloneRepositoriesWithoutMiReKyou(m.GkillRepositories, withoutMiReKyouReps), nil
}

func (m *MiReKyouRepositories) UnWrapTyped() ([]MiReKyouRepository, error) {
	unwraped := []MiReKyouRepository{}
	for _, rep := range m.MiReKyouRepositories {
		u, err := rep.UnWrapTyped()
		if err != nil {
			return nil, err
		}
		unwraped = append(unwraped, u...)
	}
	return unwraped, nil
}

func (m *MiReKyouRepositories) UnWrap() ([]Repository, error) {
	repositories := []Repository{}
	for _, rep := range m.MiReKyouRepositories {
		unwraped, err := rep.UnWrap()
		if err != nil {
			return nil, err
		}
		repositories = append(repositories, unwraped...)
	}
	return repositories, nil
}

func (m *MiReKyouRepositories) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	allAddrs := []gkill_cache.LatestDataRepositoryAddress{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []gkill_cache.LatestDataRepositoryAddress, len(m.MiReKyouRepositories))
	errch := make(chan error, len(m.MiReKyouRepositories))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range m.MiReKyouRepositories {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			addrs, err := rep.GetLatestDataRepositoryAddress(ctx, updateCache)
			if err != nil {
				errch <- err
				return
			}
			ch <- addrs
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
			err = fmt.Errorf("error at get latest data repository address: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// 集約
loop:
	for {
		select {
		case addrs := <-ch:
			if addrs == nil {
				continue loop
			}
			allAddrs = append(allAddrs, addrs...)
		default:
			break loop
		}
	}
	return allAddrs, nil
}
