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

type LantanaRepositories []LantanaRepository

func (l LantanaRepositories) FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error) {
	// ˅
	matchKyous := map[string]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kyou, len(l))
	errch := make(chan error, len(l))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range l {
		wg.Add(1)
		rep := rep
		go func(rep LantanaRepository) {
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

func (l LantanaRepositories) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	// ˅
	matchKyou := &Kyou{}
	matchKyou = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Kyou, len(l))
	errch := make(chan error, len(l))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range l {
		wg.Add(1)
		rep := rep
		go func(rep LantanaRepository) {
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

func (l LantanaRepositories) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	// ˅
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
		wg.Add(1)
		rep := rep
		go func(rep LantanaRepository) {
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

func (l LantanaRepositories) GetPath(ctx context.Context, id string) (string, error) {
	// ˅
	err := fmt.Errorf("not implements LantanaReps.GetPath")
	return "", err
	// ˄
}

func (l LantanaRepositories) UpdateCache(ctx context.Context) error {
	// ˅
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(l))
	defer close(errch)

	// 並列処理
	for _, rep := range l {
		wg.Add(1)
		rep := rep
		go func(rep LantanaRepository) {
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

func (l LantanaRepositories) GetRepName(ctx context.Context) (string, error) {
	// ˅
	return "LantanaReps", nil
	// ˄
}

func (l LantanaRepositories) Close(ctx context.Context) error {
	// ˅
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(l))
	defer close(errch)

	// 並列処理
	for _, rep := range l {
		wg.Add(1)
		rep := rep
		go func(rep LantanaRepository) {
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

func (l LantanaRepositories) FindLantana(ctx context.Context, queryJSON string) ([]*Lantana, error) {
	// ˅
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
		wg.Add(1)
		rep := rep
		go func(rep LantanaRepository) {
			defer wg.Done()
			matchLantanasInRep, err := rep.FindLantana(ctx, queryJSON)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchLantanasInRep
		}(rep)
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
				if existLantana, exist := matchLantanas[kyou.ID]; exist {
					if kyou.UpdateTime.Before(existLantana.UpdateTime) {
						matchLantanas[kyou.ID] = kyou
					}
				} else {
					matchLantanas[kyou.ID] = kyou
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

	sort.Slice(matchLantanasList, func(i, j int) bool {
		return matchLantanasList[i].RelatedTime.After(matchLantanasList[j].RelatedTime)
	})
	return matchLantanasList, nil
	// ˄
}

func (l LantanaRepositories) GetLantana(ctx context.Context, id string) (*Lantana, error) {
	// ˅
	matchLantana := &Lantana{}
	matchLantana = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Lantana, len(l))
	errch := make(chan error, len(l))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range l {
		wg.Add(1)
		rep := rep
		go func(rep LantanaRepository) {
			defer wg.Done()
			matchLantanaInRep, err := rep.GetLantana(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchLantanaInRep
		}(rep)
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
	// ˄
}

func (l LantanaRepositories) GetLantanaHistories(ctx context.Context, id string) ([]*Lantana, error) {
	// ˅
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
		wg.Add(1)
		rep := rep
		go func(rep LantanaRepository) {
			defer wg.Done()
			matchLantanasInRep, err := rep.GetLantanaHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchLantanasInRep
		}(rep)
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
					if kyou.UpdateTime.Before(existLantana.UpdateTime) {
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
	// ˄
}

func (l LantanaRepositories) AddLantanaInfo(ctx context.Context, lantana *Lantana) error {
	// ˅
	err := fmt.Errorf("not implements LantanaReps.AddLantanaInfo")
	return err
	// ˄
}

// ˅

// ˄
