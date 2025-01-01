package api

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao"
	"github.com/mt3hr/gkill/src/app/gkill/dao/account_state"
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
	findKyouContext.MatchKyousAtFilterMi = map[string]*reps.Kyou{}
	findKyouContext.MatchKyousAtFilterTags = map[string]*reps.Kyou{}
	findKyouContext.MatchKyousAtFilterTimeIs = map[string]*reps.Kyou{}
	findKyouContext.MatchKyousAtFilterLocation = map[string]*reps.Kyou{}
	findKyouContext.MatchKyousAtFilterImage = map[string]*reps.Kyou{}

	// ユーザのRep取得
	gkillErr, err := f.getRepositories(ctx, userID, device, gkillDAOManager, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at get repositories: %w", err)
		return nil, gkillErr, err
	}

	// フィルタ
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

	// キャッシュ更新終わったら、
	// 最新のKyouがどのRepにあるかを取得
	latestDatasWg := &sync.WaitGroup{}
	latestDatasCh := make(chan []*account_state.LatestDataRepositoryAddress, 1)
	errCh := make(chan error, 1)
	defer close(latestDatasCh)
	defer close(errCh)
	latestDatasWg.Add(1)
	go func() {
		defer latestDatasWg.Done()
		latestDatas, err := findKyouContext.Repositories.LatestDataRepositoryAddressDAO.GetAllLatestDataRepositoryAddresses(ctx)
		if err != nil {
			latestDatasCh <- nil
			errCh <- err
		}
		latestDatasCh <- latestDatas
		errCh <- nil
	}()

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
	gkillErr, err = f.filterMiForMi(ctx, findKyouContext) //miの場合のみ
	if err != nil {
		err = fmt.Errorf("error at filter mi for mi: %w", err)
		return nil, gkillErr, err
	}
	gkillErr, err = f.filterTagsKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at filter tags kyous: %w", err)
		return nil, gkillErr, err
	}
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
	gkillErr, err = f.filterImageKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at filter image kyous: %w", err)
		return nil, gkillErr, err
	}

	latestDatasWg.Wait()
	err = <-errCh
	if err != nil {
		return nil, nil, err
	}
	gkillErr, err = f.replaceLatestKyouInfos(ctx, findKyouContext, <-latestDatasCh)
	if err != nil {
		err = fmt.Errorf("error at replace latest kyou infos: %w", err)
		return nil, gkillErr, err
	}

	for _, rep := range findKyouContext.MatchKyousCurrent {
		findKyouContext.ResultKyous = append(findKyouContext.ResultKyous, rep)
	}

	gkillErr, err = f.overrideKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at override kyous: %w", err)
		return nil, gkillErr, err
	}

	gkillErr, err = f.sortResultKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at sort result kyous: %w", err)
		return nil, gkillErr, err
	}

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
	repositories := findCtx.Repositories

	typeMatchReps := []reps.Repository{}
	if findCtx.ParsedFindQuery.UseRepTypes != nil && *findCtx.ParsedFindQuery.UseRepTypes {
		// RepType指定の場合、指定以外は除外する
		for _, repType := range *findCtx.ParsedFindQuery.RepTypes {
			switch repType {
			case "kmemo":
				for _, rep := range repositories.KmemoReps {
					typeMatchReps = append(typeMatchReps, rep)
				}
			case "urlog":
				for _, rep := range repositories.URLogReps {
					typeMatchReps = append(typeMatchReps, rep)
				}
			case "timeis":
				for _, rep := range repositories.TimeIsReps {
					typeMatchReps = append(typeMatchReps, rep)
				}
			case "mi":
				for _, rep := range repositories.MiReps {
					typeMatchReps = append(typeMatchReps, rep)
				}
			case "nlog":
				for _, rep := range repositories.NlogReps {
					typeMatchReps = append(typeMatchReps, rep)
				}
			case "lantana":
				for _, rep := range repositories.LantanaReps {
					typeMatchReps = append(typeMatchReps, rep)
				}
			case "rekyou":
				for _, rep := range repositories.ReKyouReps.ReKyouRepositories {
					typeMatchReps = append(typeMatchReps, rep)
				}
			case "directory":
				for _, rep := range repositories.IDFKyouReps {
					typeMatchReps = append(typeMatchReps, rep)
				}
			case "git_commit_log":
				for _, rep := range repositories.GitCommitLogReps {
					typeMatchReps = append(typeMatchReps, rep)
				}
			}
		}
	} else {
		typeMatchReps = repositories.Reps
	}

	targetRepNames := []string{}
	if findCtx.ParsedFindQuery.Reps != nil {
		targetRepNames = *findCtx.ParsedFindQuery.Reps
	}

	// PlaingだったらTimeIsRep以外は無視する
	if findCtx.ParsedFindQuery.UsePlaing != nil && *findCtx.ParsedFindQuery.UsePlaing {
		for _, rep := range findCtx.Repositories.TimeIsReps {
			repName, err := rep.GetRepName(ctx)
			if err != nil {
				return nil, err
			}

			for _, targetRepName := range targetRepNames {
				if targetRepName == repName {
					if _, exist := findCtx.MatchReps[repName]; !exist {
						findCtx.MatchReps[repName] = rep
					}
				}
			}
		}
	} else {
		for _, rep := range typeMatchReps {
			repName, err := rep.GetRepName(ctx)
			if err != nil {
				return nil, err
			}

			for _, targetRepName := range targetRepNames {
				if targetRepName == repName {
					if _, exist := findCtx.MatchReps[repName]; !exist {
						findCtx.MatchReps[repName] = rep
					}
				}
			}
		}
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

	matchReps := reps.Repositories{}
	for _, rep := range findCtx.MatchReps {
		matchReps = append(matchReps, rep)
	}

	// repで検索
	kyous, err := matchReps.FindKyous(ctx, findCtx.ParsedFindQuery)
	if err != nil {
		return nil, err
	}
	// textでマッチしたものをID検索
	textMatchKyous := []*reps.Kyou{}
	if len(targetIDs) != 0 {
		textMatchKyous, err = matchReps.FindKyous(ctx, matchTextFindByIDQuery)
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

func (f *FindFilter) filterMiForMi(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if findCtx.ParsedFindQuery.ForMi == nil || !(*findCtx.ParsedFindQuery.ForMi) {
		return nil, nil
	}

	// Miを取得位する
	// 作成日時以外の条件でmiを取得する。その後、作成日時で取得して追加する。
	allMis := map[string]*reps.Mi{}
	falseValue := false
	withoutCreatedMiFindQuery := *findCtx.ParsedFindQuery
	withoutCreatedMiFindQuery.IncludeCreateMi = &falseValue
	withoutCreatedMis, err := findCtx.Repositories.MiReps.FindMi(ctx, &withoutCreatedMiFindQuery)
	if err != nil {
		err = fmt.Errorf("error at get without created mis: %w", err)
		return nil, err
	}
	for _, mi := range withoutCreatedMis {
		if existMi, exist := allMis[mi.ID]; exist {
			if mi.UpdateTime.After(existMi.UpdateTime) {
				allMis[mi.ID] = mi
			}
		} else {
			allMis[mi.ID] = mi
		}
	}

	withCreatedMis, err := findCtx.Repositories.MiReps.FindMi(ctx, findCtx.ParsedFindQuery)
	if err != nil {
		err = fmt.Errorf("error at get all mis: %w", err)
		return nil, err
	}
	for _, mi := range withCreatedMis {
		if _, exist := allMis[mi.ID]; !exist {
			allMis[mi.ID] = mi
		}
	}

	// チェック状態から対象Miを抽出する
	targetMis := []*reps.Mi{}
	for _, mi := range allMis {
		if findCtx.ParsedFindQuery.MiCheckState != nil {
			switch string(*findCtx.ParsedFindQuery.MiCheckState) {
			case string(find.Checked):
				if mi.IsChecked {
					targetMis = append(targetMis, mi)
				}
			case string(find.UncCheck):
				if !mi.IsChecked {
					targetMis = append(targetMis, mi)
				}
			case string(find.All):
				targetMis = append(targetMis, mi)
			}
		}
	}

	// 対象MiのKyouのみを中有出する
	for _, mi := range targetMis {
		kyou, exist := findCtx.MatchKyousCurrent[mi.ID]
		if exist {
			findCtx.MatchKyousAtFilterMi[kyou.ID] = kyou
		}
	}

	findCtx.MatchMisAtFilterMi = allMis
	findCtx.MatchKyousCurrent = findCtx.MatchKyousAtFilterMi
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
		}
	}

	return nil, nil
}

func (f *FindFilter) filterLocationKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if findCtx.ParsedFindQuery.UseMap == nil || !*findCtx.ParsedFindQuery.UseMap {
		return nil, nil
	}

	matchKyous := map[string]*reps.Kyou{}
	var err error

	// 開始日を取得
	startTime := findCtx.ParsedFindQuery.CalendarStartDate
	endTime := findCtx.ParsedFindQuery.CalendarEndDate

	// radius, latitude, longitudeを取得
	var radius float64
	var latitude float64
	var longitude float64

	if findCtx.ParsedFindQuery.MapRadius != nil {
		radius = *findCtx.ParsedFindQuery.MapRadius / 1000
	}
	if findCtx.ParsedFindQuery.MapLatitude != nil {
		latitude = *findCtx.ParsedFindQuery.MapLatitude
	}
	if findCtx.ParsedFindQuery.MapLongitude != nil {
		longitude = *findCtx.ParsedFindQuery.MapLongitude
	}

	// 日付のnil解決 もしくは全部の日付
	isAllDays := false
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
	} else if startTime == nil && endTime == nil {
		isAllDays = true
	}
	// GPSLogを取得する
	gpsLogs := []*reps.GPSLog{}
	lenOfReps := len(findCtx.Repositories.GPSLogReps)

	existErr := false
	wg := &sync.WaitGroup{}
	gpsLogsCh := make(chan []*reps.GPSLog, lenOfReps)
	errch := make(chan error, lenOfReps)
	defer close(gpsLogsCh)
	defer close(errch)

	// 並列処理
	for _, rep := range findCtx.Repositories.GPSLogReps {
		wg.Add(1)
		go func(rep reps.GPSLogRepository) {
			defer wg.Done()
			// repで検索
			gpsLogs := []*reps.GPSLog{}
			if isAllDays {
				gpsLogs, err = rep.GetAllGPSLogs(ctx)
			} else {
				gpsLogs, err = rep.GetGPSLogs(ctx, startTime, endTime)
			}
			if err != nil {
				errch <- err
				panic(err)
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
		gpsLogs = append(gpsLogs, <-gpsLogsCh...)
	}

	// 並び替え
	sort.Slice(gpsLogs, func(i, j int) bool { return gpsLogs[i].RelatedTime.Before(gpsLogs[j].RelatedTime) })

	// 該当する時間を出す
	matchGPSLogSetList := [][]*reps.GPSLog{}

	preTrue := false // 一つ前の時間でtrueだった
	for i := range gpsLogs {
		if preTrue {
			matchGPSLogSetList = append(matchGPSLogSetList, []*reps.GPSLog{
				gpsLogs[i-1],
				gpsLogs[i],
			})
		}

		if calcDistance(latitude, longitude, gpsLogs[i].Latitude, gpsLogs[i].Longitude) <= radius {
			preTrue = true
		} else {
			preTrue = false
		}
	}

	// KyouがLocation内か判定
	for _, gpsLogSet := range matchGPSLogSetList {
		for _, kyou := range findCtx.MatchKyousCurrent {
			if kyou.RelatedTime.After(gpsLogSet[0].RelatedTime) && kyou.RelatedTime.Before(gpsLogSet[1].RelatedTime) {
				matchKyous[kyou.ID] = kyou
			}
		}
	}

	findCtx.MatchKyousAtFilterLocation = matchKyous
	findCtx.MatchKyousCurrent = matchKyous
	return nil, nil
}
func (f *FindFilter) overrideKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if findCtx.ParsedFindQuery.ForMi == nil || (*findCtx.ParsedFindQuery.ForMi) == false || findCtx.ParsedFindQuery.MiSortType == nil {
		// kyou検索の場合は何もしない
		return nil, nil
	}
	// miの場合は
	// 表示したとき、指定日時か作成日時かわかるようにDataTypeを上書きする
	for _, mi := range findCtx.MatchMisAtFilterMi {
		kyou, exist := findCtx.MatchKyousCurrent[mi.ID]
		if exist {
			kyou.DataType = mi.DataType
			if string(*findCtx.ParsedFindQuery.MiSortType) == string(find.CreateTime) {
				kyou.DataType = "mi_create"
				kyou.RelatedTime = mi.CreateTime
			} else if string(*findCtx.ParsedFindQuery.MiSortType) == string(find.EstimateStartTime) && mi.EstimateStartTime != nil {
				kyou.DataType = "mi_start"
				kyou.RelatedTime = *mi.EstimateStartTime
			} else if string(*findCtx.ParsedFindQuery.MiSortType) == string(find.EstimateEndTime) && mi.EstimateEndTime != nil {
				kyou.DataType = "mi_end"
				kyou.RelatedTime = *mi.EstimateEndTime
			} else if string(*findCtx.ParsedFindQuery.MiSortType) == string(find.LimitTime) && mi.LimitTime != nil {
				kyou.DataType = "mi_limit"
				kyou.RelatedTime = *mi.LimitTime
			} else {
				kyou.DataType = "mi_create"
				kyou.RelatedTime = mi.CreateTime
			}

		}
	}
	return nil, nil
}

func (f *FindFilter) sortResultKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if findCtx.ParsedFindQuery.ForMi == nil || (*findCtx.ParsedFindQuery.ForMi) == false || findCtx.ParsedFindQuery.MiSortType == nil {
		// kyouとしてソート
		sort.Slice(findCtx.ResultKyous, func(i, j int) bool {
			return findCtx.ResultKyous[i].RelatedTime.After(findCtx.ResultKyous[j].RelatedTime)
		})
		return nil, nil
	}

	// miとしてソート。指定日時でソートする。指定日時がないものは、末尾に作成日時でくっつける
	sortType := *findCtx.ParsedFindQuery.MiSortType
	sort.Slice(findCtx.ResultKyous, func(i, j int) bool {
		var iTime *time.Time = nil
		var jTime *time.Time = nil

		iMi := findCtx.MatchMisAtFilterMi[findCtx.ResultKyous[i].ID]
		jMi := findCtx.MatchMisAtFilterMi[findCtx.ResultKyous[j].ID]

		switch string(sortType) {
		case string(find.CreateTime):
			iTime = &iMi.CreateTime
			jTime = &jMi.CreateTime
			return iTime.After(*jTime)
		case string(find.EstimateStartTime):
			if iMi.EstimateStartTime != nil {
				iTime = iMi.EstimateStartTime
			}
			if jMi.EstimateStartTime != nil {
				jTime = jMi.EstimateStartTime
			}

			if iTime != nil && jTime != nil {
				return iTime.After(*jTime)
			}
			if iTime == nil && jTime != nil {
				return false
			}
			if iTime != nil && jTime == nil {
				return true
			}

			iTime = &iMi.CreateTime
			jTime = &jMi.CreateTime
			return iTime.After(*jTime)
		case string(find.EstimateEndTime):
			if iMi.EstimateEndTime != nil {
				iTime = iMi.EstimateEndTime
			}
			if jMi.EstimateEndTime != nil {
				jTime = jMi.EstimateEndTime
			}

			if iTime != nil && jTime != nil {
				return iTime.After(*jTime)
			}
			if iTime == nil && jTime != nil {
				return false
			}
			if iTime != nil && jTime == nil {
				return true
			}

			iTime = &iMi.CreateTime
			jTime = &jMi.CreateTime
			return iTime.After(*jTime)
		case string(find.LimitTime):
			if iMi.LimitTime != nil {
				iTime = iMi.LimitTime
			}
			if jMi.LimitTime != nil {
				jTime = jMi.LimitTime
			}

			if iTime != nil && jTime != nil {
				return iTime.After(*jTime)
			}
			if iTime == nil && jTime != nil {
				return false
			}
			if iTime != nil && jTime == nil {
				return true
			}

			iTime = &iMi.CreateTime
			jTime = &jMi.CreateTime
			return iTime.After(*jTime)
		}
		return false
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

func (f *FindFilter) filterImageKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if findCtx.ParsedFindQuery.IsImageOnly == nil || !(*findCtx.ParsedFindQuery.IsImageOnly) {
		return nil, nil
	}

	filterdImageKyous := map[string]*reps.Kyou{}
	for _, kyou := range findCtx.MatchKyousCurrent {
		if kyou.IsImage {
			filterdImageKyous[kyou.ID] = kyou
		}
	}
	findCtx.MatchKyousCurrent = filterdImageKyous
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
func (f *FindFilter) replaceLatestKyouInfos(ctx context.Context, findCtx *FindKyouContext, latestDatas []*account_state.LatestDataRepositoryAddress) ([]*message.GkillError, error) {
	latestKyousMap := map[string]*reps.Kyou{}
	for _, latestData := range latestDatas {
		// 対象じゃなければスキップ
		currentKyou, existInResult := findCtx.MatchKyousCurrent[latestData.TargetID]
		if !existInResult {
			continue
		}

		// すでに最新が入っていそうだったらそのままいれる
		if currentKyou.RepName == latestData.LatestDataRepositoryName && currentKyou.UpdateTime.Equal(latestData.DataUpdateTime) {
			latestKyousMap[currentKyou.ID] = currentKyou
			continue
		}

		// 最新が入っていなかったらもらってくる
		for _, rep := range findCtx.Repositories.Reps {
			repName, err := rep.GetRepName(ctx)
			if err != nil {
				err = fmt.Errorf("error at get rep name: %w", err)
				return nil, err
			}
			if repName != latestData.LatestDataRepositoryName {
				continue
			}

			kyouHistories, err := rep.GetKyouHistories(ctx, latestData.TargetID)
			if err != nil {
				err = fmt.Errorf("error at get kyou histories: %w", err)
				return nil, err
			}
			if len(kyouHistories) == 0 {
				continue
			}
			latestKyou := kyouHistories[0]

			// 削除されていればスキップ
			if latestKyou.IsDeleted {
				continue
			}

			latestKyousMap[latestKyou.ID] = latestKyou
		}
	}
	findCtx.MatchKyousCurrent = latestKyousMap
	return nil, nil
}

func calcDistance(lat1 float64, lng1 float64, lat2 float64, lng2 float64) float64 {
	lat1 *= R
	lng1 *= R
	lat2 *= R
	lng2 *= R
	return float64(6371.0) * math.Acos(math.Cos(lat1)*math.Cos(lat2)*math.Cos(lng2-lng1)+math.Sin(lat1)*math.Sin(lat2))
}
