package reps

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
)

type TextRepositories []TextRepository

func (t TextRepositories) FindTexts(ctx context.Context, query *find.FindQuery) ([]*Text, error) {
	matchTexts := map[string]*Text{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Text, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)

		go func(rep TextRepository) {
			defer wg.Done()
			matchTextsInRep, err := rep.FindTexts(ctx, query)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTextsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find text: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Text集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchTextsInRep := <-ch:
			if matchTextsInRep == nil {
				continue loop
			}
			for _, text := range matchTextsInRep {
				if existText, exist := matchTexts[text.ID]; exist {
					if text.UpdateTime.After(existText.UpdateTime) {
						matchTexts[text.ID] = text
					}
				} else {
					matchTexts[text.ID] = text
				}
			}
		default:
			break loop
		}
	}

	matchTextsList := []*Text{}
	for _, text := range matchTexts {
		if text == nil {
			continue
		}
		if text.IsDeleted {
			continue
		}
		matchTextsList = append(matchTextsList, text)
	}

	sort.Slice(matchTextsList, func(i, j int) bool {
		return matchTextsList[i].RelatedTime.After(matchTextsList[j].RelatedTime)
	})
	return matchTextsList, nil
}

func (t TextRepositories) Close(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(t))
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)

		go func(rep TextRepository) {
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

func (t TextRepositories) GetText(ctx context.Context, id string, updateTime *time.Time) (*Text, error) {
	var matchText *Text
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Text, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)

		go func(rep TextRepository) {
			defer wg.Done()
			matchTextInRep, err := rep.GetText(ctx, id, updateTime)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTextInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get text: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Text集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchTextInRep := <-ch:
			if matchTextInRep == nil {
				continue loop
			}
			if matchText != nil {
				if matchTextInRep.UpdateTime.Before(matchText.UpdateTime) {
					matchText = matchTextInRep
				}
			} else {
				matchText = matchTextInRep
			}
		default:
			break loop
		}
	}

	return matchText, nil
}

func (t TextRepositories) GetTextsByTargetID(ctx context.Context, target_id string) ([]*Text, error) {
	matchTexts := map[string]*Text{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Text, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)

		go func(rep TextRepository) {
			defer wg.Done()
			matchTextsInRep, err := rep.GetTextsByTargetID(ctx, target_id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTextsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get text histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Text集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchTextsInRep := <-ch:
			if matchTextsInRep == nil {
				continue loop
			}
			for _, text := range matchTextsInRep {
				if existText, exist := matchTexts[text.ID]; exist {
					if text.UpdateTime.After(existText.UpdateTime) {
						matchTexts[text.ID] = text
					}
				} else {
					matchTexts[text.ID] = text
				}
			}
		default:
			break loop
		}
	}

	textHistoriesList := []*Text{}
	for _, text := range matchTexts {
		if text == nil {
			continue
		}
		textHistoriesList = append(textHistoriesList, text)
	}

	sort.Slice(textHistoriesList, func(i, j int) bool {
		return textHistoriesList[i].UpdateTime.After(textHistoriesList[j].UpdateTime)
	})

	return textHistoriesList, nil
}

func (t TextRepositories) UpdateCache(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(t))
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)

		go func(rep TextRepository) {
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

func (t TextRepositories) GetPath(ctx context.Context, id string) (string, error) {
	err := fmt.Errorf("not implements TextReps.GetPath")
	return "", err
}

func (t TextRepositories) GetRepName(ctx context.Context) (string, error) {
	return "TextReps", nil
}

func (t TextRepositories) GetTextHistories(ctx context.Context, id string) ([]*Text, error) {
	textHistories := map[string]*Text{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Text, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)

		go func(rep TextRepository) {
			defer wg.Done()
			matchTextsInRep, err := rep.GetTextHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTextsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get text histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Text集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchTextsInRep := <-ch:
			if matchTextsInRep == nil {
				continue loop
			}
			for _, text := range matchTextsInRep {
				if existText, exist := textHistories[text.ID+text.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if text.UpdateTime.After(existText.UpdateTime) {
						textHistories[text.ID+text.UpdateTime.Format(sqlite3impl.TimeLayout)] = text
					}
				} else {
					textHistories[text.ID+text.UpdateTime.Format(sqlite3impl.TimeLayout)] = text
				}
			}
		default:
			break loop
		}
	}

	textHistoriesList := []*Text{}
	for _, text := range textHistories {
		if text == nil {
			continue
		}
		textHistoriesList = append(textHistoriesList, text)
	}

	sort.Slice(textHistoriesList, func(i, j int) bool {
		return textHistoriesList[i].UpdateTime.After(textHistoriesList[j].UpdateTime)
	})

	return textHistoriesList, nil
}

func (t TextRepositories) AddTextInfo(ctx context.Context, text *Text) error {
	err := fmt.Errorf("not implements TextReps.AddTextInfo")
	return err
}
