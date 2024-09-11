// ˅
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
)

const (
	NoTags = "no tags"
	R      = math.Pi / 180
)

// ˄

type FindFilter struct {
	// ˅

	// ˄

	// ˅

	// ˄
}

func (f *FindFilter) FindKyous(ctx context.Context, userID string, device string, gkillDAOManager *dao.GkillDAOManager, queryJSON string) ([]*reps.Kyou, []*message.GkillError, error) {
	// ˅
	findKyouContext := &FindKyouContext{}

	// QueryをContextに入れる
	// jsonからパースする
	queryMap := map[string]string{}
	err := json.Unmarshal([]byte(queryJSON), &queryMap)
	if err != nil {
		err = fmt.Errorf("error at parse query json at find kyous %s: %w", queryJSON, err)
		return nil, nil, err
	}
	findKyouContext.RawQueryJSON = queryJSON
	findKyouContext.ParsedQuery = queryMap

	// フィルタ
	gkillErr, err := f.getRepositories(ctx, userID, device, gkillDAOManager, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at get repositories: %w", err)
		return nil, gkillErr, err
	}
	gkillErr, err = f.selectMatchRepsFromQuery(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at select match reps: %w", err)
		return nil, gkillErr, err
	}
	gkillErr, err = f.updateCache(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at update cache: %w", err)
		return nil, gkillErr, err
	}
	gkillErr, err = f.parseTagFilterModeFromQuery(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at parse tag filter mode: %w", err)
		return nil, gkillErr, err
	}
	gkillErr, err = f.parseTimeIsTagFilterModeFromQuery(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at parse timeis tag filter mode: %w", err)
		return nil, gkillErr, err
	}
	gkillErr, err = f.getAllTags(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at get all tags: %w", err)
		return nil, gkillErr, err
	}
	gkillErr, err = f.findTags(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at find tags: %w", err)
		return nil, gkillErr, err
	}
	gkillErr, err = f.findTexts(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at find texts: %w", err)
		return nil, gkillErr, err
	}
	gkillErr, err = f.findTimeIsTexts(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at find timeis texts: %w", err)
		return nil, gkillErr, err
	}
	gkillErr, err = f.findTimeIs(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at find timeis: %w", err)
		return nil, gkillErr, err
	}
	gkillErr, err = f.findTimeIsTags(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at find timeis tags: %w", err)
		return nil, gkillErr, err
	}
	gkillErr, err = f.filterTagsTimeIs(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at filter tags timeis: %w", err)
		return nil, gkillErr, err
	}
	gkillErr, err = f.findKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at find kyous: %w", err)
		return nil, gkillErr, err
	}
	gkillErr, err = f.filterTagsKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at filter tags kyous: %w", err)
		return nil, gkillErr, err
	}
	//TODO filterHiddenTag
	gkillErr, err = f.filterPlaingTimeIsKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at filter plaing time is kyous: %w", err)
		return nil, gkillErr, err
	}
	gkillErr, err = f.filterLocationKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at filter location kyous: %w", err)
		return nil, gkillErr, err
	}
	gkillErr, err = f.sortResultKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at sort result kyous: %w", err)
		return nil, gkillErr, err
	}

	return findKyouContext.ResultKyous, nil, nil
	// ˄
}

func (f *FindFilter) getRepositories(ctx context.Context, userID string, device string, gkillDAOManager *dao.GkillDAOManager, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	var err error
	repositories, err := gkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		return nil, err
	}
	findCtx.Repositories = repositories
	return nil, nil
}

func (f *FindFilter) selectMatchRepsFromQuery(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	// ˅
	var err error
	repositories := findCtx.Repositories

	existErr := false
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(repositories.Reps))
	defer close(errch)

	targetRepNames := []string{}
	err = json.Unmarshal([]byte(findCtx.ParsedQuery["reps"]), &targetRepNames)
	if err != nil {
		err = fmt.Errorf("error at parse reps %s: %w", findCtx.ParsedQuery["reps"], err)
		return nil, err
	}

	// 並列処理
	m := &sync.Mutex{}
	for _, rep := range repositories.Reps {
		wg.Add(1)
		rep := rep
		go func(rep reps.Repository) {
			defer wg.Done()
			repName, err := rep.GetPath(ctx, "")
			if err != nil {
				errch <- err
				return
			}

			for _, targetRepName := range targetRepNames {
				if targetRepName == repName {
					if _, exist := findCtx.MatchReps[targetRepName]; !exist {
						m.Lock()
						findCtx.MatchReps[targetRepName] = rep
						m.Unlock()
					}
				}
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
		return nil, err
	}
	return nil, nil
	// ˄
}

func (f *FindFilter) updateCache(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	// ˅
	var err error
	existErr := false
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(findCtx.MatchReps))
	defer close(errch)

	// 並列処理
	for _, rep := range findCtx.MatchReps {
		wg.Add(1)
		rep := rep
		go func(rep reps.Repository) {
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
		return nil, err
	}
	return nil, nil
	// ˄
}

func (f *FindFilter) getAllTags(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	// ˅
	var err error

	lenOfTagReps := 0
	for _ = range findCtx.Repositories.TagReps {
		lenOfTagReps++
	}

	// 全タグ取得用検索クエリ
	findTagsQueryJSON := ""
	findTagsQueryJSON += "{\n"
	findTagsQueryJSON += "  is_deleted: false\n"
	findTagsQueryJSON += "}"

	existErr := false
	wg := &sync.WaitGroup{}
	tagsCh := make(chan []*reps.Tag, lenOfTagReps)
	errch := make(chan error, lenOfTagReps)
	defer close(tagsCh)
	defer close(errch)

	// 並列処理
	for _, rep := range findCtx.Repositories.TagReps {
		wg.Add(1)
		rep := rep
		go func(tagRep reps.TagRepository) {
			defer wg.Done()
			tags, err := tagRep.FindTags(ctx, findTagsQueryJSON)
			if err != nil {
				errch <- err
				return
			}
			tagsCh <- tags
		}(rep)
	}
	wg.Wait()
	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get all tags: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	// Tag集約
loop:
	for {
		select {
		case matchTags := <-tagsCh:
			if matchTags == nil {
				continue loop
			}
			for _, tag := range matchTags {
				if existTag, exist := findCtx.AllTags[tag.ID]; exist {
					if tag.UpdateTime.Before(existTag.UpdateTime) {
						findCtx.AllTags[tag.ID] = tag
					}
				} else {
					findCtx.AllTags[tag.ID] = tag
				}
			}
		default:
			break loop
		}
	}

	if existErr {
		return nil, err
	}
	return nil, nil
	// ˄
}

func (f *FindFilter) findTimeIsTags(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	// ˅
	var err error

	lenOfTagReps := 0
	for _ = range findCtx.Repositories.TagReps {
		lenOfTagReps++
	}

	// クエリのタグをParse
	tags := []string{}
	err = json.Unmarshal([]byte(findCtx.ParsedQuery["time_is_tags"]), &tags)
	if err != nil {
		err = fmt.Errorf("error at parse tags %s: %w", findCtx.ParsedQuery["tags"], err)
		return nil, err
	}

	// 対象タグ取得用検索クエリ
	findTagsQueryJSON := ""
	findTagsQueryJSON += "{\n"
	findTagsQueryJSON += "  is_deleted: false,\n"
	findTagsQueryJSON += "  use_word: true,\n"
	findTagsQueryJSON += "  words: [\n"
	for i, tag := range tags {
		findTagsQueryJSON += `    "` + tag + `"`
		if i != len(tags)-1 {
			findTagsQueryJSON += ",\n"
		} else {
			findTagsQueryJSON += "\n"
		}
	}
	findTagsQueryJSON += "  ],\n"
	findTagsQueryJSON += "  words_and: false\n"
	findTagsQueryJSON += "}"

	existErr := false
	wg := &sync.WaitGroup{}
	tagsCh := make(chan []*reps.Tag, lenOfTagReps)
	errch := make(chan error, lenOfTagReps)
	defer close(tagsCh)
	defer close(errch)

	// 並列処理
	for _, rep := range findCtx.Repositories.TagReps {
		wg.Add(1)
		rep := rep
		go func(tagRep reps.TagRepository) {
			defer wg.Done()
			tags, err := tagRep.FindTags(ctx, findTagsQueryJSON)
			if err != nil {
				errch <- err
				return
			}
			tagsCh <- tags
		}(rep)
	}
	wg.Wait()
	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find timeis tags: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	// TimeIsのTag集約
loop:
	for {
		select {
		case matchTags := <-tagsCh:
			if matchTags == nil {
				continue loop
			}
			for _, tag := range matchTags {
				if existTag, exist := findCtx.MatchTimeIsTags[tag.ID]; exist {
					if tag.UpdateTime.Before(existTag.UpdateTime) {
						findCtx.MatchTimeIsTags[tag.ID] = tag
					}
				} else {
					findCtx.MatchTimeIsTags[tag.ID] = tag
				}
			}
		default:
			break loop
		}
	}

	if existErr {
		return nil, err
	}

	return nil, nil
	// ˄
}

func (f *FindFilter) findTags(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	// ˅
	var err error

	lenOfTagReps := 0
	for _ = range findCtx.Repositories.TagReps {
		lenOfTagReps++
	}

	// クエリのタグをParse
	tags := []string{}
	err = json.Unmarshal([]byte(findCtx.ParsedQuery["time_is_tags"]), &tags)
	if err != nil {
		err = fmt.Errorf("error at parse tags %s: %w", findCtx.ParsedQuery["tags"], err)
		return nil, err
	}

	// 対象タグ取得用検索クエリ
	findTagsQueryJSON := ""
	findTagsQueryJSON += "{\n"
	findTagsQueryJSON += "  is_deleted: false,\n"
	findTagsQueryJSON += "  use_word: true,\n"
	findTagsQueryJSON += "  words: [\n"
	for i, tag := range tags {
		findTagsQueryJSON += `    "` + tag + `"`
		if i != len(tags)-1 {
			findTagsQueryJSON += ",\n"
		} else {
			findTagsQueryJSON += "\n"
		}
	}
	findTagsQueryJSON += "  ],\n"
	findTagsQueryJSON += "  words_and: false\n"
	findTagsQueryJSON += "}"

	existErr := false
	wg := &sync.WaitGroup{}
	tagsCh := make(chan []*reps.Tag, lenOfTagReps)
	errch := make(chan error, lenOfTagReps)
	defer close(tagsCh)
	defer close(errch)

	// 並列処理
	for _, rep := range findCtx.Repositories.TagReps {
		wg.Add(1)
		rep := rep
		go func(tagRep reps.TagRepository) {
			defer wg.Done()
			tags, err := tagRep.FindTags(ctx, findTagsQueryJSON)
			if err != nil {
				errch <- err
				return
			}
			tagsCh <- tags
		}(rep)
	}
	wg.Wait()
	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find  tags: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	// Tag集約
loop:
	for {
		select {
		case matchTags := <-tagsCh:
			if matchTags == nil {
				continue loop
			}
			for _, tag := range matchTags {
				if existTag, exist := findCtx.MatchTags[tag.ID]; exist {
					if tag.UpdateTime.Before(existTag.UpdateTime) {
						findCtx.MatchTags[tag.ID] = tag
					}
				} else {
					findCtx.MatchTags[tag.ID] = tag
				}
			}
		default:
			break loop
		}
	}

	if existErr {
		return nil, err
	}

	return nil, nil
	// ˄
}

func (f *FindFilter) parseTagFilterModeFromQuery(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	// ˅
	tagFilterModeIsAnd := false
	tagFilterModeIsAndStr := findCtx.ParsedQuery["tags_and"]
	err := json.Unmarshal([]byte(tagFilterModeIsAndStr), &tagFilterModeIsAnd)
	if err != nil {
		err = fmt.Errorf("error at parse tag filter mode %s: %w", tagFilterModeIsAndStr, err)
		return nil, err
	}
	if tagFilterModeIsAnd {
		var tagFilterMode TagFilterMode
		tagFilterMode = Or
		findCtx.TagFilterMode = &tagFilterMode
	} else {
		var tagFilterMode TagFilterMode
		tagFilterMode = And
		findCtx.TagFilterMode = &tagFilterMode
	}
	return nil, nil
	// ˄
}

func (f *FindFilter) parseTimeIsTagFilterModeFromQuery(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	// ˅
	timeisTagFilterModeIsAnd := false
	timeisTagFilterModeIsAndStr := findCtx.ParsedQuery["tags_and"]
	err := json.Unmarshal([]byte(timeisTagFilterModeIsAndStr), &timeisTagFilterModeIsAnd)
	if err != nil {
		err = fmt.Errorf("error at parse tag filter mode %s: %w", timeisTagFilterModeIsAndStr, err)
		return nil, err
	}
	if timeisTagFilterModeIsAnd {
		var timeisTagFilterMode TagFilterMode
		timeisTagFilterMode = Or
		findCtx.TagFilterMode = &timeisTagFilterMode
	} else {
		var timeisTagFilterMode TagFilterMode
		timeisTagFilterMode = And
		findCtx.TagFilterMode = &timeisTagFilterMode
	}
	return nil, nil
	// ˄
}

func (f *FindFilter) findKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	// ˅
	var err error

	lenOfReps := 0
	for _ = range findCtx.MatchReps {
		lenOfReps++
	}

	existErr := false
	wg := &sync.WaitGroup{}
	kyousCh := make(chan []*reps.Kyou, lenOfReps)
	errch := make(chan error, lenOfReps)
	defer close(kyousCh)
	defer close(errch)

	// text検索用クエリ
	lenOfTexts := 0
	for _ = range findCtx.MatchTexts {
		lenOfTexts++
	}
	matchTextFindByIDQueryJSON := ""
	matchTextFindByIDQueryJSON += "{"
	matchTextFindByIDQueryJSON += "  is_deleted: false,\n"
	matchTextFindByIDQueryJSON += "  use_ids: true,\n"
	matchTextFindByIDQueryJSON += "  ids: ["
	i := 0
	for _, text := range findCtx.MatchTexts {
		matchTextFindByIDQueryJSON += `"` + text.TargetID + `"`
		if i != lenOfTexts-1 {
			matchTextFindByIDQueryJSON += ",\n"
		} else {
			matchTextFindByIDQueryJSON += "\n"
		}
		i++
	}
	matchTextFindByIDQueryJSON += "],\n"
	matchTextFindByIDQueryJSON += "}"

	// 並列処理
	for _, rep := range findCtx.MatchReps {
		wg.Add(1)
		rep := rep
		go func(rep reps.Repository) {
			defer wg.Done()
			// repで検索
			kyous, err := rep.FindKyous(ctx, findCtx.RawQueryJSON)
			if err != nil {
				errch <- err
				return
			}

			// textでマッチしたものをID検索
			textMatchKyous, err := rep.FindKyous(ctx, matchTextFindByIDQueryJSON)
			if err != nil {
				errch <- err
				return
			}

			kyousCh <- append(kyous, textMatchKyous...)
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
	// Kyou集約
loop:
	for {
		select {
		case matchKyousInRep := <-kyousCh:
			if matchKyousInRep == nil {
				continue loop
			}
			for _, kyou := range matchKyousInRep {
				if existKyou, exist := findCtx.MatchKyousAtFindKyou[kyou.ID]; exist {
					if kyou.UpdateTime.Before(existKyou.UpdateTime) {
						findCtx.MatchKyousAtFindKyou[kyou.ID] = kyou
					}
				} else {
					findCtx.MatchKyousAtFindKyou[kyou.ID] = kyou
				}
			}
		default:
			break loop
		}
	}

	if existErr {
		return nil, err
	}

	findCtx.MatchKyousCurrent = findCtx.MatchKyousAtFindKyou
	return nil, nil
	// ˄
}

func (f *FindFilter) filterTagsKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	// ˅
	if *findCtx.TagFilterMode == Or {
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
					if kyou.UpdateTime.Before(existKyou.UpdateTime) {
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
		err := json.Unmarshal([]byte(findCtx.ParsedQuery["tags"]), &tags)
		if err != nil {
			err = fmt.Errorf("error at parse tags %s: %w", findCtx.ParsedQuery["tags"], err)
			return nil, err
		}

		for _, tag := range tags {
			if tag == NoTags {
				existNoTags = true
				break
			}
		}

		// タグフィルタしたものをCtxに収める
		for _, kyou := range matchOrTagKyous {
			if existKyou, exist := findCtx.MatchKyousAtFindKyou[kyou.ID]; exist {
				if kyou.UpdateTime.Before(existKyou.UpdateTime) {
					findCtx.MatchKyousAtFilterTags[kyou.ID] = kyou
				}
			} else {
				findCtx.MatchKyousAtFilterTags[kyou.ID] = kyou
			}
		}
		if existNoTags {
			for _, kyou := range noTagKyous {
				if existKyou, exist := findCtx.MatchKyousAtFindKyou[kyou.ID]; exist {
					if kyou.UpdateTime.Before(existKyou.UpdateTime) {
						findCtx.MatchKyousAtFilterTags[kyou.ID] = kyou
					}
				} else {
					findCtx.MatchKyousAtFilterTags[kyou.ID] = kyou
				}
			}
		}
		findCtx.MatchKyousCurrent = findCtx.MatchKyousAtFilterTags
	} else if *findCtx.TagFilterMode == And {
		// ANDの場合のフィルタリング処理

		tagNameMap := map[string]map[string]*reps.Kyou{} // map[タグ名][kyou.ID（tagTargetID）] = reps.kyou

		for _, kyou := range findCtx.MatchKyousCurrent {
			for _, tag := range findCtx.MatchTags {
				if kyou.ID == tag.TargetID {
					if existKyou, exist := tagNameMap[tag.Tag][kyou.ID]; exist {
						if kyou.UpdateTime.Before(existKyou.UpdateTime) {
							tagNameMap[tag.Tag][kyou.ID] = kyou
						}
					} else {
						tagNameMap[tag.Tag][kyou.ID] = kyou
					}
				}
			}
		}

		// タグ無しの情報もtagNameMapにいれる
		for _, kyou := range findCtx.MatchKyousCurrent {
			relatedTagKyou := false
			for _, tag := range findCtx.AllTags {
				if kyou.ID == tag.TargetID {
					relatedTagKyou = true
				}
			}
			if !relatedTagKyou {
				if existKyou, exist := tagNameMap[NoTags][kyou.ID]; exist {
					if kyou.UpdateTime.Before(existKyou.UpdateTime) {
						tagNameMap[NoTags][kyou.ID] = kyou
					}
				} else {
					tagNameMap[NoTags][kyou.ID] = kyou
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
						if kyou.UpdateTime.Before(existKyou.UpdateTime) {
							hasAllMatchTagsKyousMap[kyou.ID] = kyou
						}
					} else {
						hasAllMatchTagsKyousMap[kyou.ID] = kyou
					}
				}
			default:
				// 初回ループ以外は、
				// 以前のタグにマッチしたもの（hasAllMatchTagsKyous）にあり、かつ
				// 今回のタグにマッチしたもの　をいれる。
				matchThisLoopKyousMap := map[string]*reps.Kyou{}
				for _, hasAllMatchTagsKyou := range hasAllMatchTagsKyousMap {
					for kyouID, kyou := range kyouIDMap {
						if hasAllMatchTagsKyou.ID == kyouID {
							if existKyou, exist := matchThisLoopKyousMap[kyou.ID]; exist {
								if kyou.UpdateTime.Before(existKyou.UpdateTime) {
									matchThisLoopKyousMap[kyou.ID] = kyou
								}
							} else {
								matchThisLoopKyousMap[kyou.ID] = kyou
							}
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
	// ˄
}

func (f *FindFilter) filterTagsTimeIs(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	// ˅
	if *findCtx.TimeIsTagFilterMode == Or {
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
					if timeis.UpdateTime.Before(existTimeIs.UpdateTime) {
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
		err := json.Unmarshal([]byte(findCtx.ParsedQuery["timeis_tags"]), &tags)
		if err != nil {
			err = fmt.Errorf("error at parse tags %s: %w", findCtx.ParsedQuery["timeis_tags"], err)
			return nil, err
		}

		for _, tag := range tags {
			if tag == NoTags {
				existNoTags = true
				break
			}
		}

		// タグフィルタしたものをCtxに収める
		for _, timeis := range matchOrTagTimeIss {
			if existTimeIs, exist := findCtx.MatchTimeIssAtFindTimeIs[timeis.ID]; exist {
				if timeis.UpdateTime.Before(existTimeIs.UpdateTime) {
					findCtx.MatchTimeIssAtFilterTags[timeis.ID] = timeis
				}
			} else {
				findCtx.MatchTimeIssAtFilterTags[timeis.ID] = timeis
			}
		}
		if existNoTags {
			for _, timeis := range noTagTimeIss {
				if existTimeIs, exist := findCtx.MatchTimeIssAtFindTimeIs[timeis.ID]; exist {
					if timeis.UpdateTime.Before(existTimeIs.UpdateTime) {
						findCtx.MatchTimeIssAtFilterTags[timeis.ID] = timeis
					}
				} else {
					findCtx.MatchTimeIssAtFilterTags[timeis.ID] = timeis
				}
			}
		}
	} else if *findCtx.TimeIsTagFilterMode == And {
		// ANDの場合のフィルタリング処理

		tagNameMap := map[string]map[string]*reps.TimeIs{} // map[タグ名][kyou.ID（tagTargetID）] = reps.TimeIs

		for _, timeis := range findCtx.MatchTimeIssAtFindTimeIs {
			for _, tag := range findCtx.MatchTimeIsTags {
				if timeis.ID == tag.TargetID {
					if existTimeIs, exist := tagNameMap[tag.Tag][timeis.ID]; exist {
						if timeis.UpdateTime.Before(existTimeIs.UpdateTime) {
							tagNameMap[tag.Tag][timeis.ID] = timeis
						}
					} else {
						tagNameMap[tag.Tag][timeis.ID] = timeis
					}
				}
			}
		}

		// タグ無しの情報もtagNameMapにいれる
		for _, timeis := range findCtx.MatchTimeIssAtFindTimeIs {
			relatedTagTimeIs := false
			for _, tag := range findCtx.AllTags {
				if timeis.ID == tag.TargetID {
					relatedTagTimeIs = true
				}
			}
			if !relatedTagTimeIs {
				if existTimeIs, exist := tagNameMap[NoTags][timeis.ID]; exist {
					if timeis.UpdateTime.Before(existTimeIs.UpdateTime) {
						tagNameMap[NoTags][timeis.ID] = timeis
					}
				} else {
					tagNameMap[NoTags][timeis.ID] = timeis
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
						if timeis.UpdateTime.Before(existTimeIs.UpdateTime) {
							hasAllMatchTagsTimeIssMap[timeis.ID] = timeis
						}
					} else {
						hasAllMatchTagsTimeIssMap[timeis.ID] = timeis
					}
				}
			default:
				// 初回ループ以外は、
				// 以前のタグにマッチしたもの（hasAllMatchTagsKyous）にあり、かつ
				// 今回のタグにマッチしたもの　をいれる。
				matchThisLoopTimeIssMap := map[string]*reps.TimeIs{}
				for _, hasAllMatchTagsTimeIs := range hasAllMatchTagsTimeIssMap {
					for timeisID, timeis := range timeisIDMap {
						if hasAllMatchTagsTimeIs.ID == timeisID {
							if existTimeIs, exist := matchThisLoopTimeIssMap[timeis.ID]; exist {
								if timeis.UpdateTime.Before(existTimeIs.UpdateTime) {
									matchThisLoopTimeIssMap[timeis.ID] = timeis
								}
							} else {
								matchThisLoopTimeIssMap[timeis.ID] = timeis
							}
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
	// ˄
}

func (f *FindFilter) filterPlaingTimeIsKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	for _, timeis := range findCtx.MatchTimeIssAtFilterTags {
		for _, kyou := range findCtx.MatchKyousCurrent {
			if (timeis.EndTime != nil && kyou.RelatedTime.After(timeis.StartTime) && kyou.RelatedTime.Before(*timeis.EndTime)) || (timeis.EndTime == nil && kyou.RelatedTime.After(timeis.StartTime)) {
				if existKyou, exist := findCtx.MatchKyousAtFilterTimeIs[kyou.ID]; exist {
					if kyou.UpdateTime.Before(existKyou.UpdateTime) {
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
	// ˅

	// 対象TimeIs取得用検索クエリ
	findTagsQueryJSON := ""
	findTagsQueryJSON += "{\n"
	findTagsQueryJSON += "  is_deleted: false,\n"
	findTagsQueryJSON += "  use_word: true,\n"
	findTagsQueryJSON += "  words: "
	findTagsQueryJSON += findCtx.ParsedQuery["time_is_word"]
	findTagsQueryJSON += ",\n"
	findTagsQueryJSON += "  not_words: "
	findTagsQueryJSON += findCtx.ParsedQuery["time_is_not_word"]
	findTagsQueryJSON += ",\n"
	findTagsQueryJSON += "  words_and: "
	findTagsQueryJSON += findCtx.ParsedQuery["time_is_word_and"]
	findTagsQueryJSON += "\n"
	findTagsQueryJSON += "}"

	// text検索用クエリ
	lenOfTexts := 0
	for _ = range findCtx.MatchTimeIsTexts {
		lenOfTexts++
	}
	matchTextFindByIDQueryJSON := ""
	matchTextFindByIDQueryJSON += "{"
	matchTextFindByIDQueryJSON += "  is_deleted: false,\n"
	matchTextFindByIDQueryJSON += "  use_ids: true,\n"
	matchTextFindByIDQueryJSON += "  ids: ["
	i := 0
	for _, text := range findCtx.MatchTimeIsTexts {
		matchTextFindByIDQueryJSON += `"` + text.TargetID + `"`
		if i != lenOfTexts-1 {
			matchTextFindByIDQueryJSON += ",\n"
		} else {
			matchTextFindByIDQueryJSON += "\n"
		}
		i++
	}
	matchTextFindByIDQueryJSON += "],\n"
	matchTextFindByIDQueryJSON += "}"

	var err error

	lenOfReps := 0
	for _ = range findCtx.Repositories.TimeIsReps {
		lenOfReps++
	}

	existErr := false
	wg := &sync.WaitGroup{}
	timeIssCh := make(chan []*reps.TimeIs, lenOfReps)
	errch := make(chan error, lenOfReps)
	defer close(timeIssCh)
	defer close(errch)

	// 並列処理
	for _, rep := range findCtx.Repositories.TimeIsReps {
		wg.Add(1)
		rep := rep
		go func(rep reps.TimeIsRepository) {
			defer wg.Done()
			timeiss, err := rep.FindTimeIs(ctx, findCtx.RawQueryJSON)
			if err != nil {
				errch <- err
				return
			}

			// textでマッチしたものをID検索
			textMatchTimeiss, err := rep.FindTimeIs(ctx, matchTextFindByIDQueryJSON)
			if err != nil {
				errch <- err
				return
			}
			timeIssCh <- append(timeiss, textMatchTimeiss...)
		}(rep)
	}
	wg.Wait()
	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find timeiss: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	// TimeIs集約
loop:
	for {
		select {
		case matchtimeissInRep := <-timeIssCh:
			if matchtimeissInRep == nil {
				continue loop
			}
			for _, timeis := range matchtimeissInRep {
				if existtimeis, exist := findCtx.MatchTimeIssAtFindTimeIs[timeis.ID]; exist {
					if timeis.UpdateTime.Before(existtimeis.UpdateTime) {
						findCtx.MatchTimeIssAtFindTimeIs[timeis.ID] = timeis
					}
				} else {
					findCtx.MatchTimeIssAtFindTimeIs[timeis.ID] = timeis
				}
			}
		default:
			break loop
		}
	}

	if existErr {
		return nil, err
	}
	return nil, nil
	// ˄
}

func (f *FindFilter) filterLocationKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	// ˅
	matchKyous := map[string]*reps.Kyou{}
	const dateLayout = "2006-01-02"
	var err error
	matchGPSLogs := []*reps.GPSLog{}

	// 開始日を取得
	var startTime *time.Time
	var startTimeStr string
	err = json.Unmarshal([]byte(findCtx.ParsedQuery["calendar_start_time"]), &startTimeStr)
	if err != nil {
		err = fmt.Errorf("error at parse calendar_start_time %s: %w", findCtx.ParsedQuery["calendar_start_time"], err)
		return nil, err
	}
	startTimeValue, err := time.Parse(sqlite3impl.TimeLayout, startTimeStr)
	if err == nil {
		startTimeValue, err = time.Parse(dateLayout, startTimeValue.Format(dateLayout))
		if err == nil {
			startTime = &startTimeValue
		}
	}

	// 終了日を取得
	var endTime *time.Time
	var endTimeStr string
	err = json.Unmarshal([]byte(findCtx.ParsedQuery["calendar_end_time"]), &endTimeStr)
	if err != nil {
		err = fmt.Errorf("error at parse calendar_end_time %s: %w", findCtx.ParsedQuery["calendar_end_time"], err)
		return nil, err
	}
	endTimeValue, _ := time.Parse(sqlite3impl.TimeLayout, endTimeStr)
	if err == nil {
		endTimeValue, err = time.Parse(dateLayout, endTimeValue.Format(dateLayout))
		if err == nil {
			endTime = &endTimeValue
		}
	}

	// radius, latitude, longitudeを取得
	var radius float64
	err = json.Unmarshal([]byte(findCtx.ParsedQuery["map_radius"]), &radius)
	if err != nil {
		err = fmt.Errorf("error at parse map_radius %s: %w", findCtx.ParsedQuery["map_radius"], err)
		return nil, err
	}
	var latitude float64
	err = json.Unmarshal([]byte(findCtx.ParsedQuery["map_latitude"]), &latitude)
	if err != nil {
		err = fmt.Errorf("error at parse map_latitude %s: %w", findCtx.ParsedQuery["map_latitude"], err)
		return nil, err
	}
	var longitude float64
	err = json.Unmarshal([]byte(findCtx.ParsedQuery["map_longitude"]), &longitude)
	if err != nil {
		err = fmt.Errorf("error at parse map_longitude %s: %w", findCtx.ParsedQuery["map_longitude"], err)
		return nil, err
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
	for _ = range findCtx.Repositories.GPSLogReps {
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
	for _ = range findCtx.Repositories.GPSLogReps {
		lenOfTexts++
	}

	// 並列処理
	for _, rep := range findCtx.Repositories.GPSLogReps {
		wg.Add(1)
		rep := rep
		go func(rep reps.GPSLogRepository) {
			defer wg.Done()
			// repで検索
			gpsLogs, err := rep.GetGPSLogs(ctx, *startTime, *endTime)
			if err != nil {
				errch <- err
				return
			}

			gpsLogsCh <- gpsLogs
		}(rep)
	}
	wg.Wait()
	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at filter gpslogs: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	// GPSLog集約
loop:
	for {
		select {
		case matchGPSLogsInRep := <-gpsLogsCh:
			if matchGPSLogsInRep == nil {
				continue loop
			}
			gpsLogs = append(gpsLogs, matchGPSLogsInRep...)
		default:
			break loop
		}
	}

	if existErr {
		return nil, err
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
		}(splitedGPSLogs)
	}
	matchGPSLogWg.Wait()
	existError := false
	// エラー集約
errloopForGPSLog:
	for {
		select {
		case e := <-errchForGPSLog:
			err = fmt.Errorf("error at filter location: %w", e)
			existError = true
		default:
			break errloopForGPSLog
		}
	}
	if existError {
		return nil, err
	}
	// GPSLog集約
loopForTime:
	for {
		select {
		case pointsList := <-matchGPSLogCh:
			for _, points := range pointsList {
				pointOfStart := points[0]
				pointOfEnd := points[1]
				matchGPSLogSetList = append(matchGPSLogSetList, []*reps.GPSLog{
					pointOfStart,
					pointOfEnd,
				})
			}
		default:
			break loopForTime
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
			err = fmt.Errorf("error at filter location: %w", e)
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
	// ˄
}

func (f *FindFilter) sortResultKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	// ˅
	sort.Slice(findCtx.ResultKyous, func(i, j int) bool {
		return findCtx.ResultKyous[i].RelatedTime.After(findCtx.ResultKyous[j].RelatedTime)
	})
	return nil, nil
	// ˄
}

func (f *FindFilter) findTexts(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	// ˅
	var err error

	lenOfTextReps := 0
	for _ = range findCtx.Repositories.TextReps {
		lenOfTextReps++
	}

	// words, notWordsをパースする
	words := []string{}
	err = json.Unmarshal([]byte(findCtx.ParsedQuery["words"]), &words)
	if err != nil {
		err = fmt.Errorf("error at parse query word %s: %w", findCtx.ParsedQuery["words"], err)
		return nil, err
	}
	notWords := []string{}
	err = json.Unmarshal([]byte(findCtx.ParsedQuery["not_words"]), &words)
	if err != nil {
		err = fmt.Errorf("error at parse query not word %s: %w", findCtx.ParsedQuery["not_words"], err)
		return nil, err
	}

	// 対象タグ取得用検索クエリ
	findTextsQueryJSON := ""
	findTextsQueryJSON += "{\n"
	findTextsQueryJSON += "  is_deleted: false,\n"
	findTextsQueryJSON += "  use_word: true,\n"
	findTextsQueryJSON += "  words: [\n"
	for i, word := range words {
		findTextsQueryJSON += `    "` + word + `"`
		if i != len(words)-1 {
			findTextsQueryJSON += ",\n"
		} else {
			findTextsQueryJSON += "\n"
		}
	}
	findTextsQueryJSON += "  ],\n"
	findTextsQueryJSON += "  not_words: [\n"
	for i, notWord := range notWords {
		findTextsQueryJSON += `    "` + notWord + `"`
		if i != len(words)-1 {
			findTextsQueryJSON += ",\n"
		} else {
			findTextsQueryJSON += "\n"
		}
	}
	findTextsQueryJSON += "  ],\n"

	findTextsQueryJSON += "  words_and: false\n"
	findTextsQueryJSON += "}"

	existErr := false
	wg := &sync.WaitGroup{}
	textsCh := make(chan []*reps.Text, lenOfTextReps)
	errch := make(chan error, lenOfTextReps)
	defer close(textsCh)
	defer close(errch)

	// 並列処理
	for _, rep := range findCtx.Repositories.TextReps {
		wg.Add(1)
		rep := rep
		go func(textRep reps.TextRepository) {
			defer wg.Done()
			texts, err := textRep.FindTexts(ctx, findTextsQueryJSON)
			if err != nil {
				errch <- err
				return
			}
			textsCh <- texts
		}(rep)
	}
	wg.Wait()
	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find  texts: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	// Text集約
loop:
	for {
		select {
		case matchTexts := <-textsCh:
			if matchTexts == nil {
				continue loop
			}
			for _, text := range matchTexts {
				if existText, exist := findCtx.MatchTexts[text.ID]; exist {
					if text.UpdateTime.Before(existText.UpdateTime) {
						findCtx.MatchTexts[text.ID] = text
					}
				} else {
					findCtx.MatchTexts[text.ID] = text
				}
			}
		default:
			break loop
		}
	}

	if existErr {
		return nil, err
	}

	return nil, nil
	// ˄
}

func (f *FindFilter) findTimeIsTexts(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	// ˅
	var err error

	lenOfTextReps := 0
	for _ = range findCtx.Repositories.TextReps {
		lenOfTextReps++
	}

	// words, notWordsをパースする
	words := []string{}
	err = json.Unmarshal([]byte(findCtx.ParsedQuery["timeis_word"]), &words)
	if err != nil {
		err = fmt.Errorf("error at parse query word %s: %w", findCtx.ParsedQuery["timeis_word"], err)
		return nil, err
	}
	notWords := []string{}
	err = json.Unmarshal([]byte(findCtx.ParsedQuery["timeis_not_word"]), &words)
	if err != nil {
		err = fmt.Errorf("error at parse query not word %s: %w", findCtx.ParsedQuery["timeis_not_word"], err)
		return nil, err
	}

	// 対象タグ取得用検索クエリ
	findTextsQueryJSON := ""
	findTextsQueryJSON += "{\n"
	findTextsQueryJSON += "  is_deleted: false,\n"
	findTextsQueryJSON += "  use_word: true,\n"
	findTextsQueryJSON += "  words: [\n"
	for i, word := range words {
		findTextsQueryJSON += `    "` + word + `"`
		if i != len(words)-1 {
			findTextsQueryJSON += ",\n"
		} else {
			findTextsQueryJSON += "\n"
		}
	}
	findTextsQueryJSON += "  ],\n"
	findTextsQueryJSON += "  not_words: [\n"
	for i, notWord := range notWords {
		findTextsQueryJSON += `    "` + notWord + `"`
		if i != len(words)-1 {
			findTextsQueryJSON += ",\n"
		} else {
			findTextsQueryJSON += "\n"
		}
	}
	findTextsQueryJSON += "  ],\n"

	findTextsQueryJSON += "  words_and: false\n"
	findTextsQueryJSON += "}"

	existErr := false
	wg := &sync.WaitGroup{}
	textsCh := make(chan []*reps.Text, lenOfTextReps)
	errch := make(chan error, lenOfTextReps)
	defer close(textsCh)
	defer close(errch)

	// 並列処理
	for _, rep := range findCtx.Repositories.TextReps {
		wg.Add(1)
		rep := rep
		go func(textRep reps.TextRepository) {
			defer wg.Done()
			texts, err := textRep.FindTexts(ctx, findTextsQueryJSON)
			if err != nil {
				errch <- err
				return
			}
			textsCh <- texts
		}(rep)
	}
	wg.Wait()
	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find  texts: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	// Text集約
loop:
	for {
		select {
		case matchTexts := <-textsCh:
			if matchTexts == nil {
				continue loop
			}
			for _, text := range matchTexts {
				if existText, exist := findCtx.MatchTimeIsTexts[text.ID]; exist {
					if text.UpdateTime.Before(existText.UpdateTime) {
						findCtx.MatchTimeIsTexts[text.ID] = text
					}
				} else {
					findCtx.MatchTimeIsTexts[text.ID] = text
				}
			}
		default:
			break loop
		}
	}

	if existErr {
		return nil, err
	}

	return nil, nil
	// ˄
}

func calcDistance(lat1 float64, lng1 float64, lat2 float64, lng2 float64) float64 {
	lat1 *= R
	lng1 *= R
	lat2 *= R
	lng2 *= R
	return float64(6371.0) * math.Acos(math.Cos(lat1)*math.Cos(lat2)*math.Cos(lng2-lng1)+math.Sin(lat1)*math.Sin(lat2))
}

// ˅

// ˄
