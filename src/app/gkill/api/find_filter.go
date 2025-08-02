package api

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao"
	"github.com/mt3hr/gkill/src/app/gkill/dao/account_state"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/threads"
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
	findKyouContext.UserID = userID
	findKyouContext.Device = device
	findKyouContext.GkillDAOManager = gkillDAOManager
	findKyouContext.ParsedFindQuery = findQuery
	findKyouContext.MatchReps = map[string]reps.Repository{}
	findKyouContext.AllTags = map[string]*reps.Tag{}
	findKyouContext.AllHideTagsWhenUnchecked = map[string]*reps.Tag{}
	findKyouContext.MatchHideTagsWhenUncheckedKyou = map[string]*reps.Tag{}
	findKyouContext.MatchHideTagsWhenUncheckedTimeIs = map[string]*reps.Tag{}
	findKyouContext.RelatedTagIDs = map[string]interface{}{}
	findKyouContext.MatchTags = map[string]*reps.Tag{}
	findKyouContext.MatchTexts = map[string]*reps.Text{}
	findKyouContext.MatchTimeIssAtFindTimeIs = map[string]*reps.TimeIs{}
	findKyouContext.MatchTimeIssAtFilterTags = map[string]*reps.TimeIs{}
	findKyouContext.MatchTimeIsTags = map[string]*reps.Tag{}
	findKyouContext.MatchTimeIsTexts = map[string]*reps.Text{}
	findKyouContext.MatchKyousCurrent = map[string][]*reps.Kyou{}
	findKyouContext.MatchKyousAtFindKyou = map[string][]*reps.Kyou{}
	findKyouContext.MatchKyousAtFilterMi = map[string][]*reps.Kyou{}
	findKyouContext.MatchKyousAtFilterTags = map[string][]*reps.Kyou{}
	findKyouContext.MatchKyousAtFilterTimeIs = map[string][]*reps.Kyou{}
	findKyouContext.MatchKyousAtFilterLocation = map[string][]*reps.Kyou{}
	findKyouContext.MatchKyousAtFilterImage = map[string][]*reps.Kyou{}

	// ユーザのRep取得
	gkillErr, err := f.getRepositories(ctx, userID, device, gkillDAOManager, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at get repositories: %w", err)
		return nil, gkillErr, err
	}
	gkill_log.Trace.Printf("finish getRepositories")

	// フィルタ
	gkillErr, err = f.selectMatchRepsFromQuery(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at select match reps: %w", err)
		return nil, gkillErr, err
	}
	gkill_log.Trace.Printf("finish selectMatchRepsFromQuery")
	gkillErr, err = f.updateCache(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at update cache: %w", err)
		return nil, gkillErr, err
	}
	gkill_log.Trace.Printf("finish updateCache")

	// キャッシュ更新終わったら、
	// 最新のKyouがどのRepにあるかを取得
	latestDatas := map[string]*account_state.LatestDataRepositoryAddress{}
	latestDataRepositoryAddresses := map[string]*account_state.LatestDataRepositoryAddress{}

	if findKyouContext.Repositories.LatestDataRepositoryAddresses == nil {
		latestDatas, err = findKyouContext.Repositories.LatestDataRepositoryAddressDAO.GetAllLatestDataRepositoryAddresses(ctx)
		if err != nil {
			err = fmt.Errorf("error at get all latest data repository addresses: %w", err)
			return nil, nil, err
		}
		latestDataRepositoryAddresses = latestDatas
	} else {
		updatedLatestDatas, err := findKyouContext.Repositories.LatestDataRepositoryAddressDAO.GetLatestDataRepositoryAddressByUpdateTimeAfter(ctx, findKyouContext.Repositories.LastFindTime, math.MaxInt)
		if err != nil {
			err = fmt.Errorf("error at get updated latest data repository addresses: %w", err)
			return nil, nil, err
		}
		for _, latestData := range updatedLatestDatas {
			findKyouContext.Repositories.LatestDataRepositoryAddresses[latestData.TargetID] = latestData
		}
		latestDatas = latestDataRepositoryAddresses
	}
	findKyouContext.Repositories.LastFindTime = time.Now()

	if findQuery.UseTags != nil && *(findQuery.UseTags) {
		gkillErr, err = f.getAllTags(ctx, findKyouContext, latestDatas)
		if err != nil {
			err = fmt.Errorf("error at get all tags: %w", err)
			return nil, gkillErr, err
		}
		gkill_log.Trace.Printf("finish getAllTags")
		gkill_log.Trace.Printf("CurrentMatchKyous: %#v", findKyouContext.MatchKyousCurrent)
	}
	if findQuery.UseTags != nil && *(findQuery.UseTags) {
		gkillErr, err = f.getAllHideTagsWhenUnChecked(ctx, findKyouContext, userID, device, latestDatas)
		if err != nil {
			err = fmt.Errorf("error at get hide tags when unchecked tags: %w", err)
			return nil, gkillErr, err
		}
		gkill_log.Trace.Printf("finish getAllHideTagsWhenUnChecked")
		gkill_log.Trace.Printf("CurrentMatchKyous: %#v", findKyouContext.MatchKyousCurrent)
	}
	if findQuery.UseTags != nil && *(findQuery.UseTags) {
		gkillErr, err = f.findTags(ctx, findKyouContext, latestDatas)
		if err != nil {
			err = fmt.Errorf("error at find tags: %w", err)
			return nil, gkillErr, err
		}
		gkill_log.Trace.Printf("finish findTags")
		gkill_log.Trace.Printf("CurrentMatchKyous: %#v", findKyouContext.MatchKyousCurrent)
	}
	gkillErr, err = f.findTexts(ctx, findKyouContext, latestDatas)
	if err != nil {
		err = fmt.Errorf("error at find texts: %w", err)
		return nil, gkillErr, err
	}
	gkill_log.Trace.Printf("finish findTexts")
	gkill_log.Trace.Printf("CurrentMatchKyous: %#v", findKyouContext.MatchKyousCurrent)
	gkillErr, err = f.findTimeIsTexts(ctx, findKyouContext, latestDatas)
	if err != nil {
		err = fmt.Errorf("error at find timeis texts: %w", err)
		return nil, gkillErr, err
	}
	gkill_log.Trace.Printf("finish findTimeIsTexts")
	gkill_log.Trace.Printf("CurrentMatchKyous: %#v", findKyouContext.MatchKyousCurrent)
	gkillErr, err = f.findTimeIs(ctx, findKyouContext, latestDatas)
	if err != nil {
		err = fmt.Errorf("error at find timeis: %w", err)
		return nil, gkillErr, err
	}
	if findQuery.UseTags != nil && *(findQuery.UseTags) {
		gkill_log.Trace.Printf("finish findTimeIs")
		gkill_log.Trace.Printf("CurrentMatchKyous: %#v", findKyouContext.MatchKyousCurrent)
		gkillErr, err = f.findTimeIsTags(ctx, findKyouContext, latestDatas)
		if err != nil {
			err = fmt.Errorf("error at find timeis tags: %w", err)
			return nil, gkillErr, err
		}
		gkill_log.Trace.Printf("finish findTimeIsTags")
		gkill_log.Trace.Printf("CurrentMatchKyous: %#v", findKyouContext.MatchKyousCurrent)
	}
	if findQuery.UseTags != nil && *(findQuery.UseTags) {
		gkillErr, err = f.getMatchHideTagsWhenUnckedTimeIs(ctx, findKyouContext)
		if err != nil {
			err = fmt.Errorf("error at get match hide tags when unchecked timeis: %w", err)
			return nil, gkillErr, err
		}
		gkill_log.Trace.Printf("finish getMatchHideTagsWhenUnckedTimeIs")
		gkill_log.Trace.Printf("CurrentMatchKyous: %#v", findKyouContext.MatchKyousCurrent)
	}
	if findQuery.UseTags != nil && *(findQuery.UseTags) {
		gkillErr, err = f.filterTagsTimeIs(ctx, findKyouContext)
		if err != nil {
			err = fmt.Errorf("error at filter tags timeis: %w", err)
			return nil, gkillErr, err
		}
		gkill_log.Trace.Printf("finish filterTagsTimeIs")
		gkill_log.Trace.Printf("CurrentMatchKyous: %#v", findKyouContext.MatchKyousCurrent)
	}
	gkillErr, err = f.findKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at find kyous: %w", err)
		return nil, gkillErr, err
	}
	gkill_log.Trace.Printf("finish findKyous")
	gkill_log.Trace.Printf("CurrentMatchKyous: %#v", findKyouContext.MatchKyousCurrent)

	gkillErr, err = f.sortAndTrimKyousMap(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at sort and trim kyousMap: %w", err)
		return nil, gkillErr, err
	}
	gkill_log.Trace.Printf("finish findKyous")
	gkill_log.Trace.Printf("CurrentMatchKyous: %#v", findKyouContext.MatchKyousCurrent)

	gkillErr, err = f.filterMiForMi(ctx, findKyouContext) //miの場合のみ
	if err != nil {
		err = fmt.Errorf("error at filter mi for mi: %w", err)
		return nil, gkillErr, err
	}
	gkill_log.Trace.Printf("finish filterMiForMi")
	gkill_log.Trace.Printf("CurrentMatchKyous: %#v", findKyouContext.MatchKyousCurrent)
	if findQuery.UseTags != nil && *(findQuery.UseTags) {
		gkillErr, err = f.getMatchHideTagsWhenUnckedKyou(ctx, findKyouContext)
		if err != nil {
			err = fmt.Errorf("error at get match hide tags when unchecked timeis: %w", err)
			return nil, gkillErr, err
		}
		gkill_log.Trace.Printf("finish getMatchHideTagsWhenUnckedKyou")
		gkill_log.Trace.Printf("CurrentMatchKyous: %#v", findKyouContext.MatchKyousCurrent)
	}
	if findQuery.UseTags != nil && *(findQuery.UseTags) {
		gkillErr, err = f.filterTagsKyous(ctx, findKyouContext)
		if err != nil {
			err = fmt.Errorf("error at filter tags kyous: %w", err)
			return nil, gkillErr, err
		}
		gkill_log.Trace.Printf("finish filterTagsKyous")
		gkill_log.Trace.Printf("CurrentMatchKyous: %#v", findKyouContext.MatchKyousCurrent)
	}
	gkillErr, err = f.filterPlaingTimeIsKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at filter plaing time is kyous: %w", err)
		return nil, gkillErr, err
	}
	gkill_log.Trace.Printf("finish filterPlaingTimeIsKyous")
	gkill_log.Trace.Printf("CurrentMatchKyous: %#v", findKyouContext.MatchKyousCurrent)
	gkillErr, err = f.filterLocationKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at filter location kyous: %w", err)
		return nil, gkillErr, err
	}
	gkill_log.Trace.Printf("finish filterLocationKyous")
	gkill_log.Trace.Printf("CurrentMatchKyous: %#v", findKyouContext.MatchKyousCurrent)
	gkillErr, err = f.filterImageKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at filter image kyous: %w", err)
		return nil, gkillErr, err
	}
	gkill_log.Trace.Printf("finish filterImageKyous")
	gkill_log.Trace.Printf("CurrentMatchKyous: %#v", findKyouContext.MatchKyousCurrent)

	gkill_log.Trace.Printf("finish waitLatestDataWg")
	gkill_log.Trace.Printf("CurrentMatchKyous: %#v", findKyouContext.MatchKyousCurrent)

	gkillErr, err = f.replaceLatestKyouInfos(ctx, findKyouContext, latestDatas)
	if err != nil {
		err = fmt.Errorf("error at replace latest kyou infos: %w", err)
		return nil, gkillErr, err
	}
	gkill_log.Trace.Printf("finish replaceLatestKyouInfos")
	gkill_log.Trace.Printf("CurrentMatchKyous: %#v", findKyouContext.MatchKyousCurrent)

	for _, kyous := range findKyouContext.MatchKyousCurrent {
		findKyouContext.ResultKyous = append(findKyouContext.ResultKyous, kyous...)
	}

	gkillErr, err = f.overrideKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at override kyous: %w", err)
		return nil, gkillErr, err
	}
	gkill_log.Trace.Printf("finish overrideKyous")
	gkill_log.Trace.Printf("CurrentMatchKyous: %#v", findKyouContext.MatchKyousCurrent)

	gkillErr, err = f.sortResultKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at sort result kyous: %w", err)
		return nil, gkillErr, err
	}
	gkill_log.Trace.Printf("finish sortResultKyous")
	gkill_log.Trace.Printf("CurrentMatchKyous: %#v", findKyouContext.MatchKyousCurrent)

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

	if findCtx.ParsedFindQuery.IsImageOnly != nil && *findCtx.ParsedFindQuery.IsImageOnly {
		// ImageOnlyだったらIDFRep以外は無視する
		for _, rep := range repositories.IDFKyouReps {
			typeMatchReps = append(typeMatchReps, rep)
		}
	} else if findCtx.ParsedFindQuery.UsePlaing != nil && *findCtx.ParsedFindQuery.UsePlaing {
		// PlaingだったらTimeIsRep以外は無視する
		for _, rep := range repositories.TimeIsReps {
			typeMatchReps = append(typeMatchReps, rep)
		}
	} else if findCtx.ParsedFindQuery.UseRepTypes != nil && *findCtx.ParsedFindQuery.UseRepTypes {
		// RepType指定の場合、指定以外は除外する
		for _, repType := range *findCtx.ParsedFindQuery.RepTypes {
			switch repType {
			case "kmemo":
				for _, rep := range repositories.KmemoReps {
					typeMatchReps = append(typeMatchReps, rep)
				}
			case "kc":
				for _, rep := range repositories.KCReps {
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
		typeMatchReps = append(typeMatchReps, repositories.Reps...)
	}

	targetRepNames := []string{}
	if findCtx.ParsedFindQuery.Reps != nil {
		targetRepNames = *findCtx.ParsedFindQuery.Reps
	}

	miReps := reps.MiRepositories{}
	timeisReps := reps.TimeIsRepositories{}

	for _, rep := range typeMatchReps {
		repName, err := rep.GetRepName(ctx)
		if err != nil {
			return nil, err
		}

	rep_search:
		for _, targetRepName := range targetRepNames {
			if targetRepName == repName {
				for _, miRep := range repositories.MiReps {
					miRepName, err := miRep.GetRepName(ctx)
					if err != nil {
						return nil, err
					}
					if miRepName == repName {
						miReps = append(miReps, miRep)
						continue rep_search
					}
				}

				for _, timeisRep := range repositories.TimeIsReps {
					timeisRepName, err := timeisRep.GetRepName(ctx)
					if err != nil {
						return nil, err
					}
					if timeisRepName == repName {
						timeisReps = append(timeisReps, timeisRep)
						continue rep_search
					}
				}

				if _, exist := findCtx.MatchReps[repName]; !exist {
					findCtx.MatchReps[repName] = rep
				}
			}
		}
	}
	if len(miReps) != 0 {
		findCtx.MatchReps["Mi"] = miReps
	}
	if len(timeisReps) != 0 {
		findCtx.MatchReps["TimeIs"] = timeisReps
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
		done := threads.AllocateThread()
		go func(rep reps.Repository) {
			defer done()
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
			err = fmt.Errorf("error at update cache: %w: %w", e, err)
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

func (f *FindFilter) getAllTags(ctx context.Context, findCtx *FindKyouContext, latestDatas map[string]*account_state.LatestDataRepositoryAddress) ([]*message.GkillError, error) {
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
		done := threads.AllocateThread()
		go func(tagRep reps.TagRepository) {
			defer done()
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
			err = fmt.Errorf("error at get all tags: %w: %w", e, err)
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

	// タグの対象をリスト
	for _, tag := range findCtx.AllTags {
		latestData, exist := latestDatas[tag.ID]
		if !exist {
			continue
		}
		if !latestData.DataUpdateTime.Equal(tag.UpdateTime) {
			continue
		}
		if tag.IsDeleted {
			continue
		}
		findCtx.RelatedTagIDs[tag.TargetID] = struct{}{}
	}

	return nil, nil
}

func (f *FindFilter) getAllHideTagsWhenUnChecked(ctx context.Context, findCtx *FindKyouContext, userID string, device string, latestDatas map[string]*account_state.LatestDataRepositoryAddress) ([]*message.GkillError, error) {
	tagStructs, err := findCtx.GkillDAOManager.ConfigDAOs.TagStructDAO.GetTagStructs(ctx, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get tag tag structs: %w", err)
		return nil, err
	}
	hideTagNames := []string{}
	for _, tagStruct := range tagStructs {
		if tagStruct.IsForceHide {
			hideTagNames = append(hideTagNames, tagStruct.TagName)
		}
	}

	for _, hideTagName := range hideTagNames {
		hideTagsInReps, err := findCtx.Repositories.TagReps.GetTagsByTagName(ctx, hideTagName)
		if err != nil {
			err = fmt.Errorf("error at get tags by tagname tagname=%s: %w", hideTagName, err)
			return nil, err
		}
		for _, hideTag := range hideTagsInReps {
			latestData, exist := latestDatas[hideTag.ID]
			if !exist {
				continue
			}
			if !latestData.DataUpdateTime.Equal(hideTag.UpdateTime) {
				continue
			}
			if hideTag.IsDeleted {
				continue
			}
			findCtx.AllHideTagsWhenUnchecked[hideTag.ID] = hideTag
		}
	}
	return nil, nil
}

func (f *FindFilter) getMatchHideTagsWhenUnckedKyou(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if findCtx.ParsedFindQuery.Tags == nil {
		return nil, nil
	}
	for _, hideTag := range findCtx.AllHideTagsWhenUnchecked {
		isCheckedByUser := false
		for _, tagname := range *findCtx.ParsedFindQuery.Tags {
			if hideTag.Tag == tagname {
				isCheckedByUser = true
				break
			}
		}
		if !isCheckedByUser {
			findCtx.MatchHideTagsWhenUncheckedKyou[hideTag.ID] = hideTag
		}
	}
	return nil, nil
}

func (f *FindFilter) getMatchHideTagsWhenUnckedTimeIs(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if findCtx.ParsedFindQuery.UseTimeIsTags == nil || !(*findCtx.ParsedFindQuery.UseTimeIsTags) || findCtx.ParsedFindQuery.TimeIsTags == nil {
		return nil, nil
	}
	for _, hideTag := range findCtx.AllHideTagsWhenUnchecked {
		isCheckedByUser := false
		for _, tagname := range *findCtx.ParsedFindQuery.TimeIsTags {
			if hideTag.Tag == tagname {
				isCheckedByUser = true
				break
			}
		}
		if !isCheckedByUser {
			findCtx.MatchHideTagsWhenUncheckedTimeIs[hideTag.ID] = hideTag
		}
	}
	return nil, nil
}

func (f *FindFilter) findTimeIsTags(ctx context.Context, findCtx *FindKyouContext, latestDatas map[string]*account_state.LatestDataRepositoryAddress) ([]*message.GkillError, error) {
	// タグを使わない場合は全タグを使う
	if findCtx.ParsedFindQuery.UseTimeIsTags == nil || !(*findCtx.ParsedFindQuery.UseTimeIsTags) {
		for _, tag := range findCtx.AllTags {
			findCtx.MatchTimeIsTags[tag.Tag] = tag
		}
		return nil, nil
	}

	for _, tagName := range *findCtx.ParsedFindQuery.TimeIsTags {
		matchTags, err := findCtx.Repositories.TagReps.GetTagsByTagName(ctx, tagName)
		if err != nil {
			err = fmt.Errorf("error at get tags by name %s: %w", tagName, err)
			return nil, err
		}
		for _, tag := range matchTags {
			latestData, exist := latestDatas[tag.ID]
			if !exist {
				continue
			}
			if !latestData.DataUpdateTime.Equal(tag.UpdateTime) {
				continue
			}
			if tag.IsDeleted {
				continue
			}
			findCtx.MatchTimeIsTags[tag.ID] = tag
		}
	}
	return nil, nil
}

func (f *FindFilter) findTags(ctx context.Context, findCtx *FindKyouContext, latestDatas map[string]*account_state.LatestDataRepositoryAddress) ([]*message.GkillError, error) {
	trueValue := true
	falseValue := false

	query := &find.FindQuery{
		// IsDeleted: &falseValue, // TagReps.FindTags内に考慮があるため削除
		UseWords: &trueValue,
		Words:    findCtx.ParsedFindQuery.Tags,
		WordsAnd: &falseValue,
	}
	matchTags, err := findCtx.Repositories.TagReps.FindTags(ctx, query)
	if err != nil {
		err = fmt.Errorf("error at get tags by name %#v: %w", findCtx.ParsedFindQuery.Words, err)
		return nil, err
	}
	for _, tag := range matchTags {
		latestData, exist := latestDatas[tag.ID]
		if !exist {
			continue
		}
		if !latestData.DataUpdateTime.Equal(tag.UpdateTime) {
			continue
		}
		if tag.IsDeleted {
			continue
		}
		findCtx.MatchTags[tag.ID] = tag
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
	kyousMap, err := matchReps.FindKyous(ctx, findCtx.ParsedFindQuery)
	if err != nil {
		return nil, err
	}
	// textでマッチしたものをID検索
	textMatchKyousMap := map[string][]*reps.Kyou{}
	if len(targetIDs) != 0 {
		textMatchKyousMap, err = matchReps.FindKyous(ctx, matchTextFindByIDQuery)
		if err != nil {
			return nil, err
		}
	}
	for id, textMatchKyous := range textMatchKyousMap {
		if _, exist := kyousMap[id]; !exist {
			kyousMap[id] = []*reps.Kyou{}
		}
		kyousMap[id] = append(kyousMap[id], textMatchKyous...)
	}

	// 削除隅のものは消す
	deleteTargetIDs := []string{}
	for id, kyous := range kyousMap {
		var latestKyou *reps.Kyou
		for _, kyou := range kyous {
			if latestKyou == nil {
				latestKyou = kyou
			} else {
				if kyou.UpdateTime.After(latestKyou.UpdateTime) {
					latestKyou = kyou
				}
			}
		}
		if latestKyou.IsDeleted {
			deleteTargetIDs = append(deleteTargetIDs, id)
		}
	}
	for _, deleteTargetID := range deleteTargetIDs {
		delete(kyousMap, deleteTargetID)
	}
	findCtx.MatchKyousAtFindKyou = kyousMap
	findCtx.MatchKyousCurrent = findCtx.MatchKyousAtFindKyou
	return nil, nil
}

func (f *FindFilter) sortAndTrimKyousMap(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	resultKyous := map[string][]*reps.Kyou{}

	deleteTargetKyouIDs := []string{}
	for id, kyous := range findCtx.MatchKyousCurrent {
		if len(kyous) == 0 {
			deleteTargetKyouIDs = append(deleteTargetKyouIDs, id)
			continue
		}

		trimedKyousMap := map[int64]*reps.Kyou{}
		for _, kyou := range kyous {
			if findCtx.ParsedFindQuery.UseCalendar != nil && *findCtx.ParsedFindQuery.UseCalendar {
				if (findCtx.ParsedFindQuery.CalendarStartDate != nil && kyou.RelatedTime.Before(*findCtx.ParsedFindQuery.CalendarStartDate)) ||
					(findCtx.ParsedFindQuery.CalendarEndDate != nil && kyou.RelatedTime.After(*findCtx.ParsedFindQuery.CalendarEndDate)) {
					continue
				}
			}
			trimedKyousMap[kyou.RelatedTime.Unix()] = kyou
		}

		sortedKyous := []*reps.Kyou{}
		for _, kyou := range trimedKyousMap {
			sortedKyous = append(sortedKyous, kyou)
		}
		sort.Slice(sortedKyous, func(i int, j int) bool {
			return sortedKyous[i].RelatedTime.After(sortedKyous[j].RelatedTime)
		})

		resultKyous[id] = sortedKyous
	}

	for _, deleteTargetKyouID := range deleteTargetKyouIDs {
		delete(resultKyous, deleteTargetKyouID)
	}

	if (findCtx.ParsedFindQuery.UsePlaing != nil && *findCtx.ParsedFindQuery.UsePlaing) || (findCtx.ParsedFindQuery.ForMi != nil && *findCtx.ParsedFindQuery.ForMi) {
		for id := range resultKyous {
			resultKyous[id] = []*reps.Kyou{resultKyous[id][0]}
		}
	}

	findCtx.MatchKyousCurrent = resultKyous
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
		kyous, exist := findCtx.MatchKyousCurrent[mi.ID]
		if exist {
			findCtx.MatchKyousAtFilterMi[mi.ID] = kyous
		}
	}

	findCtx.MatchMisAtFilterMi = allMis
	findCtx.MatchKyousCurrent = findCtx.MatchKyousAtFilterMi
	return nil, nil
}

func (f *FindFilter) filterTagsKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if findCtx.ParsedFindQuery.Tags != nil && findCtx.ParsedFindQuery.TagsAnd != nil && !(*findCtx.ParsedFindQuery.TagsAnd) {
		// ORの場合のフィルタリング処理

		// タグ対象Kyouリスト
		matchOrTagKyousMap := map[string][]*reps.Kyou{}
		for _, tag := range findCtx.MatchTags {
			kyou, exist := findCtx.MatchKyousCurrent[tag.TargetID]
			if !exist {
				continue
			}
			matchOrTagKyousMap[tag.TargetID] = kyou
		}

		// タグ無しKyouリスト
		noTagKyous := map[string][]*reps.Kyou{}
		for id, kyou := range findCtx.MatchKyousCurrent {
			_, relatedTagKyou := findCtx.RelatedTagIDs[id]
			if !relatedTagKyou {
				noTagKyous[id] = kyou
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
		for id, kyou := range matchOrTagKyousMap {
			findCtx.MatchKyousAtFilterTags[id] = kyou
		}
		if existNoTags {
			for id, kyou := range noTagKyous {
				findCtx.MatchKyousAtFilterTags[id] = kyou
			}
		}

		// 非表示タグの対象を消す
		for _, hideTag := range findCtx.MatchHideTagsWhenUncheckedKyou {
			delete(findCtx.MatchKyousAtFilterTags, hideTag.TargetID)
		}

		findCtx.MatchKyousCurrent = findCtx.MatchKyousAtFilterTags
	} else if findCtx.ParsedFindQuery.Tags != nil && findCtx.ParsedFindQuery.TagsAnd != nil && (*findCtx.ParsedFindQuery.TagsAnd) {
		// ANDの場合のフィルタリング処理
		tagNameMap := map[string]map[string][]*reps.Kyou{} // map[タグ名][kyou.ID（tagTargetID）] = reps.kyou

		for _, tag := range findCtx.MatchTags {
			isTagInQuery := false
			for _, tagName := range *findCtx.ParsedFindQuery.Tags {
				if tagName == tag.Tag {
					isTagInQuery = true
					break
				}
			}
			if !isTagInQuery {
				continue
			}

			kyous, exist := findCtx.MatchKyousCurrent[tag.TargetID]
			if !exist {
				continue
			}

			if _, exist := tagNameMap[tag.Tag]; !exist {
				tagNameMap[tag.Tag] = map[string][]*reps.Kyou{}
			}

			tagNameMap[tag.Tag][tag.TargetID] = kyous
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
			for id, kyous := range findCtx.MatchKyousCurrent {
				_, relatedTagKyou := findCtx.RelatedTagIDs[id]
				if !relatedTagKyou {
					if _, exist := tagNameMap[NoTags][id]; !exist {
						tagNameMap[NoTags] = map[string][]*reps.Kyou{}
					}
					tagNameMap[NoTags][id] = kyous
				}
			}
		}

		// tagNameMapの全部のタグ名に存在するKyouだけを抽出
		hasAllMatchTagsKyousMap := map[string]map[string][]*reps.Kyou{}
		// 初回は全部いれる
		for tagName, kyouIDMap := range tagNameMap {
			for kyouID, kyous := range kyouIDMap {
				if _, exist := hasAllMatchTagsKyousMap[tagName]; !exist {
					hasAllMatchTagsKyousMap[tagName] = map[string][]*reps.Kyou{}
				}
				hasAllMatchTagsKyousMap[tagName][kyouID] = kyous
			}
		}
		for tagName := range tagNameMap {
			matchThisLoopKyousMap := map[string]map[string][]*reps.Kyou{}
			// 初回ループ以外は、
			// 以前のタグにマッチしたもの（hasAllMatchTagsKyous）にあり、かつ
			// 今回のタグにマッチしたもの　をいれる。
			if _, exist := matchThisLoopKyousMap[tagName]; !exist {
				matchThisLoopKyousMap[tagName] = map[string][]*reps.Kyou{}
			}

			beforeMatchKyous := map[string][]*reps.Kyou{}
			for _, kyous := range hasAllMatchTagsKyousMap {
				for kyouID, kyou := range kyous {
					beforeMatchKyous[kyouID] = kyou
				}
			}
			currentMatchKyous := map[string][]*reps.Kyou{}
			for kyouID, kyous := range tagNameMap[tagName] {
				currentMatchKyous[kyouID] = kyous
			}

			for beforeKyouID, kyous := range beforeMatchKyous {
				for currentKyouID := range currentMatchKyous {
					if beforeKyouID == currentKyouID {
						matchThisLoopKyousMap[tagName][currentKyouID] = kyous
						break
					}
				}
			}
			hasAllMatchTagsKyousMap = matchThisLoopKyousMap
		}

		findCtx.MatchKyousAtFilterTags = map[string][]*reps.Kyou{}
		for _, matchTagsKyousMap := range hasAllMatchTagsKyousMap {
			for kyouID, kyous := range matchTagsKyousMap {
				findCtx.MatchKyousAtFilterTags[kyouID] = kyous
			}
		}

		// 非表示タグの対象を消す
		for _, hideTag := range findCtx.MatchHideTagsWhenUncheckedKyou {
			delete(findCtx.MatchKyousAtFilterTags, hideTag.TargetID)
		}

		findCtx.MatchKyousCurrent = findCtx.MatchKyousAtFilterTags
	}

	return nil, nil
}

func (f *FindFilter) filterTagsTimeIs(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if findCtx.ParsedFindQuery.UseTimeIsTags == nil || !(*findCtx.ParsedFindQuery.UseTimeIsTags) {
		findCtx.MatchTimeIssAtFilterTags = findCtx.MatchTimeIssAtFindTimeIs
		return nil, nil
	}

	if findCtx.ParsedFindQuery.TimeIsTags != nil && findCtx.ParsedFindQuery.TimeIsTagsAnd != nil && !(*findCtx.ParsedFindQuery.TimeIsTagsAnd) {
		// ORの場合のフィルタリング処理

		// タグ対象Kyouリスト
		matchOrTagTimeIss := map[string]*reps.TimeIs{}
		for _, tag := range findCtx.MatchTimeIsTags {
			matchTimeis, exist := findCtx.MatchTimeIssAtFindTimeIs[tag.TargetID]
			if !exist {
				continue
			}
			matchOrTagTimeIss[matchTimeis.ID] = matchTimeis
		}

		// タグ無しKyouリスト
		noTagTimeIss := map[string]*reps.TimeIs{}
		for _, timeis := range findCtx.MatchTimeIssAtFindTimeIs {
			_, relatedTagTimeIs := findCtx.RelatedTagIDs[timeis.ID]
			if !relatedTagTimeIs {
				noTagTimeIss[timeis.ID] = timeis
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
			findCtx.MatchTimeIssAtFilterTags[timeis.ID] = timeis
		}
		if existNoTags {
			for _, timeis := range noTagTimeIss {
				findCtx.MatchTimeIssAtFilterTags[timeis.ID] = timeis
			}
		}

		// 非表示タグの対象を消す
		for _, hideTag := range findCtx.MatchHideTagsWhenUncheckedTimeIs {
			delete(findCtx.MatchTimeIssAtFilterTags, hideTag.TargetID)
		}

	} else if findCtx.ParsedFindQuery.TimeIsTags != nil && findCtx.ParsedFindQuery.TimeIsTagsAnd != nil && (*findCtx.ParsedFindQuery.TimeIsTagsAnd) {
		// ANDの場合のフィルタリング処理

		tagNameMap := map[string]map[string]*reps.TimeIs{} // map[タグ名][kyou.ID（tagTargetID）] = reps.TimeIs

		for _, tag := range findCtx.MatchTimeIsTags {
			isTagInQuery := false
			for _, tagName := range *findCtx.ParsedFindQuery.TimeIsTags {
				if tagName == tag.Tag {
					isTagInQuery = true
					break
				}
			}
			if !isTagInQuery {
				continue
			}
			timeis, exist := findCtx.MatchTimeIssAtFindTimeIs[tag.TargetID]
			if !exist {
				continue
			}
			if _, exist := tagNameMap[tag.Tag]; !exist {
				tagNameMap[tag.Tag] = map[string]*reps.TimeIs{}
			}

			tagNameMap[tag.Tag][timeis.ID] = timeis
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
				_, relatedTagTimeIs := findCtx.RelatedTagIDs[timeis.ID]
				if !relatedTagTimeIs {
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
					hasAllMatchTagsTimeIssMap[timeis.ID] = timeis
				}
			default:
				matchThisLoopTimeIssMap := map[string]*reps.TimeIs{}
				for _, timeis := range timeisIDMap {
					// 初回ループ以外は、
					// 以前のタグにマッチしたもの（hasAllMatchTagsKyous）にあり、かつ
					// 今回のタグにマッチしたもの　をいれる。
					if _, exist := hasAllMatchTagsTimeIssMap[timeis.ID]; exist {
						matchThisLoopTimeIssMap[timeis.ID] = timeis
					}
				}
				hasAllMatchTagsTimeIssMap = matchThisLoopTimeIssMap
			}
			index++
		}

		findCtx.MatchTimeIssAtFilterTags = hasAllMatchTagsTimeIssMap

		// 非表示タグの対象を消す
		for _, hideTag := range findCtx.MatchHideTagsWhenUncheckedTimeIs {
			delete(findCtx.MatchTimeIssAtFilterTags, hideTag.TargetID)
		}
	}
	return nil, nil
}

func (f *FindFilter) filterPlaingTimeIsKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if findCtx.ParsedFindQuery.UseTimeIs == nil || !(*findCtx.ParsedFindQuery.UseTimeIs) {
		return nil, nil
	}

	for _, timeis := range findCtx.MatchTimeIssAtFilterTags {
		for id, kyous := range findCtx.MatchKyousCurrent {
			if (timeis.EndTime != nil && kyous[0].RelatedTime.After(timeis.StartTime) && kyous[0].RelatedTime.Before(*timeis.EndTime)) || (timeis.EndTime == nil && kyous[0].RelatedTime.After(timeis.StartTime)) {
				findCtx.MatchKyousAtFilterTimeIs[id] = kyous
			}
		}
	}
	findCtx.MatchKyousCurrent = findCtx.MatchKyousAtFilterTimeIs
	return nil, nil
}

func (f *FindFilter) findTimeIs(ctx context.Context, findCtx *FindKyouContext, latestDatas map[string]*account_state.LatestDataRepositoryAddress) ([]*message.GkillError, error) {
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
		done := threads.AllocateThread()
		go func(rep reps.TimeIsRepository) {
			defer done()
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
			err = fmt.Errorf("error at find timeiss: %w: %w", e, err)
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
				latestData, exist := latestDatas[timeis.ID]
				if !exist {
					continue
				}
				if !latestData.DataUpdateTime.Equal(timeis.UpdateTime) {
					continue
				}
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

	matchKyous := map[string][]*reps.Kyou{}
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
	if (startTime != nil && endTime == nil) && findCtx.ParsedFindQuery.UseCalendar != nil && *findCtx.ParsedFindQuery.UseCalendar {
		s := time.Time(*startTime)
		e := time.Time(*startTime).Add(time.Hour*23 + time.Minute*59 + time.Second*59)
		startTime = &s
		endTime = &e
	} else if (startTime != nil && endTime != nil) && findCtx.ParsedFindQuery.UseCalendar != nil && *findCtx.ParsedFindQuery.UseCalendar {
		s := time.Time(*startTime)
		e := time.Time(*endTime).Add(time.Hour*23 + time.Minute*59 + time.Second*59)
		startTime = &s
		endTime = &e
	} else if (startTime == nil && endTime == nil) || (findCtx.ParsedFindQuery.UseCalendar == nil || !*findCtx.ParsedFindQuery.UseCalendar) {
		isAllDays = true
	}
	// GPSLogを取得する
	matchGPSLogs := []*reps.GPSLog{}
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
		done := threads.AllocateThread()
		go func(rep reps.GPSLogRepository) {
			defer done()
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
			err = fmt.Errorf("error at filter gpslogs: %w: %w", e, err)
			existErr = true
		}
	}
	if existErr {
		return nil, err
	}

	// GPSLog集約
	for range lenOfReps {
		matchGPSLogs = append(matchGPSLogs, <-gpsLogsCh...)
	}

	// 並び替え
	sort.Slice(matchGPSLogs, func(i, j int) bool { return matchGPSLogs[i].RelatedTime.Before(matchGPSLogs[j].RelatedTime) })

	// 該当する時間を出す
	matchGPSLogSetList := [][]*reps.GPSLog{}

	preTrue := false // 一つ前の時間でtrueだった
	for i := range matchGPSLogs {
		if preTrue {
			matchGPSLogSetList = append(matchGPSLogSetList, []*reps.GPSLog{
				matchGPSLogs[i-1],
				matchGPSLogs[i],
			})
		}

		if calcDistance(latitude, longitude, matchGPSLogs[i].Latitude, matchGPSLogs[i].Longitude) <= radius {
			preTrue = true
		} else {
			preTrue = false
		}
	}

	// KyouがLocation内か判定
	for _, gpsLogSet := range matchGPSLogSetList {
		for id, kyous := range findCtx.MatchKyousCurrent {
			if kyous[0].RelatedTime.After(gpsLogSet[0].RelatedTime) && kyous[0].RelatedTime.Before(gpsLogSet[1].RelatedTime) {
				matchKyous[id] = kyous
			}
		}
	}

	findCtx.MatchKyousAtFilterLocation = matchKyous
	findCtx.MatchKyousCurrent = matchKyous
	return nil, nil
}
func (f *FindFilter) overrideKyous(_ context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if findCtx.ParsedFindQuery.ForMi == nil || findCtx.ParsedFindQuery.MiSortType == nil || !*findCtx.ParsedFindQuery.ForMi {
		// kyou検索の場合は何もしない
		return nil, nil
	}
	// miの場合は
	// 表示したとき、指定日時か作成日時かわかるようにDataTypeを上書きする
	for _, mi := range findCtx.MatchMisAtFilterMi {
		kyous, exist := findCtx.MatchKyousCurrent[mi.ID]
		if exist {
			kyous[0].DataType = mi.DataType
			if string(*findCtx.ParsedFindQuery.MiSortType) == string(find.CreateTime) {
				kyous[0].DataType = "mi_create"
				kyous[0].RelatedTime = mi.CreateTime
			} else if string(*findCtx.ParsedFindQuery.MiSortType) == string(find.EstimateStartTime) && mi.EstimateStartTime != nil {
				kyous[0].DataType = "mi_start"
				kyous[0].RelatedTime = *mi.EstimateStartTime
			} else if string(*findCtx.ParsedFindQuery.MiSortType) == string(find.EstimateEndTime) && mi.EstimateEndTime != nil {
				kyous[0].DataType = "mi_end"
				kyous[0].RelatedTime = *mi.EstimateEndTime
			} else if string(*findCtx.ParsedFindQuery.MiSortType) == string(find.LimitTime) && mi.LimitTime != nil {
				kyous[0].DataType = "mi_limit"
				kyous[0].RelatedTime = *mi.LimitTime
			} else {
				kyous[0].DataType = "mi_create"
				kyous[0].RelatedTime = mi.CreateTime
			}

		}
	}
	return nil, nil
}

func (f *FindFilter) sortResultKyous(_ context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if findCtx.ParsedFindQuery.ForMi == nil || findCtx.ParsedFindQuery.MiSortType == nil || !*findCtx.ParsedFindQuery.ForMi {
		// kyouとしてソート
		sort.Slice(findCtx.ResultKyous, func(i, j int) bool {
			iUnix := findCtx.ResultKyous[i].RelatedTime.Unix()
			jUnix := findCtx.ResultKyous[j].RelatedTime.Unix()
			if iUnix > jUnix {
				return true
			} else if iUnix < jUnix {
				return false
			} else if findCtx.ResultKyous[i].ID < findCtx.ResultKyous[j].ID {
				return true
			}
			return false
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
			return iTime.Before(*jTime)
		case string(find.EstimateStartTime):
			if iMi.EstimateStartTime != nil {
				iTime = iMi.EstimateStartTime
			}
			if jMi.EstimateStartTime != nil {
				jTime = jMi.EstimateStartTime
			}

			if iTime != nil && jTime != nil {
				return iTime.Before(*jTime)
			}
			if iTime == nil && jTime != nil {
				return false
			}
			if iTime != nil && jTime == nil {
				return true
			}

			iTime = &iMi.CreateTime
			jTime = &jMi.CreateTime
			return iTime.Before(*jTime)
		case string(find.EstimateEndTime):
			if iMi.EstimateEndTime != nil {
				iTime = iMi.EstimateEndTime
			}
			if jMi.EstimateEndTime != nil {
				jTime = jMi.EstimateEndTime
			}

			if iTime != nil && jTime != nil {
				return iTime.Before(*jTime)
			}
			if iTime == nil && jTime != nil {
				return false
			}
			if iTime != nil && jTime == nil {
				return true
			}

			iTime = &iMi.CreateTime
			jTime = &jMi.CreateTime
			return iTime.Before(*jTime)
		case string(find.LimitTime):
			if iMi.LimitTime != nil {
				iTime = iMi.LimitTime
			}
			if jMi.LimitTime != nil {
				jTime = jMi.LimitTime
			}

			if iTime != nil && jTime != nil {
				return iTime.Before(*jTime)
			}
			if iTime == nil && jTime != nil {
				return false
			}
			if iTime != nil && jTime == nil {
				return true
			}

			iTime = &iMi.CreateTime
			jTime = &jMi.CreateTime
			return iTime.Before(*jTime)
		}
		return false
	})
	return nil, nil
}

func (f *FindFilter) findTexts(ctx context.Context, findCtx *FindKyouContext, latestDatas map[string]*account_state.LatestDataRepositoryAddress) ([]*message.GkillError, error) {
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
		// IsDeleted: &falseValue, // TextReps.FindTexts内に考慮があるため削除
		UseWords: &trueValue,
		Words:    &words,
		NotWords: &notWords,
		WordsAnd: &falseValue,
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
		done := threads.AllocateThread()
		go func(textRep reps.TextRepository) {
			defer done()
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
			err = fmt.Errorf("error at find  texts: %w: %w", e, err)
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
			latestData, exist := latestDatas[text.ID]
			if !exist {
				continue
			}
			if !latestData.DataUpdateTime.Equal(text.UpdateTime) {
				continue
			}
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

	filterdImageKyous := map[string][]*reps.Kyou{}
	for id, kyous := range findCtx.MatchKyousCurrent {
		if kyous[0].IsImage {
			filterdImageKyous[id] = kyous
		}
	}
	findCtx.MatchKyousCurrent = filterdImageKyous
	return nil, nil
}

func (f *FindFilter) findTimeIsTexts(ctx context.Context, findCtx *FindKyouContext, latestDatas map[string]*account_state.LatestDataRepositoryAddress) ([]*message.GkillError, error) {
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
		// IsDeleted: &falseValue, // TextReps.FindTexts内に考慮があるため削除
		UseWords: &trueValue,
		Words:    &words,
		NotWords: &notWords,
		WordsAnd: &falseValue,
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
		done := threads.AllocateThread()
		go func(textRep reps.TextRepository) {
			defer done()
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
			err = fmt.Errorf("error at find  texts: %w: %w", e, err)
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
			latestData, exist := latestDatas[text.ID]
			if !exist {
				continue
			}
			if !latestData.DataUpdateTime.Equal(text.UpdateTime) {
				continue
			}
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
func (f *FindFilter) replaceLatestKyouInfos(ctx context.Context, findCtx *FindKyouContext, latestDatas map[string]*account_state.LatestDataRepositoryAddress) ([]*message.GkillError, error) {
	latestKyousMap := map[string][]*reps.Kyou{}

	// startTime := findCtx.ParsedFindQuery.CalendarStartDate
	// endTime := findCtx.ParsedFindQuery.CalendarEndDate

	for id, currentKyou := range findCtx.MatchKyousCurrent {
		latestData, exist := latestDatas[id]
		if !exist {
			continue
		}

		isMiData := strings.HasPrefix(currentKyou[0].DataType, "mi")
		isTimeIsData := strings.HasPrefix(currentKyou[0].DataType, "timeis")
		isUsePlaing := findCtx.ParsedFindQuery.UsePlaing != nil && *(findCtx.ParsedFindQuery.UsePlaing) && findCtx.ParsedFindQuery.PlaingTime != nil

		// すでに最新が入っていそうだったらそのままいれる RepNameは運用都合でチェックしない
		// Miもそのままいれる
		if ((currentKyou[0].UpdateTime.Equal(latestData.DataUpdateTime) || isMiData || isTimeIsData) && !isUsePlaing) ||
			(currentKyou[0].UpdateTime.Equal(latestData.DataUpdateTime) && isUsePlaing) {
			latestKyousMap[id] = currentKyou
			continue
		} else if isUsePlaing {
			continue
		}

		// はい入ってなかったら最新のKyouを取得する
		latestKyou, err := findCtx.Repositories.Reps.GetKyou(ctx, latestData.TargetID, &latestData.DataUpdateTime)
		if err != nil {
			return nil, fmt.Errorf("error at get latest kyou: %w", err)
		}
		latestKyousMap[id] = []*reps.Kyou{latestKyou}
	}

	// miの場合は最新以外消す
	isForMi := findCtx.ParsedFindQuery.ForMi != nil && *findCtx.ParsedFindQuery.ForMi
	if isForMi {
		for id, kyous := range latestKyousMap {
			sort.Slice(kyous, func(i, j int) bool {
				return kyous[i].UpdateTime.After(kyous[j].UpdateTime)
			})
			latestKyousMap[id] = []*reps.Kyou{kyous[0]}
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
