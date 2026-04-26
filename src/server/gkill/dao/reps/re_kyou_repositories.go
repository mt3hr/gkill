package reps

import (
	"context"
	"fmt"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"slices"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/threads"
)

type ReKyouRepositories struct {
	ReKyouRepositories []ReKyouRepository
	GkillRepositories  *GkillRepositories
}

func cloneRepositoriesWithoutReKyou(original *GkillRepositories, withoutRekyouReps Repositories) *GkillRepositories {
	cloned := *original
	cloned.Reps = append(Repositories(nil), withoutRekyouReps...)
	cloned.ReKyouReps = ReKyouRepositories{
		ReKyouRepositories: nil,
		GkillRepositories:  &cloned,
	}
	return &cloned
}

func (r *ReKyouRepositories) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	matchKyous := map[string][]Kyou{}

	// 未削除ReKyouを抽出
	notDeletedAllReKyous := []ReKyou{}
	allReKyous, err := r.GetReKyousAllLatest(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rekyous all latest :%w", err)
		return nil, err
	}
	for _, rekyou := range allReKyous {
		if !rekyou.IsDeleted {
			notDeletedAllReKyous = append(notDeletedAllReKyous, rekyou)
		}
	}

	repsWithoutRekyou, err := r.GetRepositoriesWithoutReKyouRep(ctx)
	if err != nil {
		err = fmt.Errorf("error at get repositories: %w", err)
		return nil, err
	}

	latestDataRepositoryAddresses, err := repsWithoutRekyou.LatestDataRepositoryAddressDAO.GetAllLatestDataRepositoryAddresses(ctx)
	if err != nil {
		err = fmt.Errorf("error at get all latest data repository addresses: %w", err)
		return nil, err
	}

	// ワードフィルタ: UseWordsが有効な場合、Targetに対してワード検索を実行しマッチしたIDを収集する
	wordMatchTargetIDs := map[string]bool{}
	useWordFilter := query.UseWords && (len(query.Words) > 0 || len(query.NotWords) > 0)
	if useWordFilter {
		wordMatchKyousMap, err := repsWithoutRekyou.Reps.FindKyous(ctx, query)
		if err != nil {
			err = fmt.Errorf("error at find kyous for word filter: %w", err)
			return nil, err
		}
		for _, kyous := range wordMatchKyousMap {
			for _, kyou := range kyous {
				wordMatchTargetIDs[kyou.ID] = true
			}
		}
	}

	for _, rekyou := range notDeletedAllReKyous {
		if rekyou.IsDeleted {
			continue
		}
		if _, ok := latestDataRepositoryAddresses[rekyou.TargetID]; !ok {
			continue
		}
		// ワードフィルタが有効な場合、TargetIDがマッチしなければスキップ
		if useWordFilter {
			if !wordMatchTargetIDs[rekyou.TargetID] {
				continue
			}
		}

		// 存在すれば検索ヒットとする
		kyou := Kyou{}
		kyou.IsDeleted = rekyou.IsDeleted
		kyou.ID = rekyou.ID
		kyou.RepName = rekyou.RepName
		kyou.RelatedTime = rekyou.RelatedTime
		kyou.DataType = rekyou.DataType
		kyou.CreateTime = rekyou.CreateTime
		kyou.CreateApp = rekyou.CreateApp
		kyou.CreateDevice = rekyou.CreateDevice
		kyou.CreateUser = rekyou.CreateUser
		kyou.UpdateTime = rekyou.UpdateTime
		kyou.UpdateApp = rekyou.UpdateApp
		kyou.UpdateUser = rekyou.UpdateUser
		kyou.UpdateDevice = rekyou.UpdateDevice
		if _, exist := matchKyous[kyou.ID]; !exist {
			matchKyous[kyou.ID] = []Kyou{}
		}

		key := kyou.ID
		if !query.OnlyLatestData {
			key += fmt.Sprintf("%d", kyou.UpdateTime.Unix())
		}
		matchKyous[key] = append(matchKyous[key], kyou)
	}
	return matchKyous, nil
}

func (r *ReKyouRepositories) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	var matchKyou *Kyou
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Kyou, len(r.ReKyouRepositories))
	errch := make(chan error, len(r.ReKyouRepositories))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range r.ReKyouRepositories {
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

func (r *ReKyouRepositories) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	kyouHistories := map[string]Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Kyou, len(r.ReKyouRepositories))
	errch := make(chan error, len(r.ReKyouRepositories))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range r.ReKyouRepositories {
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

func (r *ReKyouRepositories) GetPath(ctx context.Context, id string) (string, error) {
	// 並列処理
	matchPaths := []string{}

	ids := []string{id}
	for _, rep := range r.ReKyouRepositories {
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

func (r *ReKyouRepositories) UpdateCache(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(r.ReKyouRepositories))
	defer close(errch)

	for _, rep := range r.ReKyouRepositories {
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

func (r *ReKyouRepositories) LastUpdateCacheChanged() bool {
	for _, rep := range r.ReKyouRepositories {
		if rep.LastUpdateCacheChanged() {
			return true
		}
	}
	return false
}

func (r *ReKyouRepositories) GetRepName(ctx context.Context) (string, error) {
	return "ReKyou", nil
}

func (r *ReKyouRepositories) Close(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(r.ReKyouRepositories))
	defer close(errch)

	// 並列処理
	for _, rep := range r.ReKyouRepositories {
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

func (r *ReKyouRepositories) FindReKyou(ctx context.Context, query *find.FindQuery) ([]ReKyou, error) {
	matchReKyous := []ReKyou{}

	// 未削除ReKyouを抽出
	notDeletedAllReKyous := []ReKyou{}
	allReKyous, err := r.GetReKyousAllLatest(ctx)
	if err != nil {
		err = fmt.Errorf("error at get rekyous all latest :%w", err)
		return nil, err
	}
	for _, rekyou := range allReKyous {
		if !rekyou.IsDeleted {
			notDeletedAllReKyous = append(notDeletedAllReKyous, rekyou)
		}
	}

	repsWithoutRekyou, err := r.GetRepositoriesWithoutReKyouRep(ctx)
	if err != nil {
		err = fmt.Errorf("error at get without rekyou: %w", err)
		return nil, err
	}

	latestDataRepositoryAddresses, err := repsWithoutRekyou.LatestDataRepositoryAddressDAO.GetAllLatestDataRepositoryAddresses(ctx)
	if err != nil {
		err = fmt.Errorf("error at get all latest data repository addresses: %w", err)
		return nil, err
	}

	for _, rekyou := range notDeletedAllReKyous {
		if rekyou.IsDeleted {
			continue
		}
		if _, ok := latestDataRepositoryAddresses[rekyou.TargetID]; !ok {
			continue
		}
		matchReKyous = append(matchReKyous, rekyou)
	}
	return matchReKyous, nil
}

func (r *ReKyouRepositories) GetReKyou(ctx context.Context, id string, updateTime *time.Time) (*ReKyou, error) {
	var matchReKyou *ReKyou
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *ReKyou, len(r.ReKyouRepositories))
	errch := make(chan error, len(r.ReKyouRepositories))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range r.ReKyouRepositories {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchReKyouInRep, err := rep.GetReKyou(ctx, id, updateTime)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchReKyouInRep
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
			err = fmt.Errorf("error at get rekyou: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// ReKyou集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchReKyouInRep := <-ch:
			if matchReKyouInRep == nil {
				continue loop
			}
			if matchReKyou != nil {
				if matchReKyouInRep.UpdateTime.Before(matchReKyou.UpdateTime) {
					matchReKyou = matchReKyouInRep
				}
			} else {
				matchReKyou = matchReKyouInRep
			}
		default:
			break loop
		}
	}

	return matchReKyou, nil
}

func (r *ReKyouRepositories) GetReKyouHistories(ctx context.Context, id string) ([]ReKyou, error) {
	kyouHistories := map[string]ReKyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []ReKyou, len(r.ReKyouRepositories))
	errch := make(chan error, len(r.ReKyouRepositories))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range r.ReKyouRepositories {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchReKyousInRep, err := rep.GetReKyouHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchReKyousInRep
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
			err = fmt.Errorf("error at find get rekyou histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// ReKyou集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchReKyousInRep := <-ch:
			if matchReKyousInRep == nil {
				continue loop
			}
			for _, kyou := range matchReKyousInRep {
				if existReKyou, exist := kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if kyou.UpdateTime.After(existReKyou.UpdateTime) {
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

	kyouHistoriesList := []ReKyou{}
	for _, kyou := range kyouHistories {

		kyouHistoriesList = append(kyouHistoriesList, kyou)
	}

	slices.SortFunc(kyouHistoriesList, func(a, b ReKyou) int {
		return b.UpdateTime.Compare(a.UpdateTime)
	})

	return kyouHistoriesList, nil
}

func (r *ReKyouRepositories) GetReKyouHistoriesByRepName(ctx context.Context, id string, repName *string) ([]ReKyou, error) {
	repImpls, err := r.UnWrapTyped()
	if err != nil {
		return nil, err
	}

	kyouHistories := map[string]ReKyou{}
	existErr := false
	wg := &sync.WaitGroup{}
	ch := make(chan []ReKyou, len(repImpls))
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

			matchReKyousInRep, err := rep.GetReKyouHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchReKyousInRep
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
			err = fmt.Errorf("error at find get rekyou histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// ReKyou集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchReKyousInRep := <-ch:
			if matchReKyousInRep == nil {
				continue loop
			}
			for _, kyou := range matchReKyousInRep {
				if existReKyou, exist := kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if kyou.UpdateTime.After(existReKyou.UpdateTime) {
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

	kyouHistoriesList := []ReKyou{}
	for _, kyou := range kyouHistories {

		kyouHistoriesList = append(kyouHistoriesList, kyou)
	}

	slices.SortFunc(kyouHistoriesList, func(a, b ReKyou) int {
		return b.UpdateTime.Compare(a.UpdateTime)
	})

	return kyouHistoriesList, nil
}

func (r *ReKyouRepositories) AddReKyouInfo(ctx context.Context, rekyou ReKyou) error {
	err := fmt.Errorf("not implements ReKyouReps.AddReKyouInfo")
	return err
}

func (r *ReKyouRepositories) GetReKyousAllLatest(ctx context.Context) ([]ReKyou, error) {
	matchReKyous := map[string]ReKyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []ReKyou, len(r.ReKyouRepositories))
	errch := make(chan error, len(r.ReKyouRepositories))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range r.ReKyouRepositories {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchReKyousInRep, err := rep.GetReKyousAllLatest(ctx)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchReKyousInRep
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
			err = fmt.Errorf("error at find rekyou: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// ReKyou集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchReKyousInRep := <-ch:
			if matchReKyousInRep == nil {
				continue loop
			}
			for _, kyou := range matchReKyousInRep {
				if existReKyou, exist := matchReKyous[kyou.ID]; exist {
					if kyou.UpdateTime.After(existReKyou.UpdateTime) {
						matchReKyous[kyou.ID] = kyou
					}
				} else {
					matchReKyous[kyou.ID] = kyou
				}
			}
		default:
			break loop
		}
	}

	matchReKyousList := []ReKyou{}
	for _, kyou := range matchReKyous {

		matchReKyousList = append(matchReKyousList, kyou)
	}

	slices.SortFunc(matchReKyousList, func(a, b ReKyou) int {
		return b.RelatedTime.Compare(a.RelatedTime)
	})
	return matchReKyousList, nil
}

func (r *ReKyouRepositories) GetRepositoriesWithoutReKyouRep(ctx context.Context) (*GkillRepositories, error) {
	withoutRekyouReps := Repositories{}
	for _, rep := range r.GkillRepositories.KmemoReps {
		withoutRekyouReps = append(withoutRekyouReps, rep)
	}
	for _, rep := range r.GkillRepositories.KCReps {
		withoutRekyouReps = append(withoutRekyouReps, rep)
	}
	for _, rep := range r.GkillRepositories.URLogReps {
		withoutRekyouReps = append(withoutRekyouReps, rep)
	}
	for _, rep := range r.GkillRepositories.NlogReps {
		withoutRekyouReps = append(withoutRekyouReps, rep)
	}
	for _, rep := range r.GkillRepositories.TimeIsReps {
		withoutRekyouReps = append(withoutRekyouReps, rep)
	}
	for _, rep := range r.GkillRepositories.MiReps {
		withoutRekyouReps = append(withoutRekyouReps, rep)
	}
	for _, rep := range r.GkillRepositories.LantanaReps {
		withoutRekyouReps = append(withoutRekyouReps, rep)
	}
	for _, rep := range r.GkillRepositories.IDFKyouReps {
		withoutRekyouReps = append(withoutRekyouReps, rep)
	}
	for _, rep := range r.GkillRepositories.GitCommitLogReps {
		withoutRekyouReps = append(withoutRekyouReps, rep)
	}

	return cloneRepositoriesWithoutReKyou(r.GkillRepositories, withoutRekyouReps), nil
}

func (r *ReKyouRepositories) UnWrapTyped() ([]ReKyouRepository, error) {
	unwraped := []ReKyouRepository{}
	for _, rep := range r.ReKyouRepositories {
		u, err := rep.UnWrapTyped()
		if err != nil {
			return nil, err
		}
		unwraped = append(unwraped, u...)
	}
	return unwraped, nil
}

func (r *ReKyouRepositories) UnWrap() ([]Repository, error) {
	repositories := []Repository{}
	for _, rep := range r.ReKyouRepositories {
		unwraped, err := rep.UnWrap()
		if err != nil {
			return nil, err
		}
		repositories = append(repositories, unwraped...)
	}
	return repositories, nil
}

func (r *ReKyouRepositories) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	allAddrs := []gkill_cache.LatestDataRepositoryAddress{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []gkill_cache.LatestDataRepositoryAddress, len(r.ReKyouRepositories))
	errch := make(chan error, len(r.ReKyouRepositories))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range r.ReKyouRepositories {
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
