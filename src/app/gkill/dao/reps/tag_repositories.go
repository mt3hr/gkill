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

type TagRepositories []TagRepository

func (t TagRepositories) FindTags(ctx context.Context, query *find.FindQuery) ([]*Tag, error) {
	matchTags := map[string]*Tag{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Tag, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)

		done := threads.AllocateThread()
		go func(rep TagRepository) {
			defer done()
			defer wg.Done()
			matchTagsInRep, err := rep.FindTags(ctx, query)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTagsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find tag: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Tag集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchTagsInRep := <-ch:
			if matchTagsInRep == nil {
				continue loop
			}
			for _, tag := range matchTagsInRep {
				key := tag.ID
				if query.OnlyLatestData == nil || !*query.OnlyLatestData {
					key += fmt.Sprintf("%d", tag.UpdateTime.Unix())
				}

				if existTag, exist := matchTags[key]; exist {
					if tag.UpdateTime.After(existTag.UpdateTime) {
						matchTags[key] = tag
					}
				} else {
					matchTags[key] = tag
				}
			}
		default:
			break loop
		}
	}

	matchTagsList := []*Tag{}
	for _, tag := range matchTags {
		if tag == nil {
			continue
		}
		if tag.IsDeleted {
			continue
		}
		matchTagsList = append(matchTagsList, tag)
	}
	return matchTagsList, nil
}

func (t TagRepositories) Close(ctx context.Context) error {
	reps, err := t.UnWrapTyped()
	if err != nil {
		return err
	}

	existErr := false
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(reps))
	defer close(errch)

	// 並列処理
	for _, rep := range reps {
		wg.Add(1)

		done := threads.AllocateThread()
		go func(rep TagRepository) {
			defer done()
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

func (t TagRepositories) GetTag(ctx context.Context, id string, updateTime *time.Time) (*Tag, error) {
	var matchTag *Tag
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Tag, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)

		done := threads.AllocateThread()
		go func(rep TagRepository) {
			defer done()
			defer wg.Done()
			matchTagInRep, err := rep.GetTag(ctx, id, updateTime)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTagInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get tag: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Tag集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchTagInRep := <-ch:
			if matchTagInRep == nil {
				continue loop
			}
			if matchTag != nil {
				if matchTagInRep.UpdateTime.Before(matchTag.UpdateTime) {
					matchTag = matchTagInRep
				}
			} else {
				matchTag = matchTagInRep
			}
		default:
			break loop
		}
	}

	return matchTag, nil
}

func (t TagRepositories) GetTagsByTagName(ctx context.Context, tagname string) ([]*Tag, error) {
	matchTags := map[string]*Tag{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Tag, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)

		done := threads.AllocateThread()
		go func(rep TagRepository) {
			defer done()
			defer wg.Done()
			matchTagsInRep, err := rep.GetTagsByTagName(ctx, tagname)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTagsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get tag histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Tag集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchTagsInRep := <-ch:
			if matchTagsInRep == nil {
				continue loop
			}
			for _, tag := range matchTagsInRep {
				if existTag, exist := matchTags[tag.ID]; exist {
					if tag.UpdateTime.After(existTag.UpdateTime) {
						matchTags[tag.ID] = tag
					}
				} else {
					matchTags[tag.ID] = tag
				}
			}
		default:
			break loop
		}
	}

	tagHistoriesList := []*Tag{}
	for _, tag := range matchTags {
		if tag == nil {
			continue
		}
		if tag.IsDeleted {
			continue
		}
		tagHistoriesList = append(tagHistoriesList, tag)
	}
	return tagHistoriesList, nil
}

func (t TagRepositories) GetTagsByTargetID(ctx context.Context, target_id string) ([]*Tag, error) {
	matchTags := map[string]*Tag{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Tag, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)

		done := threads.AllocateThread()
		go func(rep TagRepository) {
			defer done()
			defer wg.Done()
			matchTagsInRep, err := rep.GetTagsByTargetID(ctx, target_id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTagsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get tag histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Tag集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchTagsInRep := <-ch:
			if matchTagsInRep == nil {
				continue loop
			}
			for _, tag := range matchTagsInRep {
				if existTag, exist := matchTags[tag.ID]; exist {
					if tag.UpdateTime.After(existTag.UpdateTime) {
						matchTags[tag.ID] = tag
					}
				} else {
					matchTags[tag.ID] = tag
				}
			}
		default:
			break loop
		}
	}

	tagHistoriesList := []*Tag{}
	for _, tag := range matchTags {
		if tag == nil {
			continue
		}
		if tag.IsDeleted {
			continue
		}
		tagHistoriesList = append(tagHistoriesList, tag)
	}
	return tagHistoriesList, nil
}

func (t TagRepositories) UpdateCache(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(t))
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)

		done := threads.AllocateThread()
		go func(rep TagRepository) {
			defer done()
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

func (t TagRepositories) GetPath(ctx context.Context, id string) (string, error) {
	// 並列処理
	matchPaths := []string{}
	trueValue := true
	ids := []string{id}
	for _, rep := range t {
		query := &find.FindQuery{
			IDs:    &ids,
			UseIDs: &trueValue,
		}
		tags, err := rep.FindTags(ctx, query)
		if len(tags) == 0 || err != nil {
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

func (t TagRepositories) GetRepName(ctx context.Context) (string, error) {
	return "TagReps", nil
}

func (t TagRepositories) GetTagHistories(ctx context.Context, id string) ([]*Tag, error) {
	tagHistories := map[string]*Tag{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Tag, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)

		done := threads.AllocateThread()
		go func(rep TagRepository) {
			defer done()
			defer wg.Done()
			matchTagsInRep, err := rep.GetTagHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTagsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get tag histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Tag集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchTagsInRep := <-ch:
			if matchTagsInRep == nil {
				continue loop
			}
			for _, tag := range matchTagsInRep {
				if existTag, exist := tagHistories[tag.ID+tag.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if tag.UpdateTime.After(existTag.UpdateTime) {
						tagHistories[tag.ID+tag.UpdateTime.Format(sqlite3impl.TimeLayout)] = tag
					}
				} else {
					tagHistories[tag.ID+tag.UpdateTime.Format(sqlite3impl.TimeLayout)] = tag
				}
			}
		default:
			break loop
		}
	}

	tagHistoriesList := []*Tag{}
	for _, tag := range tagHistories {
		if tag == nil {
			continue
		}
		tagHistoriesList = append(tagHistoriesList, tag)
	}

	sort.Slice(tagHistoriesList, func(i, j int) bool {
		return tagHistoriesList[i].UpdateTime.After(tagHistoriesList[j].UpdateTime)
	})

	return tagHistoriesList, nil
}
func (t TagRepositories) GetTagHistoriesByRepName(ctx context.Context, id string, repName *string) ([]*Tag, error) {
	tagHistories := map[string]*Tag{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Tag, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)

		done := threads.AllocateThread()
		go func(rep TagRepository) {
			defer done()
			defer wg.Done()

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

			matchTagsInRep, err := rep.GetTagHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTagsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get tag histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Tag集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchTagsInRep := <-ch:
			if matchTagsInRep == nil {
				continue loop
			}
			for _, tag := range matchTagsInRep {
				if existTag, exist := tagHistories[tag.ID+tag.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if tag.UpdateTime.After(existTag.UpdateTime) {
						tagHistories[tag.ID+tag.UpdateTime.Format(sqlite3impl.TimeLayout)] = tag
					}
				} else {
					tagHistories[tag.ID+tag.UpdateTime.Format(sqlite3impl.TimeLayout)] = tag
				}
			}
		default:
			break loop
		}
	}

	tagHistoriesList := []*Tag{}
	for _, tag := range tagHistories {
		if tag == nil {
			continue
		}
		tagHistoriesList = append(tagHistoriesList, tag)
	}

	sort.Slice(tagHistoriesList, func(i, j int) bool {
		return tagHistoriesList[i].UpdateTime.After(tagHistoriesList[j].UpdateTime)
	})

	return tagHistoriesList, nil
}

func (t TagRepositories) AddTagInfo(ctx context.Context, tag *Tag) error {
	err := fmt.Errorf("not implements TagReps.AddTagInfo")
	return err
}

func (t TagRepositories) GetAllTagNames(ctx context.Context) ([]string, error) {
	tagNames := []string{}
	tagNamesMap := map[string]struct{}{}
	tags, err := t.GetAllTags(ctx)
	if err != nil {
		err = fmt.Errorf("error at get all tags: %w", err)
		return nil, err
	}

	latestTags := map[string]*Tag{}
	for _, tag := range tags {
		if existTag, exist := latestTags[tag.ID]; exist {
			if tag.UpdateTime.After(existTag.UpdateTime) {
				latestTags[tag.ID] = tag
			}
		} else {
			latestTags[tag.ID] = tag
		}
	}

	for _, tag := range latestTags {
		if tag == nil {
			continue
		}
		if tag.IsDeleted {
			continue
		}
		tagNamesMap[tag.Tag] = struct{}{}
	}
	for tag := range tagNamesMap {
		tagNames = append(tagNames, tag)
	}
	return tagNames, nil
}

func (t TagRepositories) GetAllTags(ctx context.Context) ([]*Tag, error) {
	allTags := map[string]*Tag{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Tag, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)

		done := threads.AllocateThread()
		go func(rep TagRepository) {
			defer done()
			defer wg.Done()
			matchTagsInRep, err := rep.GetAllTags(ctx)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTagsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get tag histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Tag集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchTagsInRep := <-ch:
			if matchTagsInRep == nil {
				continue loop
			}
			for _, tag := range matchTagsInRep {
				if existTag, exist := allTags[tag.ID]; exist {
					if tag.UpdateTime.After(existTag.UpdateTime) {
						allTags[tag.ID] = tag
					}
				} else {
					allTags[tag.ID] = tag
				}
			}
		default:
			break loop
		}
	}

	allTagsList := []*Tag{}
	for _, tag := range allTags {
		if tag == nil {
			continue
		}
		if tag.IsDeleted {
			continue
		}
		allTagsList = append(allTagsList, tag)
	}

	return allTagsList, nil
}

func (t TagRepositories) UnWrapTyped() ([]TagRepository, error) {
	unwraped := []TagRepository{}
	for _, rep := range t {
		u, err := rep.UnWrapTyped()
		if err != nil {
			return nil, err
		}
		unwraped = append(unwraped, u...)
	}
	return unwraped, nil
}
