// ˅
package reps

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/mt3hr/gkill/src/app/gkill/dbo/sqlite3impl"
)

// ˄

type MiRepositories []MiRepository

func (m MiRepositories) FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error) {
	// ˅
	matchKyous := map[string]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kyou, len(m))
	errch := make(chan error, len(m))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range m {
		wg.Add(1)
		rep := rep
		go func(rep MiRepository) {
			defer wg.Done()
			matchKyousInRep, err := rep.FindKyous(ctx, queryJSON)
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
			for _, kyou := range matchKyousInRep {
				if existKyou, exist := matchKyous[kyou.ID]; exist {
					if kyou.UpdateTime.Before(existKyou.UpdateTime) {
						matchKyous[kyou.ID] = kyou
					}
				} else {
					matchKyous[kyou.ID] = kyou
				}
			}
		default:
			break loop
		}
	}

	matchKyousList := []*Kyou{}
	for _, kyou := range matchKyous {
		if kyou == nil {
			continue
		}
		matchKyousList = append(matchKyousList, kyou)
	}

	sort.Slice(matchKyousList, func(i, j int) bool {
		return matchKyousList[i].RelatedTime.After(matchKyousList[j].RelatedTime)
	})
	return matchKyousList, nil
	// ˄
}

func (m MiRepositories) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	// ˅
	matchKyou := &Kyou{}
	matchKyou = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Kyou, len(m))
	errch := make(chan error, len(m))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range m {
		wg.Add(1)
		rep := rep
		go func(rep MiRepository) {
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
	// ˄
}

func (m MiRepositories) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	// ˅
	kyouHistories := map[string]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kyou, len(m))
	errch := make(chan error, len(m))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range m {
		wg.Add(1)
		rep := rep
		go func(rep MiRepository) {
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
	// ˄
}

func (m MiRepositories) GetPath(ctx context.Context, id string) (string, error) {
	// ˅
	err := fmt.Errorf("not implements MiReps.GetPath")
	return "", err
	// ˄
}

func (m MiRepositories) UpdateCache(ctx context.Context) error {
	// ˅
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(m))
	defer close(errch)

	// 並列処理
	for _, rep := range m {
		wg.Add(1)
		rep := rep
		go func(rep MiRepository) {
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
	// ˄
}

func (m MiRepositories) GetRepName(ctx context.Context) (string, error) {
	// ˅
	return "MiReps", nil
	// ˄
}

func (m MiRepositories) Close(ctx context.Context) error {
	// ˅
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(m))
	defer close(errch)

	// 並列処理
	for _, rep := range m {
		wg.Add(1)
		rep := rep
		go func(rep MiRepository) {
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
	// ˄
}

func (m MiRepositories) FindMi(ctx context.Context, queryJSON string) ([]*Mi, error) {
	// ˅
	matchMis := map[string]*Mi{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Mi, len(m))
	errch := make(chan error, len(m))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range m {
		wg.Add(1)
		rep := rep
		go func(rep MiRepository) {
			defer wg.Done()
			matchMisInRep, err := rep.FindMi(ctx, queryJSON)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchMisInRep
		}(rep)
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
				if existMi, exist := matchMis[kyou.ID]; exist {
					if kyou.UpdateTime.Before(existMi.UpdateTime) {
						matchMis[kyou.ID] = kyou
					}
				} else {
					matchMis[kyou.ID] = kyou
				}
			}
		default:
			break loop
		}
	}

	matchMisList := []*Mi{}
	for _, kyou := range matchMis {
		if kyou == nil {
			continue
		}
		matchMisList = append(matchMisList, kyou)
	}

	sort.Slice(matchMisList, func(i, j int) bool {
		return matchMisList[i].CreateTime.After(matchMisList[j].CreateTime)
	})
	return matchMisList, nil
	// ˄
}

func (m MiRepositories) GetMi(ctx context.Context, id string) (*Mi, error) {
	// ˅
	matchMi := &Mi{}
	matchMi = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Mi, len(m))
	errch := make(chan error, len(m))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range m {
		wg.Add(1)
		rep := rep
		go func(rep MiRepository) {
			defer wg.Done()
			matchMiInRep, err := rep.GetMi(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchMiInRep
		}(rep)
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
	// ˄
}

func (m MiRepositories) GetMiHistories(ctx context.Context, id string) ([]*Mi, error) {
	// ˅
	kyouHistories := map[string]*Mi{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Mi, len(m))
	errch := make(chan error, len(m))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range m {
		wg.Add(1)
		rep := rep
		go func(rep MiRepository) {
			defer wg.Done()
			matchMisInRep, err := rep.GetMiHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchMisInRep
		}(rep)
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
					if kyou.UpdateTime.Before(existMi.UpdateTime) {
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

	kyouHistoriesList := []*Mi{}
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
	// ˄
}

func (m MiRepositories) AddMiInfo(ctx context.Context, mi *Mi) error {
	// ˅
	err := fmt.Errorf("not implements MiReps.AddMiInfo")
	return err
	// ˄
}

// ˅

// ˄
