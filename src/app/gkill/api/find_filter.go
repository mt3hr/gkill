package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

const (
	NoTags = "no tags"
	R      = math.Pi / 180
)

type FindFilter struct {
}

func (f *FindFilter) FindKyous(ctx context.Context, userID string, device string, gkillDAOManager *dao.GkillDAOManager, findQuery *find.FindQuery) ([]*reps.Kyou, []*message.GkillError, error) {
	findKyouContext := &FindKyouContext{}

	// QueryをContextに入れる
	// jsonからパースする
	findKyouContext.ParsedFindQuery = findQuery
	findKyouContext.MatchReps = map[string]reps.Repository{}
	findKyouContext.AllTags = map[string]*reps.Tag{}
	findKyouContext.MatchTags = map[string]*reps.Tag{}
	findKyouContext.MatchTexts = map[string]*reps.Text{}
	findKyouContext.MatchTimeIssAtFindTimeIs = map[string]*reps.TimeIs{}
	findKyouContext.MatchTimeIssAtFilterTags = map[string]*reps.TimeIs{}
	findKyouContext.MatchTimeIsTags = map[string]*reps.Tag{}
	findKyouContext.MatchTimeIsTexts = map[string]*reps.Text{}
	findKyouContext.MatchKyousCurrent = map[string]*reps.Kyou{}
	findKyouContext.MatchKyousAtFindKyou = map[string]*reps.Kyou{}
	findKyouContext.MatchKyousAtFilterTags = map[string]*reps.Kyou{}
	findKyouContext.MatchKyousAtFilterTimeIs = map[string]*reps.Kyou{}
	findKyouContext.MatchKyousAtFilterLocation = map[string]*reps.Kyou{}
	i := 1

	// フィルタ
	gkillErr, err := f.getRepositories(ctx, userID, device, gkillDAOManager, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at get repositories: %w", err)
		return nil, gkillErr, err
	}
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchKyousCurrent) //TODO けして
	i++                                                            //TODO けして
	gkillErr, err = f.selectMatchRepsFromQuery(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at select match reps: %w", err)
		return nil, gkillErr, err
	}
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchKyousCurrent) //TODO けして
	i++                                                            //TODO けして
	gkillErr, err = f.updateCache(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at update cache: %w", err)
		return nil, gkillErr, err
	}
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchKyousCurrent) //TODO けして
	i++                                                            //TODO けして
	gkillErr, err = f.parseTagFilterModeFromQuery(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at parse tag filter mode: %w", err)
		return nil, gkillErr, err
	}
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchKyousCurrent) //TODO けして
	i++                                                            //TODO けして
	gkillErr, err = f.parseTimeIsTagFilterModeFromQuery(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at parse timeis tag filter mode: %w", err)
		return nil, gkillErr, err
	}
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchKyousCurrent) //TODO けして
	i++                                                            //TODO けして
	gkillErr, err = f.getAllTags(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at get all tags: %w", err)
		return nil, gkillErr, err
	}
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchKyousCurrent) //TODO けして
	i++                                                            //TODO けして
	gkillErr, err = f.findTags(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at find tags: %w", err)
		return nil, gkillErr, err
	}
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchKyousCurrent) //TODO けして
	i++                                                            //TODO けして
	gkillErr, err = f.findTexts(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at find texts: %w", err)
		return nil, gkillErr, err
	}
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchKyousCurrent) //TODO けして
	i++                                                            //TODO けして
	gkillErr, err = f.findTimeIsTexts(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at find timeis texts: %w", err)
		return nil, gkillErr, err
	}
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchKyousCurrent) //TODO けして
	i++                                                            //TODO けして
	gkillErr, err = f.findTimeIs(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at find timeis: %w", err)
		return nil, gkillErr, err
	}
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchKyousCurrent) //TODO けして
	i++                                                            //TODO けして
	gkillErr, err = f.findTimeIsTags(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at find timeis tags: %w", err)
		return nil, gkillErr, err
	}
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchKyousCurrent) //TODO けして
	i++                                                            //TODO けして
	gkillErr, err = f.filterTagsTimeIs(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at filter tags timeis: %w", err)
		return nil, gkillErr, err
	}
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchKyousCurrent) //TODO けして
	i++                                                            //TODO けして
	gkillErr, err = f.findKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at find kyous: %w", err)
		return nil, gkillErr, err
	}
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchKyousCurrent) //TODO けして
	i++                                                            //TODO けして
	gkillErr, err = f.filterTagsKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at filter tags kyous: %w", err)
		return nil, gkillErr, err
	}
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchKyousCurrent) //TODO けして
	i++                                                            //TODO けして
	gkillErr, err = f.filterPlaingTimeIsKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at filter plaing time is kyous: %w", err)
		return nil, gkillErr, err
	}
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchKyousCurrent) //TODO けして
	i++                                                            //TODO けして
	gkillErr, err = f.filterLocationKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at filter location kyous: %w", err)
		return nil, gkillErr, err
	}
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchKyousCurrent) //TODO けして
	i++                                                            //TODO けして
	for _, rep := range findKyouContext.MatchKyousCurrent {
		findKyouContext.ResultKyous = append(findKyouContext.ResultKyous, rep)
	}
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchKyousCurrent) //TODO けして
	i++                                                            //TODO けして
	gkillErr, err = f.sortResultKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at sort result kyous: %w", err)
		return nil, gkillErr, err
	}
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchKyousCurrent) //TODO けして
	i++                                                            //TODO けして

	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchTimeIsTags)          //TODO けして
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchTimeIsTexts)         //TODO けして
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchTimeIssAtFilterTags) //TODO けして
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchTimeIssAtFindTimeIs) //TODO けして
	fmt.Printf("%d = %+v\n", i, findKyouContext.MatchKyousAtFilterTimeIs) //TODO けして

	json.NewEncoder(os.Stdout).Encode(findKyouContext)

	return findKyouContext.ResultKyous, nil, nil
}

func (f *FindFilter) getRepositories(ctx context.Context, userID string, device string, gkillDAOManager *dao.GkillDAOManager, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	var err error
	repositories, err := gkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		return nil, err
	}
	findCtx.Repositories = repositories

	if findCtx.ParsedFindQuery.UpdateCache != nil && *findCtx.ParsedFindQuery.UpdateCache {
		err := repositories.UpdateCache(ctx)
		if err != nil {
			err = fmt.Errorf("error at update repositories cache: %w", err)
			return nil, err
		}
	}
	return nil, nil
}

func (f *FindFilter) selectMatchRepsFromQuery(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	var err error
	repositories := findCtx.Repositories

	existErr := false
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(repositories.Reps))
	defer close(errch)

	targetRepNames := []string{}
	if findCtx.ParsedFindQuery.Reps != nil {
		targetRepNames = *findCtx.ParsedFindQuery.Reps
	}

	// 並列処理
	m := &sync.Mutex{}
	for _, rep := range repositories.Reps {
		wg.Add(1)
		go func(rep reps.Repository) {
			defer wg.Done()

			// PlaingだったらTimeIsRep以外は無視する
			if findCtx.ParsedFindQuery.UsePlaing != nil && *findCtx.ParsedFindQuery.UsePlaing {
				_, isTimeIsRep := rep.(reps.TimeIsRepository)
				if !isTimeIsRep {
					errch <- nil
					return
				}
			}

			repName, err := rep.GetRepName(ctx)
			if err != nil {
				errch <- err
				return
			}

			for _, targetRepName := range targetRepNames {
				if targetRepName == repName {
					m.Lock()
					if _, exist := findCtx.MatchReps[repName]; !exist {
						findCtx.MatchReps[repName] = rep
					}
					m.Unlock()
				}
			}
			errch <- nil
		}(rep)
	}
	wg.Wait()
	// エラー集約
	for range len(repositories.Reps) {
		e := <-errch
		if e != nil {
			err = fmt.Errorf("error at update cache: %w: %w", e, err)
			existErr = true
		}
	}
	if existErr {
		return nil, err
	}
	return nil, nil
}

func (f *FindFilter) updateCache(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	var err error
	existErr := false
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(findCtx.MatchReps))
	defer close(errch)

	// 並列処理
	for _, rep := range findCtx.MatchReps {
		wg.Add(1)
		go func(rep reps.Repository) {
			defer wg.Done()
			err = rep.UpdateCache(ctx)
			if err != nil {
				errch <- err
				return
			}
			errch <- nil
		}(rep)
	}
	wg.Wait()

	// エラー集約
	for range len(findCtx.MatchReps) {
		e := <-errch
		if e != nil {
			err = fmt.Errorf("error at update cache: %w: %w:", e, err)
			existErr = true
		}
	}
	if existErr {
		return nil, err
	}
	falseValue := false
	findCtx.ParsedFindQuery.UpdateCache = &falseValue
	return nil, nil
}

func (f *FindFilter) getAllTags(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	var err error

	lenOfTagReps := len(findCtx.Repositories.TagReps)

	// 全タグ取得用検索クエリ
	falseValue := false
	findTagsQuery := &find.FindQuery{IsDeleted: &falseValue}

	existErr := false
	wg := &sync.WaitGroup{}
	tagsCh := make(chan []*reps.Tag, lenOfTagReps)
	errch := make(chan error, lenOfTagReps)
	defer close(tagsCh)
	defer close(errch)

	// 並列処理
	for _, rep := range findCtx.Repositories.TagReps {
		wg.Add(1)
		go func(tagRep reps.TagRepository) {
			defer wg.Done()
			tags, err := tagRep.FindTags(ctx, findTagsQuery)
			if err != nil {
				errch <- err
				return
			}
			tagsCh <- tags
			errch <- nil
		}(rep)
	}
	wg.Wait()
	// エラー集約
	for range lenOfTagReps {
		e := <-errch
		if e != nil {
			err = fmt.Errorf("error at get all tags: %w: %w:", e, err)
			existErr = true
		}
	}
	if existErr {
		return nil, err
	}
	// Tag集約
	for range lenOfTagReps {
		matchTags := <-tagsCh
		for _, tag := range matchTags {
			if existTag, exist := findCtx.AllTags[tag.ID]; exist {
				if tag.UpdateTime.After(existTag.UpdateTime) {
					findCtx.AllTags[tag.ID] = tag
				}
			} else {
				findCtx.AllTags[tag.ID] = tag
			}
		}
	}

	return nil, nil
}

func (f *FindFilter) findTimeIsTags(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	var err error

	// タグを使わない場合は全タグを使う
	if findCtx.ParsedFindQuery.UseTimeIsTags == nil || !(*findCtx.ParsedFindQuery.UseTimeIsTags) {
		for _, tag := range findCtx.AllTags {
			findCtx.MatchTimeIsTags[tag.Tag] = tag
		}
		return nil, nil
	}

	lenOfTagReps := len(findCtx.Repositories.TagReps)

	// 対象タグ取得用検索クエリ
	trueValue := true
	falseValue := false
	findTagsQueries := []*find.FindQuery{}
	for _, tag := range *findCtx.ParsedFindQuery.TimeIsTags {
		if tag == NoTags {
			continue
		}
		words := []string{tag}
		findTagsQuery := &find.FindQuery{IsDeleted: &falseValue,
			UseWords: &trueValue,
			Words:    &words,
			WordsAnd: &falseValue,
		}
		findTagsQueries = append(findTagsQueries, findTagsQuery)
	}

	existErr := false
	wg := &sync.WaitGroup{}
	tagsCh := make(chan []*reps.Tag, lenOfTagReps)
	errch := make(chan error, lenOfTagReps)
	defer close(tagsCh)
	defer close(errch)

	// 並列処理
	for _, rep := range findCtx.Repositories.TagReps {
		wg.Add(1)
		go func(tagRep reps.TagRepository) {
			defer wg.Done()
			tagsInRep := []*reps.Tag{}
			for _, findTagsQuery := range findTagsQueries {
				tags, err := tagRep.FindTags(ctx, findTagsQuery)
				if err != nil {
					errch <- err
					return
				}
				tagsInRep = append(tagsInRep, tags...)
			}
			tagsCh <- tagsInRep
			errch <- nil
		}(rep)
	}
	wg.Wait()
	// エラー集約
	for range lenOfTagReps {
		e := <-errch
		if e != nil {
			err = fmt.Errorf("error at find timeis tags: %w: %w:", e, err)
			existErr = true
		}
	}

	if existErr {
		return nil, err
	}

	// TimeIsのTag集約
	for range lenOfTagReps {
		matchTags := <-tagsCh
		for _, tag := range matchTags {
			if existTag, exist := findCtx.MatchTimeIsTags[tag.ID]; exist {
				if tag.UpdateTime.After(existTag.UpdateTime) {
					findCtx.MatchTimeIsTags[tag.ID] = tag
				}
			} else {
				findCtx.MatchTimeIsTags[tag.ID] = tag
			}
		}
	}

	return nil, nil
}

func (f *FindFilter) findTags(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	var err error

	lenOfTagReps := 0
	for range findCtx.Repositories.TagReps {
		lenOfTagReps++
	}

	// 対象タグ取得用検索クエリ
	trueValue := true
	falseValue := false
	findTagsQueries := []*find.FindQuery{}
	for _, tag := range *findCtx.ParsedFindQuery.Tags {
		if tag == NoTags {
			continue
		}
		words := []string{tag}
		findTagsQuery := &find.FindQuery{
			IsDeleted: &falseValue,
			UseWords:  &trueValue,
			Words:     &words,
			WordsAnd:  &falseValue,
		}
		findTagsQueries = append(findTagsQueries, findTagsQuery)
	}

	existErr := false
	wg := &sync.WaitGroup{}
	tagsCh := make(chan []*reps.Tag, lenOfTagReps)
	errch := make(chan error, lenOfTagReps)
	defer close(tagsCh)
	defer close(errch)

	// 並列処理
	for _, rep := range findCtx.Repositories.TagReps {
		wg.Add(1)
		go func(tagRep reps.TagRepository) {
			defer wg.Done()
			tagsInRep := []*reps.Tag{}
			for _, findTagsQuery := range findTagsQueries {
				tags, err := tagRep.FindTags(ctx, findTagsQuery)
				if err != nil {
					errch <- err
					return
				}
				tagsInRep = append(tagsInRep, tags...)
			}
			tagsCh <- tagsInRep
			errch <- nil
		}(rep)
	}
	wg.Wait()
	// エラー集約
	for range lenOfTagReps {
		e := <-errch
		if e != nil {
			err = fmt.Errorf("error at find  tags: %w: %w:", e, err)
			existErr = true
		}
	}

	if existErr {
		return nil, err
	}

	// Tag集約
	for range lenOfTagReps {
		matchTags := <-tagsCh
		for _, tag := range matchTags {
			if existTag, exist := findCtx.MatchTags[tag.ID]; exist {
				if tag.UpdateTime.After(existTag.UpdateTime) {
					findCtx.MatchTags[tag.ID] = tag
				}
			} else {
				findCtx.MatchTags[tag.ID] = tag
			}
		}
	}

	return nil, nil
}

func (f *FindFilter) parseTagFilterModeFromQuery(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	tagFilterModeIsAnd := false
	if findCtx.ParsedFindQuery.TagsAnd != nil {
		tagFilterModeIsAnd = *findCtx.ParsedFindQuery.TagsAnd
	}
	if tagFilterModeIsAnd {
		var tagFilterMode find.TagFilterMode
		tagFilterMode = find.And
		findCtx.TagFilterMode = &tagFilterMode
	} else {
		var tagFilterMode find.TagFilterMode
		tagFilterMode = find.Or
		findCtx.TagFilterMode = &tagFilterMode
	}
	return nil, nil
}

func (f *FindFilter) parseTimeIsTagFilterModeFromQuery(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	timeisTagFilterModeIsAnd := false
	if findCtx.ParsedFindQuery.TimeIsTagsAnd != nil {
		timeisTagFilterModeIsAnd = *findCtx.ParsedFindQuery.TimeIsTagsAnd
	}
	if timeisTagFilterModeIsAnd {
		var timeisTagFilterMode find.TagFilterMode
		timeisTagFilterMode = find.And
		findCtx.TagFilterMode = &timeisTagFilterMode
	} else {
		var timeisTagFilterMode find.TagFilterMode
		timeisTagFilterMode = find.Or
		findCtx.TagFilterMode = &timeisTagFilterMode
	}
	return nil, nil
}

func (f *FindFilter) findKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	var err error

	// text検索用クエリ
	targetIDs := []string{}
	for _, text := range findCtx.MatchTexts {
		targetIDs = append(targetIDs, text.TargetID)
	}
	trueValue := true
	matchTextFindByIDQuery := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &targetIDs,
	}

	// repで検索
	kyous, err := findCtx.Repositories.FindKyous(ctx, findCtx.ParsedFindQuery)
	if err != nil {
		return nil, err
	}
	json.NewEncoder(os.Stdout).Encode(kyous)
	// textでマッチしたものをID検索
	textMatchKyous := []*reps.Kyou{}
	if len(targetIDs) != 0 {
		textMatchKyous, err = findCtx.Repositories.FindKyous(ctx, matchTextFindByIDQuery)
		if err != nil {
			return nil, err
		}
	}
	kyous = append(kyous, textMatchKyous...)

	for _, kyou := range kyous {
		existKyou, exist := findCtx.MatchKyousAtFindKyou[kyou.ID]
		if exist {
			if kyou.UpdateTime.After(existKyou.UpdateTime) {
				findCtx.MatchKyousAtFindKyou[kyou.ID] = kyou
			}
		} else {
			findCtx.MatchKyousAtFindKyou[kyou.ID] = kyou
		}
	}

	// 削除済み判定
	isDeleted := false
	if findCtx.ParsedFindQuery.IsDeleted != nil {
		isDeleted = *findCtx.ParsedFindQuery.IsDeleted
	}
	for _, kyou := range findCtx.MatchKyousAtFindKyou {
		if isDeleted {
			if !kyou.IsDeleted {
				delete(findCtx.MatchKyousAtFindKyou, kyou.ID)
			}
		} else {
			if kyou.IsDeleted {
				delete(findCtx.MatchKyousAtFindKyou, kyou.ID)
			}
		}
	}

	findCtx.MatchKyousCurrent = findCtx.MatchKyousAtFindKyou
	return nil, nil
}

func (f *FindFilter) filterTagsKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if findCtx.TagFilterMode != nil && *findCtx.TagFilterMode == find.Or {
		// ORの場合のフィルタリング処理

		// タグ対象Kyouリスト
		matchOrTagKyous := map[string]*reps.Kyou{}
		for _, kyou := range findCtx.MatchKyousCurrent {
			for _, tag := range findCtx.MatchTags {
				if kyou.ID == tag.TargetID {
					matchOrTagKyous[kyou.ID] = kyou
				}
			}
		}

		// タグ無しKyouリスト
		noTagKyous := map[string]*reps.Kyou{}
		for _, kyou := range findCtx.MatchKyousCurrent {
			relatedTagKyou := false
			for _, tag := range findCtx.AllTags {
				if kyou.ID == tag.TargetID {
					relatedTagKyou = true
				}
			}
			if !relatedTagKyou {
				if existKyou, exist := noTagKyous[kyou.ID]; exist {
					if kyou.UpdateTime.After(existKyou.UpdateTime) {
						noTagKyous[kyou.ID] = kyou
					}
				} else {
					noTagKyous[kyou.ID] = kyou
				}
			}
		}

		// タグ無し込であればそれもいれる
		existNoTags := false
		tags := []string{}
		if findCtx.ParsedFindQuery.Tags != nil {
			tags = *findCtx.ParsedFindQuery.Tags
		}

		for _, tag := range tags {
			if tag == NoTags {
				existNoTags = true
				break
			}
		}

		// タグフィルタしたものをCtxに収める
		for _, kyou := range matchOrTagKyous {
			if existKyou, exist := findCtx.MatchKyousAtFilterTags[kyou.ID]; exist {
				if kyou.UpdateTime.After(existKyou.UpdateTime) {
					findCtx.MatchKyousAtFilterTags[kyou.ID] = kyou
				}
			} else {
				findCtx.MatchKyousAtFilterTags[kyou.ID] = kyou
			}
		}
		if existNoTags {
			for _, kyou := range noTagKyous {
				if existKyou, exist := findCtx.MatchKyousAtFilterTags[kyou.ID]; exist {
					if kyou.UpdateTime.After(existKyou.UpdateTime) {
						findCtx.MatchKyousAtFilterTags[kyou.ID] = kyou
					}
				} else {
					findCtx.MatchKyousAtFilterTags[kyou.ID] = kyou
				}
			}
		}
		findCtx.MatchKyousCurrent = findCtx.MatchKyousAtFilterTags
	} else if findCtx.TagFilterMode != nil && *findCtx.TagFilterMode == find.And {
		// ANDの場合のフィルタリング処理

		tagNameMap := map[string]map[string]*reps.Kyou{} // map[タグ名][kyou.ID（tagTargetID）] = reps.kyou

		for _, kyou := range findCtx.MatchKyousCurrent {
			for _, tag := range findCtx.MatchTags {
				if kyou.ID == tag.TargetID {
					if existKyou, exist := tagNameMap[tag.Tag][kyou.ID]; exist {
						if kyou.UpdateTime.After(existKyou.UpdateTime) {
							tagNameMap[tag.Tag][kyou.ID] = kyou
						}
					} else {
						tagNameMap[tag.Tag] = map[string]*reps.Kyou{}
						tagNameMap[tag.Tag][kyou.ID] = kyou
					}
				}
			}
		}

		// タグ無しの情報もtagNameMapにいれる
		existNoTags := false
		tags := []string{}
		if findCtx.ParsedFindQuery.Tags != nil {
			tags = *findCtx.ParsedFindQuery.Tags
		}

		for _, tag := range tags {
			if tag == NoTags {
				existNoTags = true
				break
			}
		}

		if existNoTags {
			for _, kyou := range findCtx.MatchKyousCurrent {
				relatedTagKyou := false
				for _, tag := range findCtx.AllTags {
					if kyou.ID == tag.TargetID {
						relatedTagKyou = true
					}
				}
				if !relatedTagKyou {
					if existKyou, exist := tagNameMap[NoTags][kyou.ID]; exist {
						if kyou.UpdateTime.After(existKyou.UpdateTime) {
							tagNameMap[NoTags][kyou.ID] = kyou
						}
					} else {
						tagNameMap[NoTags] = map[string]*reps.Kyou{}
						tagNameMap[NoTags][kyou.ID] = kyou
					}
				}
			}
		}

		// tagNameMapの全部のタグ名に存在するKyouだけを抽出
		hasAllMatchTagsKyousMap := map[string]*reps.Kyou{}
		index := 0
		for _, kyouIDMap := range tagNameMap {
			switch index {
			case 0:
				// 初回ループは全部いれる
				for _, kyou := range kyouIDMap {
					if existKyou, exist := hasAllMatchTagsKyousMap[kyou.ID]; exist {
						if kyou.UpdateTime.After(existKyou.UpdateTime) {
							hasAllMatchTagsKyousMap[kyou.ID] = kyou
						}
					} else {
						hasAllMatchTagsKyousMap[kyou.ID] = kyou
					}
				}
			default:
				matchThisLoopKyousMap := map[string]*reps.Kyou{}
				for _, kyou := range kyouIDMap {
					// 初回ループ以外は、
					// 以前のタグにマッチしたもの（hasAllMatchTagsKyous）にあり、かつ
					// 今回のタグにマッチしたもの　をいれる。
					if existKyou, exist := hasAllMatchTagsKyousMap[kyou.ID]; exist {
						if _, exist := matchThisLoopKyousMap[kyou.ID]; exist {
							if kyou.UpdateTime.After(existKyou.UpdateTime) {
								matchThisLoopKyousMap[kyou.ID] = kyou
							}
						} else {
							matchThisLoopKyousMap[kyou.ID] = kyou
						}
					}
				}
				hasAllMatchTagsKyousMap = matchThisLoopKyousMap
			}
			index++
		}
		findCtx.MatchKyousAtFilterTags = hasAllMatchTagsKyousMap
		findCtx.MatchKyousCurrent = hasAllMatchTagsKyousMap
	}

	return nil, nil
}

func (f *FindFilter) filterTagsTimeIs(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if findCtx.ParsedFindQuery.UseTimeIsTags == nil || !(*findCtx.ParsedFindQuery.UseTimeIsTags) {
		findCtx.MatchTimeIssAtFilterTags = findCtx.MatchTimeIssAtFindTimeIs
		return nil, nil
	}

	if (findCtx.TimeIsTagFilterMode != nil && *findCtx.TimeIsTagFilterMode == find.Or) || findCtx.TimeIsTagFilterMode == nil {
		// ORの場合のフィルタリング処理

		// タグ対象Kyouリスト
		matchOrTagTimeIss := map[string]*reps.TimeIs{}
		for _, timeis := range findCtx.MatchTimeIssAtFindTimeIs {
			for _, tag := range findCtx.MatchTimeIsTags {
				if timeis.ID == tag.TargetID {
					matchOrTagTimeIss[timeis.ID] = timeis
				}
			}
		}

		// タグ無しKyouリスト
		noTagTimeIss := map[string]*reps.TimeIs{}
		for _, timeis := range findCtx.MatchTimeIssAtFindTimeIs {
			relatedTagTimeIs := false
			for _, tag := range findCtx.AllTags {
				if timeis.ID == tag.TargetID {
					relatedTagTimeIs = true
				}
			}
			if !relatedTagTimeIs {
				if existTimeIs, exist := noTagTimeIss[timeis.ID]; exist {
					if timeis.UpdateTime.After(existTimeIs.UpdateTime) {
						noTagTimeIss[timeis.ID] = timeis
					}
				} else {
					noTagTimeIss[timeis.ID] = timeis
				}
			}
		}

		// タグ無し込であればそれもいれる
		existNoTags := false
		tags := []string{}
		if findCtx.ParsedFindQuery.TimeIsTags != nil {
			tags = *findCtx.ParsedFindQuery.TimeIsTags
		}
		for _, tag := range tags {
			if tag == NoTags {
				existNoTags = true
				break
			}
		}

		// タグフィルタしたものをCtxに収める
		for _, timeis := range matchOrTagTimeIss {
			if existTimeIs, exist := findCtx.MatchTimeIssAtFilterTags[timeis.ID]; exist {
				if timeis.UpdateTime.After(existTimeIs.UpdateTime) {
					findCtx.MatchTimeIssAtFilterTags[timeis.ID] = timeis
				}
			} else {
				findCtx.MatchTimeIssAtFilterTags[timeis.ID] = timeis
			}
		}
		if existNoTags {
			for _, timeis := range noTagTimeIss {
				if existTimeIs, exist := findCtx.MatchTimeIssAtFilterTags[timeis.ID]; exist {
					if timeis.UpdateTime.After(existTimeIs.UpdateTime) {
						findCtx.MatchTimeIssAtFilterTags[timeis.ID] = timeis
					}
				} else {
					findCtx.MatchTimeIssAtFilterTags[timeis.ID] = timeis
				}
			}
		}
	} else if findCtx.TimeIsTagFilterMode != nil && *findCtx.TimeIsTagFilterMode == find.And {
		// ANDの場合のフィルタリング処理

		tagNameMap := map[string]map[string]*reps.TimeIs{} // map[タグ名][kyou.ID（tagTargetID）] = reps.TimeIs

		for _, timeis := range findCtx.MatchTimeIssAtFindTimeIs {
			for _, tag := range findCtx.MatchTimeIsTags {
				if timeis.ID == tag.TargetID {
					if existTimeIs, exist := tagNameMap[tag.Tag][timeis.ID]; exist {
						if timeis.UpdateTime.After(existTimeIs.UpdateTime) {
							tagNameMap[tag.Tag][timeis.ID] = timeis
						}
					} else {
						tagNameMap[tag.Tag][timeis.ID] = timeis
					}
				}
			}
		}

		// タグ無しの情報もtagNameMapにいれる
		existNoTags := false
		tags := []string{}
		if findCtx.ParsedFindQuery.TimeIsTags != nil {
			tags = *findCtx.ParsedFindQuery.TimeIsTags
		}

		for _, tag := range tags {
			if tag == NoTags {
				existNoTags = true
				break
			}
		}
		if existNoTags {
			for _, timeis := range findCtx.MatchTimeIssAtFindTimeIs {
				relatedTagTimeIs := false
				for _, tag := range findCtx.AllTags {
					if timeis.ID == tag.TargetID {
						relatedTagTimeIs = true
					}
				}
				if !relatedTagTimeIs {
					if existTimeIs, exist := tagNameMap[NoTags][timeis.ID]; exist {
						if timeis.UpdateTime.After(existTimeIs.UpdateTime) {
							tagNameMap[NoTags][timeis.ID] = timeis
						}
					} else {
						tagNameMap[NoTags][timeis.ID] = timeis
					}
				}
			}
		}

		// tagNameMapの全部のタグ名に存在するTimeIsだけを抽出
		hasAllMatchTagsTimeIssMap := map[string]*reps.TimeIs{}
		index := 0
		for _, timeisIDMap := range tagNameMap {
			switch index {
			case 0:
				// 初回ループは全部いれる
				for _, timeis := range timeisIDMap {
					if existTimeIs, exist := hasAllMatchTagsTimeIssMap[timeis.ID]; exist {
						if timeis.UpdateTime.After(existTimeIs.UpdateTime) {
							hasAllMatchTagsTimeIssMap[timeis.ID] = timeis
						}
					} else {
						hasAllMatchTagsTimeIssMap[timeis.ID] = timeis
					}
				}
			default:
				matchThisLoopTimeIssMap := map[string]*reps.TimeIs{}
				for _, timeis := range timeisIDMap {
					// 初回ループ以外は、
					// 以前のタグにマッチしたもの（hasAllMatchTagsKyous）にあり、かつ
					// 今回のタグにマッチしたもの　をいれる。
					if existTimeis, exist := hasAllMatchTagsTimeIssMap[timeis.ID]; exist {
						if _, exist := matchThisLoopTimeIssMap[timeis.ID]; exist {
							if timeis.UpdateTime.After(existTimeis.UpdateTime) {
								matchThisLoopTimeIssMap[timeis.ID] = timeis
							}
						} else {
							matchThisLoopTimeIssMap[timeis.ID] = timeis
						}
					}
				}
				hasAllMatchTagsTimeIssMap = matchThisLoopTimeIssMap
			}
			index++
		}
		findCtx.MatchTimeIssAtFilterTags = hasAllMatchTagsTimeIssMap
	}
	return nil, nil
}

func (f *FindFilter) filterPlaingTimeIsKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if findCtx.ParsedFindQuery.UseTimeIs == nil || !(*findCtx.ParsedFindQuery.UseTimeIs) {
		return nil, nil
	}

	for _, timeis := range findCtx.MatchTimeIssAtFilterTags {
		for _, kyou := range findCtx.MatchKyousCurrent {
			if (timeis.EndTime != nil && kyou.RelatedTime.After(timeis.StartTime) && kyou.RelatedTime.Before(*timeis.EndTime)) || (timeis.EndTime == nil && kyou.RelatedTime.After(timeis.StartTime)) {
				if existKyou, exist := findCtx.MatchKyousAtFilterTimeIs[kyou.ID]; exist {
					if kyou.UpdateTime.After(existKyou.UpdateTime) {
						findCtx.MatchKyousAtFilterTimeIs[kyou.ID] = kyou
					}
				} else {
					findCtx.MatchKyousAtFilterTimeIs[kyou.ID] = kyou
				}
			}
		}
	}
	findCtx.MatchKyousCurrent = findCtx.MatchKyousAtFilterTimeIs
	return nil, nil
}

func (f *FindFilter) findTimeIs(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	var err error
	if findCtx.ParsedFindQuery.UseTimeIs == nil || !*findCtx.ParsedFindQuery.UseTimeIs {
		return nil, nil
	}

	trueValue := true

	// 対象TimeIs取得用検索クエリ
	timeisFindKyouQuery := &find.FindQuery{
		UseWords: &trueValue,
		Words:    findCtx.ParsedFindQuery.TimeIsWords,
		NotWords: findCtx.ParsedFindQuery.TimeIsNotWords,
		WordsAnd: findCtx.ParsedFindQuery.TimeIsWordsAnd,
	}

	// text検索用クエリ
	targetIDs := []string{}
	for _, text := range findCtx.MatchTimeIsTexts {
		targetIDs = append(targetIDs, text.TargetID)
	}
	matchTextFindByIDQuery := &find.FindQuery{
		UseIDs: &trueValue,
		IDs:    &targetIDs,
	}

	lenOfReps := len(findCtx.Repositories.TimeIsReps)

	existErr := false
	wg := &sync.WaitGroup{}
	timeIssCh := make(chan []*reps.TimeIs, lenOfReps)
	errch := make(chan error, lenOfReps)
	defer close(timeIssCh)
	defer close(errch)

	// 並列処理
	for _, rep := range findCtx.Repositories.TimeIsReps {
		wg.Add(1)
		go func(rep reps.TimeIsRepository) {
			defer wg.Done()
			timeiss, err := rep.FindTimeIs(ctx, timeisFindKyouQuery)
			if err != nil {
				errch <- err
				return
			}

			if len(targetIDs) != 0 {
				// textでマッチしたものをID検索
				textMatchTimeiss, err := rep.FindTimeIs(ctx, matchTextFindByIDQuery)
				if err != nil {
					errch <- err
					return
				}
				timeiss = append(timeiss, textMatchTimeiss...)
			}
			timeIssCh <- timeiss
			errch <- nil
		}(rep)
	}
	wg.Wait()
	// エラー集約
	for range lenOfReps {
		e := <-errch
		if e != nil {
			err = fmt.Errorf("error at find timeiss: %w: %w:", e, err)
			existErr = true
		}
	}

	if existErr {
		return nil, err
	}

	// TimeIs集約
	for range lenOfReps {
		matchtimeissInRep := <-timeIssCh
		for _, timeis := range matchtimeissInRep {
			if existtimeis, exist := findCtx.MatchTimeIssAtFindTimeIs[timeis.ID]; exist {
				if timeis.UpdateTime.After(existtimeis.UpdateTime) {
					findCtx.MatchTimeIssAtFindTimeIs[timeis.ID] = timeis
				}
			} else {
				findCtx.MatchTimeIssAtFindTimeIs[timeis.ID] = timeis
			}
			fmt.Printf(" = %+v\n", findCtx.MatchTimeIssAtFindTimeIs[timeis.ID].Title)
		}
	}

	return nil, nil
}

func (f *FindFilter) filterLocationKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if findCtx.ParsedFindQuery.UseMap == nil || !*findCtx.ParsedFindQuery.UseMap {
		return nil, nil
	}

	matchKyous := map[string]*reps.Kyou{}
	const dateLayout = "2006-01-02"
	var err error
	matchGPSLogs := []*reps.GPSLog{}

	// 開始日を取得
	startTime := findCtx.ParsedFindQuery.CalendarStartDate
	endTime := findCtx.ParsedFindQuery.CalendarEndDate

	// radius, latitude, longitudeを取得
	var radius float64
	var latitude float64
	var longitude float64

	if findCtx.ParsedFindQuery.MapRadius != nil {
		radius = *findCtx.ParsedFindQuery.MapRadius
	}
	if findCtx.ParsedFindQuery.MapLatitude != nil {
		latitude = *findCtx.ParsedFindQuery.MapLatitude
	}
	if findCtx.ParsedFindQuery.MapLongitude != nil {
		longitude = *findCtx.ParsedFindQuery.MapLongitude
	}

	// 日付のnil解決
	if startTime != nil && endTime == nil {
		s := time.Time(*startTime)
		e := time.Time(*startTime).Add(time.Hour*23 + time.Minute*59 + time.Second*59)
		startTime = &s
		endTime = &e
	} else if startTime != nil && endTime != nil {
		s := time.Time(*startTime)
		e := time.Time(*endTime).Add(time.Hour*23 + time.Minute*59 + time.Second*59)
		startTime = &s
		endTime = &e
	}
	// GPSLogを取得する
	gpsLogs := []*reps.GPSLog{}
	lenOfReps := 0
	for range findCtx.Repositories.GPSLogReps {
		lenOfReps++
	}

	existErr := false
	wg := &sync.WaitGroup{}
	gpsLogsCh := make(chan []*reps.GPSLog, lenOfReps)
	errch := make(chan error, lenOfReps)
	defer close(gpsLogsCh)
	defer close(errch)

	// text検索用クエリ
	lenOfTexts := 0
	for range findCtx.Repositories.GPSLogReps {
		lenOfTexts++
	}

	// 並列処理
	for _, rep := range findCtx.Repositories.GPSLogReps {
		wg.Add(1)
		go func(rep reps.GPSLogRepository) {
			defer wg.Done()
			// repで検索
			gpsLogs, err := rep.GetGPSLogs(ctx, *startTime, *endTime)
			if err != nil {
				errch <- err
				return
			}
			gpsLogsCh <- gpsLogs
			errch <- nil
		}(rep)
	}
	wg.Wait()
	// エラー集約
	for range lenOfReps {
		e := <-errch
		if e != nil {
			err = fmt.Errorf("error at filter gpslogs: %w: %w:", e, err)
			existErr = true
		}
	}

	if existErr {
		return nil, err
	}

	// GPSLog集約
	for range lenOfReps {
		matchGPSLogsInRep := <-gpsLogsCh
		gpsLogs = append(gpsLogs, matchGPSLogsInRep...)
	}

	return nil, nil

	// 並び替え
	sort.Slice(matchGPSLogs, func(i, j int) bool { return matchGPSLogs[i].RelatedTime.Before(matchGPSLogs[j].RelatedTime) })

	// 該当する時間を出す
	matchGPSLogSetList := [][]*reps.GPSLog{}

	// 10000こずつに分けて並列処理
	splitLimit := 10000
	splitedGPSLogList := [][]*reps.GPSLog{}
	for i := 0; i < len(matchGPSLogs); i += splitLimit {
		minus1 := 0
		if i != 0 {
			minus1 -= 1
		}
		min := i + splitLimit
		if min >= len(matchGPSLogs) {
			min = len(matchGPSLogs)
		}
		splitedGPSLogList = append(splitedGPSLogList, matchGPSLogs[i+minus1:min])
	}

	// 並列処理
	matchGPSLogWg := &sync.WaitGroup{}
	matchGPSLogCh := make(chan [][]*reps.GPSLog, len(splitedGPSLogList))
	errchForGPSLog := make(chan error, len(splitedGPSLogList))
	defer close(matchGPSLogCh)
	defer close(errchForGPSLog)

	for _, splitedGPSLogs := range splitedGPSLogList {
		matchGPSLogWg.Add(1)
		go func(splitedGPSLogs []*reps.GPSLog) {
			defer matchGPSLogWg.Done()
			matchGPSLogs := [][]*reps.GPSLog{}
			preTrue := false // 一つ前の時間でtrueだった
			for i := 0; i < len(splitedGPSLogs); i++ {
				select {
				case <-ctx.Done():
					errchForGPSLog <- ctx.Err()
				default:
					if preTrue {
						matchGPSLogs = append(matchGPSLogs, []*reps.GPSLog{
							splitedGPSLogs[i-1],
							splitedGPSLogs[i],
						})
					}

					if calcDistance(latitude, longitude, splitedGPSLogs[i].Latitude, splitedGPSLogs[i].Longitude) <= radius {
						preTrue = true
					} else {
						preTrue = false
					}
				}
			}
			matchGPSLogCh <- matchGPSLogs
			errch <- nil
		}(splitedGPSLogs)
	}
	matchGPSLogWg.Wait()
	existError := false
	// エラー集約
	for range lenOfReps {
		e := <-errchForGPSLog
		if e != nil {
			err = fmt.Errorf("error at filter location: %w: %w:", e, err)
			existError = true
		}
	}
	if existError {
		return nil, err
	}
	// GPSLog集約
	for range lenOfReps {
		pointsList := <-matchGPSLogCh
		for _, points := range pointsList {
			pointOfStart := points[0]
			pointOfEnd := points[1]
			matchGPSLogSetList = append(matchGPSLogSetList, []*reps.GPSLog{
				pointOfStart,
				pointOfEnd,
			})
		}
	}

	// KyouがLocation内か判定
	workerCount := runtime.NumCPU() * 3
	matchKyouCh := make(chan *reps.Kyou)
	processingKyouCh := make(chan struct{}, workerCount)
	procesFinishKyouCh := make(chan struct{}, workerCount)
	errch2 := make(chan error, len(matchGPSLogSetList))
	calcWg := &sync.WaitGroup{}
	defer close(matchKyouCh)
	defer close(errch2)

	timeMatchJudgementer := func(kyou *reps.Kyou, locationTime1 time.Time, locationTime2 time.Time) {
		defer calcWg.Done()
		defer func() { procesFinishKyouCh <- struct{}{} }()
		if locationTime1.Before(kyou.RelatedTime) && locationTime2.After(kyou.RelatedTime) {
			matchKyouCh <- kyou
		}
	}

	// 全部みおわったか判定
	finishCh := make(chan struct{})
	defer close(finishCh)
	go func() {
	collectLoop:
		for {
			select {
			case <-processingKyouCh:
				select {
				case collectedKyou := <-matchKyouCh:
					matchKyous[collectedKyou.ID] = collectedKyou
					<-procesFinishKyouCh
				case <-procesFinishKyouCh:
				}
			case <-finishCh:
				break collectLoop
			default:
			}
		}
	}()

	// 場所内判定していく
	func() {
		processingKyouCh <- struct{}{}
		defer func() { procesFinishKyouCh <- struct{}{} }()
		calcWg.Add(1)
		defer calcWg.Done()
		for _, gpsLogSet := range matchGPSLogSetList {
			select {
			case <-ctx.Done():
				errch2 <- ctx.Err()
			default:
				for _, kyou := range findCtx.MatchKyousCurrent {
					kyou := kyou
					gpsLogSet := gpsLogSet
					calcWg.Add(1)
					processingKyouCh <- struct{}{}
					go timeMatchJudgementer(kyou, gpsLogSet[0].RelatedTime, gpsLogSet[1].RelatedTime)
				}
			}
		}
	}()
	calcWg.Wait()
	finishCh <- struct{}{}

	// エラー集約
	existError = false
errloop2:
	for {
		select {
		case e := <-errch2:
			err = fmt.Errorf("error at filter location: %w: %w:", e, err)
			existError = true
		default:
			break errloop2
		}
	}
	if existError {
		return nil, err
	}

	findCtx.MatchKyousAtFilterLocation = matchKyous
	findCtx.MatchKyousCurrent = matchKyous
	return nil, nil
}

func (f *FindFilter) sortResultKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	sort.Slice(findCtx.ResultKyous, func(i, j int) bool {
		return findCtx.ResultKyous[i].RelatedTime.After(findCtx.ResultKyous[j].RelatedTime)
	})
	return nil, nil
}

func (f *FindFilter) findTexts(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if findCtx.ParsedFindQuery.UseWords == nil || !(*findCtx.ParsedFindQuery.UseWords) || (len(*findCtx.ParsedFindQuery.Words) == 0 && (len(*findCtx.ParsedFindQuery.NotWords) == 0)) {
	}
	var err error

	lenOfTextReps := 0
	for range findCtx.Repositories.TextReps {
		lenOfTextReps++
	}

	// words, notWordsをパースする
	words := []string{}
	notWords := []string{}
	if findCtx.ParsedFindQuery.Words != nil {
		words = *findCtx.ParsedFindQuery.Words
	}
	if findCtx.ParsedFindQuery.NotWords != nil {
		notWords = *findCtx.ParsedFindQuery.NotWords
	}

	// 対象タグ取得用検索クエリ
	trueValue := true
	falseValue := false

	findTextsQuery := &find.FindQuery{
		IsDeleted: &falseValue,
		UseWords:  &trueValue,
		Words:     &words,
		NotWords:  &notWords,
		WordsAnd:  &falseValue,
	}

	existErr := false
	wg := &sync.WaitGroup{}
	textsCh := make(chan []*reps.Text, lenOfTextReps)
	errch := make(chan error, lenOfTextReps)
	defer close(textsCh)
	defer close(errch)

	// 並列処理
	for _, rep := range findCtx.Repositories.TextReps {
		wg.Add(1)
		go func(textRep reps.TextRepository) {
			defer wg.Done()
			texts, err := textRep.FindTexts(ctx, findTextsQuery)
			if err != nil {
				errch <- err
				return
			}
			textsCh <- texts
			errch <- nil
		}(rep)
	}
	wg.Wait()
	// エラー集約
	for range lenOfTextReps {
		e := <-errch
		if e != nil {
			err = fmt.Errorf("error at find  texts: %w: %w:", e, err)
			existErr = true
		}
	}

	if existErr {
		return nil, err
	}

	// Text集約
	for range lenOfTextReps {
		matchTexts := <-textsCh
		for _, text := range matchTexts {
			if existText, exist := findCtx.MatchTexts[text.ID]; exist {
				if text.UpdateTime.After(existText.UpdateTime) {
					findCtx.MatchTexts[text.ID] = text
				}
			} else {
				findCtx.MatchTexts[text.ID] = text
			}
		}
	}

	return nil, nil
}

func (f *FindFilter) findTimeIsTexts(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	var err error
	if findCtx.ParsedFindQuery.UseTimeIs == nil || !*findCtx.ParsedFindQuery.UseTimeIs {
		return nil, nil
	}

	lenOfTextReps := 0
	for range findCtx.Repositories.TextReps {
		lenOfTextReps++
	}

	// words, notWordsをパースする
	words := []string{}
	notWords := []string{}
	if findCtx.ParsedFindQuery.TimeIsWords != nil {
		words = *findCtx.ParsedFindQuery.TimeIsWords
	}
	if findCtx.ParsedFindQuery.TimeIsNotWords != nil {
		notWords = *findCtx.ParsedFindQuery.TimeIsNotWords
	}

	// 対象タグ取得用検索クエリ
	trueValue := true
	falseValue := false
	findTextsQuery := &find.FindQuery{
		IsDeleted: &falseValue,
		UseWords:  &trueValue,
		Words:     &words,
		NotWords:  &notWords,
		WordsAnd:  &falseValue,
	}

	existErr := false
	wg := &sync.WaitGroup{}
	textsCh := make(chan []*reps.Text, lenOfTextReps)
	errch := make(chan error, lenOfTextReps)
	defer close(textsCh)
	defer close(errch)

	// 並列処理
	for _, rep := range findCtx.Repositories.TextReps {
		wg.Add(1)
		go func(textRep reps.TextRepository) {
			defer wg.Done()
			texts, err := textRep.FindTexts(ctx, findTextsQuery)
			if err != nil {
				errch <- err
				return
			}
			textsCh <- texts
			errch <- nil
		}(rep)
	}
	wg.Wait()
	// エラー集約
	for range lenOfTextReps {
		e := <-errch
		if e != nil {
			err = fmt.Errorf("error at find  texts: %w: %w:", e, err)
			existErr = true
		}
	}

	if existErr {
		return nil, err
	}

	// Text集約
	for range lenOfTextReps {
		matchTexts := <-textsCh
		for _, text := range matchTexts {
			if existText, exist := findCtx.MatchTimeIsTexts[text.ID]; exist {
				if text.UpdateTime.After(existText.UpdateTime) {
					findCtx.MatchTimeIsTexts[text.ID] = text
				}
			} else {
				findCtx.MatchTimeIsTexts[text.ID] = text
			}
		}
	}

	return nil, nil
}

func calcDistance(lat1 float64, lng1 float64, lat2 float64, lng2 float64) float64 {
	lat1 *= R
	lng1 *= R
	lat2 *= R
	lng2 *= R
	return float64(6371.0) * math.Acos(math.Cos(lat1)*math.Cos(lat2)*math.Cos(lng2-lng1)+math.Sin(lat1)*math.Sin(lat2))
}
