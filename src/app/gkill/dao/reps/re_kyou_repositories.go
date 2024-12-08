package reps

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
)

type ReKyouRepositories struct {
	ReKyouRepositories []ReKyouRepository
	repositories       *GkillRepositories
}

func (r *ReKyouRepositories) FindKyous(ctx context.Context, query *find.FindQuery) ([]*Kyou, error) {
	matchKyous := []*Kyou{}

	// 未削除ReKyouを抽出
	notDeletedAllReKyous := []*ReKyou{}
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

	// ReKyou対象が検索ヒットすれば返す
	// 検索用クエリJSONを作成
	ids := []string{}
	for _, rekyou := range allReKyous {
		ids = append(ids, rekyou.TargetID)
	}

	falseValue := false
	trueValue := true
	findQuery := &find.FindQuery{
		IsDeleted: &falseValue,
		UseIDs:    &trueValue,
		IDs:       &ids,
	}

	reps, err := r.GetRepositories(ctx)
	if err != nil {
		err = fmt.Errorf("error at get repositories: %w", err)
		return nil, err
	}

	for _, rekyou := range notDeletedAllReKyous {
		kyous, err := reps.Reps.FindKyous(ctx, findQuery)
		if err != nil {
			err = fmt.Errorf("error at find kyous: %w", err)
			return nil, err
		}
		// 存在すれば検索ヒットとする
		if len(kyous) != 0 {
			kyou := &Kyou{}
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
			matchKyous = append(matchKyous, kyou)
		}
	}
	return matchKyous, nil
}

func (r *ReKyouRepositories) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	matchKyou := &Kyou{}
	matchKyou = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Kyou, len(r.ReKyouRepositories))
	errch := make(chan error, len(r.ReKyouRepositories))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range r.ReKyouRepositories {
		wg.Add(1)

		go func(rep ReKyouRepository) {
			defer wg.Done()
			matchKyouInRep, err := rep.GetKyou(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKyouInRep
		}(rep)
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

func (r *ReKyouRepositories) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	kyouHistories := map[string]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kyou, len(r.ReKyouRepositories))
	errch := make(chan error, len(r.ReKyouRepositories))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range r.ReKyouRepositories {
		wg.Add(1)

		go func(rep ReKyouRepository) {
			defer wg.Done()
			matchKyousInRep, err := rep.GetKyouHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKyousInRep
		}(rep)
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
					if kyou.UpdateTime.Before(existKyou.UpdateTime) {
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

func (r *ReKyouRepositories) GetPath(ctx context.Context, id string) (string, error) {
	err := fmt.Errorf("not implements ReKyouReps.GetPath")
	return "", err
}

func (r *ReKyouRepositories) UpdateCache(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(r.ReKyouRepositories))
	defer close(errch)

	// 並列処理
	for _, rep := range r.ReKyouRepositories {
		wg.Add(1)

		go func(rep ReKyouRepository) {
			defer wg.Done()
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

func (r *ReKyouRepositories) GetRepName(ctx context.Context) (string, error) {
	return "ReKyouReps", nil
}

func (r *ReKyouRepositories) Close(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(r.ReKyouRepositories))
	defer close(errch)

	// 並列処理
	for _, rep := range r.ReKyouRepositories {
		wg.Add(1)

		go func(rep ReKyouRepository) {
			defer wg.Done()
			err = rep.Close(ctx)
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

func (r *ReKyouRepositories) FindReKyou(ctx context.Context, query *find.FindQuery) ([]*ReKyou, error) {
	matchReKyous := []*ReKyou{}

	// 未削除ReKyouを抽出
	notDeletedAllReKyous := []*ReKyou{}
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

	// ReKyou対象が検索ヒットすれば返す
	// 検索用クエリJSONを作成
	ids := []string{}
	for _, rekyou := range allReKyous {
		ids = append(ids, rekyou.TargetID)
	}

	falseValue := false
	trueValue := true
	findQuery := &find.FindQuery{
		IsDeleted: &falseValue,
		UseIDs:    &trueValue,
		IDs:       &ids,
	}

	reps, err := r.GetRepositories(ctx)
	if err != nil {
		err = fmt.Errorf("error at get repositories: %w", err)
		return nil, err
	}

	for _, rekyou := range notDeletedAllReKyous {
		kyous, err := reps.Reps.FindKyous(ctx, findQuery)
		if err != nil {
			err = fmt.Errorf("error at find kyous: %w", err)
			return nil, err
		}
		// 存在すれば検索ヒットとする
		if len(kyous) != 0 {
			matchReKyous = append(matchReKyous, rekyou)
		}
	}
	return matchReKyous, nil
}

func (r *ReKyouRepositories) GetReKyou(ctx context.Context, id string) (*ReKyou, error) {
	matchReKyou := &ReKyou{}
	matchReKyou = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *ReKyou, len(r.ReKyouRepositories))
	errch := make(chan error, len(r.ReKyouRepositories))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range r.ReKyouRepositories {
		wg.Add(1)

		go func(rep ReKyouRepository) {
			defer wg.Done()
			matchReKyouInRep, err := rep.GetReKyou(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchReKyouInRep
		}(rep)
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

func (r *ReKyouRepositories) GetReKyouHistories(ctx context.Context, id string) ([]*ReKyou, error) {
	kyouHistories := map[string]*ReKyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*ReKyou, len(r.ReKyouRepositories))
	errch := make(chan error, len(r.ReKyouRepositories))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range r.ReKyouRepositories {
		wg.Add(1)

		go func(rep ReKyouRepository) {
			defer wg.Done()
			matchReKyousInRep, err := rep.GetReKyouHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchReKyousInRep
		}(rep)
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
					if kyou.UpdateTime.Before(existReKyou.UpdateTime) {
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

	kyouHistoriesList := []*ReKyou{}
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

func (r *ReKyouRepositories) AddReKyouInfo(ctx context.Context, rekyou *ReKyou) error {
	err := fmt.Errorf("not implements ReKyouReps.AddReKyouInfo")
	return err
}

func (r *ReKyouRepositories) GetReKyousAllLatest(ctx context.Context) ([]*ReKyou, error) {
	matchReKyous := map[string]*ReKyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*ReKyou, len(r.ReKyouRepositories))
	errch := make(chan error, len(r.ReKyouRepositories))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range r.ReKyouRepositories {
		wg.Add(1)

		go func(rep ReKyouRepository) {
			defer wg.Done()
			matchReKyousInRep, err := rep.GetReKyousAllLatest(ctx)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchReKyousInRep
		}(rep)
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
					if kyou.UpdateTime.Before(existReKyou.UpdateTime) {
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

	matchReKyousList := []*ReKyou{}
	for _, kyou := range matchReKyous {
		if kyou == nil {
			continue
		}
		matchReKyousList = append(matchReKyousList, kyou)
	}

	sort.Slice(matchReKyousList, func(i, j int) bool {
		return matchReKyousList[i].RelatedTime.After(matchReKyousList[j].RelatedTime)
	})
	return matchReKyousList, nil
}

func (r *ReKyouRepositories) GetRepositories(ctx context.Context) (*GkillRepositories, error) {
	return r.repositories, nil
}
