package api

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

const (
	NoTags = "no tags"
)

type FindFilter struct {
}

func (f *FindFilter) FindKyous(ctx context.Context, userID string, device string, gkillDAOManager *dao.GkillDAOManager, findQuery *find.FindQuery) ([]reps.Kyou, []*message.GkillError, error) {
	findKyouContext := &FindKyouContext{}

	// QueryをContextに入れる
	findKyouContext.UserID = userID
	findKyouContext.Device = device
	findKyouContext.GkillDAOManager = gkillDAOManager
	findKyouContext.ParsedFindQuery = findQuery
	findKyouContext.MatchReps = map[string]reps.Repository{}
	findKyouContext.AllTags = map[string]reps.Tag{}
	findKyouContext.AllHideTagsWhenUnchecked = map[string]reps.Tag{}
	findKyouContext.MatchHideTagsWhenUncheckedKyou = map[string]reps.Tag{}
	findKyouContext.MatchHideTagsWhenUncheckedTimeIs = map[string]reps.Tag{}
	findKyouContext.RelatedTagIDs = map[string]struct{}{}
	findKyouContext.MatchTags = map[string]reps.Tag{}
	findKyouContext.MatchTexts = map[string]reps.Text{}
	findKyouContext.MatchTimeIssAtFindTimeIs = map[string]reps.TimeIs{}
	findKyouContext.MatchTimeIssAtFilterTags = map[string]reps.TimeIs{}
	findKyouContext.MatchTimeIsTags = map[string]reps.Tag{}
	findKyouContext.MatchTimeIsTexts = map[string]reps.Text{}
	findKyouContext.MatchKyousCurrent = map[string][]reps.Kyou{}
	// メモリキャッシュ有効の場合、型につき1つのDBに全部のデータがある。
	// だから、LatestDataRepositoryAddressを知る必要がない
	findKyouContext.DisableLatestDataRepositoryCache = gkill_options.IsCacheInMemory

	// ユーザのRep取得
	gkillErr, err := f.getRepositories(ctx, userID, device, gkillDAOManager, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at get repositories: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish getRepositories")

	// フィルタ
	gkillErr, err = f.selectMatchRepsFromQuery(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at select match reps: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish selectMatchRepsFromQuery")
	if findKyouContext.ParsedFindQuery.UpdateCache {
		gkillErr, err = f.updateCache(ctx, findKyouContext)
		if err != nil {
			err = fmt.Errorf("error at update cache: %w", err)
			return nil, gkillErr, err
		}
		slog.Log(ctx, gkill_log.Trace, "finish updateCache")

	}

	if len(findKyouContext.Repositories.LatestDataRepositoryAddresses) == 0 {
		latestDatas, err := findKyouContext.Repositories.LatestDataRepositoryAddressDAO.GetAllLatestDataRepositoryAddresses(ctx)
		if err != nil {
			err = fmt.Errorf("error at get all latest data repository addresses: %w", err)
			return nil, nil, err
		}
		findKyouContext.Repositories.LatestDataRepositoryAddresses = latestDatas
	} else {
		updatedLatestDatas, err := findKyouContext.Repositories.LatestDataRepositoryAddressDAO.GetLatestDataRepositoryAddressByUpdateTimeAfter(ctx, findKyouContext.Repositories.LastUpdatedLatestDataRepositoryAddressCacheFindTime, math.MaxInt)
		if err != nil {
			err = fmt.Errorf("error at get updated latest data repository addresses: %w", err)
			return nil, nil, err
		}
		for _, latestData := range updatedLatestDatas {
			findKyouContext.Repositories.LatestDataRepositoryAddresses[latestData.TargetID] = latestData
		}
	}
	findKyouContext.Repositories.LastUpdatedLatestDataRepositoryAddressCacheFindTime = time.Now()
	slog.Log(ctx, gkill_log.Trace, "finish update latest data repository address")

	wg := &sync.WaitGroup{}
	doneCh := make(chan struct{}, 6 /* chのかず */)
	errch := make(chan error, 23 /* chのかず */)
	gkillErrch := make(chan []*message.GkillError, 6 /* chのかず */)
	defer close(errch)
	defer close(gkillErrch)

	catchErrFunc := func() ([]*message.GkillError, error) {
		gkillErrors := []*message.GkillError{}
		errs := []error{}
	loop:
		for {
			select {
			case <-doneCh:
				err := <-errch
				if err != nil {
					errs = append(errs, err)
				}
				gkillErr := <-gkillErrch
				if len(gkillErr) != 0 {
					gkillErrors = append(gkillErrors, gkillErr...)
				}
			default:
				break loop
			}
		}
		var err error
		for _, e := range errs {
			err = fmt.Errorf("%w %w", err, e)
		}
		return gkillErrors, err
	}

	// タグ取得
	if findQuery.UseTags {
		wg.Add(1)
		go func() {
			defer func() { doneCh <- struct{}{} }()
			defer wg.Done()
			ge, e := f.getAllTags(ctx, findKyouContext)
			if e != nil {
				e = fmt.Errorf("error at get all tags: %w", e)
			} else {
				slog.Log(ctx, gkill_log.Trace, "finish getAllTags", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
			}
			errch <- e
			gkillErrch <- ge
		}()

		wg.Add(1)
		go func() {
			defer func() { doneCh <- struct{}{} }()
			defer wg.Done()
			ge, e := f.getAllHideTagsWhenUnChecked(ctx, findKyouContext, userID, device)
			if e != nil {
				e = fmt.Errorf("error at get hide tags when unchecked tags: %w", e)
			} else {
				slog.Log(ctx, gkill_log.Trace, "finish getAllHideTagsWhenUnChecked", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
			}
			errch <- e
			gkillErrch <- ge
		}()

		wg.Add(1)
		go func() {
			defer func() { doneCh <- struct{}{} }()
			defer wg.Done()
			ge, e := f.findTags(ctx, findKyouContext)
			if e != nil {
				e = fmt.Errorf("error at find tags: %w", e)
			} else {
				slog.Log(ctx, gkill_log.Trace, "finish findTags", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
			}
			errch <- e
			gkillErrch <- ge
		}()
	}

	// テキスト取得
	wg.Add(1)
	go func() {
		defer func() { doneCh <- struct{}{} }()
		defer wg.Done()
		ge, e := f.findTexts(ctx, findKyouContext)
		if e != nil {
			e = fmt.Errorf("error at find texts: %w", e)
		} else {
			slog.Log(ctx, gkill_log.Trace, "finish findTexts", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
		}
		errch <- e
		gkillErrch <- ge
	}()

	if findQuery.UseTimeIs {
		wg.Add(1)
		go func() {
			defer func() { doneCh <- struct{}{} }()
			defer wg.Done()
			ge, e := f.findTimeIsTexts(ctx, findKyouContext)
			if e != nil {
				e = fmt.Errorf("error at find timeis texts: %w", e)
			} else {
				slog.Log(ctx, gkill_log.Trace, "finish findTimeIsTexts", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
			}
			errch <- e
			gkillErrch <- ge
		}()

		wg.Add(1)
		go func() {
			defer func() { doneCh <- struct{}{} }()
			defer wg.Done()
			ge, e := f.findTimeIsTags(ctx, findKyouContext)
			if e != nil {
				e = fmt.Errorf("error at find timeis tags: %w", e)
			} else {
				slog.Log(ctx, gkill_log.Trace, "finish findTimeIsTags", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
			}
			errch <- e
			gkillErrch <- ge
		}()
	}

	// タグなどの取得待ち
	gkillErr, err = catchErrFunc()
	if err != nil {
		return nil, gkillErr, err
	}
	wg.Wait()

	// TimeIs取得
	if findQuery.UseTimeIs {
		gkillErr, err = f.findTimeIs(ctx, findKyouContext)
		if err != nil {
			err = fmt.Errorf("error at find timeis: %w", err)
			return nil, gkillErr, err
		}
		slog.Log(ctx, gkill_log.Trace, "finish findTimeIs", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)

		if !(findQuery.UseTimeIsTags) || findQuery.TimeIsTags == nil {
			gkillErr, err = f.getMatchHideTagsWhenUnckedTimeIs(ctx, findKyouContext)
			if err != nil {
				err = fmt.Errorf("error at get match hide tags when unchecked timeis: %w", err)
				return nil, gkillErr, err
			}
			slog.Log(ctx, gkill_log.Trace, "finish getMatchHideTagsWhenUnckedTimeIs", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
		}

		gkillErr, err = f.filterTagsTimeIs(ctx, findKyouContext)
		if err != nil {
			err = fmt.Errorf("error at filter tags timeis: %w", err)
			return nil, gkillErr, err
		}
		slog.Log(ctx, gkill_log.Trace, "finish filterTagsTimeIs", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
	}

	gkillErr, err = f.findKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at find kyous: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish findKyous", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)

	gkillErr, err = f.sortAndTrimKyousMap(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at sort and trim kyousMap: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish findKyous", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)

	gkillErr, err = f.filterMiForMi(ctx, findKyouContext) //miの場合のみ
	if err != nil {
		err = fmt.Errorf("error at filter mi for mi: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish filterMiForMi", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
	if findQuery.UseTags {
		gkillErr, err = f.getMatchHideTagsWhenUnckedKyou(ctx, findKyouContext)
		if err != nil {
			err = fmt.Errorf("error at get match hide tags when unchecked timeis: %w", err)
			return nil, gkillErr, err
		}
		slog.Log(ctx, gkill_log.Trace, "finish getMatchHideTagsWhenUnckedKyou", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
	}
	if findQuery.UseTags {
		gkillErr, err = f.filterTagsKyous(ctx, findKyouContext)
		if err != nil {
			err = fmt.Errorf("error at filter tags kyous: %w", err)
			return nil, gkillErr, err
		}
		slog.Log(ctx, gkill_log.Trace, "finish filterTagsKyous", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
	}
	gkillErr, err = f.filterPlaingTimeIsKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at filter plaing time is kyous: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish filterPlaingTimeIsKyous", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
	gkillErr, err = f.filterLocationKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at filter location kyous: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish filterLocationKyous", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)
	gkillErr, err = f.filterImageKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at filter image kyous: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish filterImageKyous", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)

	slog.Log(ctx, gkill_log.Trace, "finish waitLatestDataWg", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)

	gkillErr, err = f.replaceLatestKyouInfos(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at replace latest kyou infos: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish replaceLatestKyouInfos", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)

	for _, kyous := range findKyouContext.MatchKyousCurrent {
		findKyouContext.ResultKyous = append(findKyouContext.ResultKyous, kyous...)
	}

	gkillErr, err = f.overrideKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at override kyous: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish overrideKyous", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)

	gkillErr, err = f.sortResultKyous(ctx, findKyouContext)
	if err != nil {
		err = fmt.Errorf("error at sort result kyous: %w", err)
		return nil, gkillErr, err
	}
	slog.Log(ctx, gkill_log.Trace, "finish sortResultKyous", "CurrentMatchKyous", findKyouContext.MatchKyousCurrent)

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

	return nil, nil
}

func (f *FindFilter) selectMatchRepsFromQuery(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	repositories := findCtx.Repositories

	// Step1: タイプ系フィルタ（ForMi / IsImageOnly / UsePlaing / UseRepTypes）で候補repを構築する
	// UseRepsの値に関わらず先に評価することで、UseRepTypesがUseRepsに依存していたバグを修正する
	typeMatchReps := []reps.Repository{}
	hasTypeFilter := findCtx.ParsedFindQuery.ForMi ||
		findCtx.ParsedFindQuery.IsImageOnly ||
		findCtx.ParsedFindQuery.UsePlaing ||
		findCtx.ParsedFindQuery.UseRepTypes

	if findCtx.ParsedFindQuery.ForMi {
		// ForMiだったらMi以外は無視する
		for _, rep := range repositories.MiReps {
			typeMatchReps = append(typeMatchReps, rep)
		}
	} else if findCtx.ParsedFindQuery.IsImageOnly {
		// ImageOnlyだったらIDFRep以外は無視する
		for _, rep := range repositories.IDFKyouReps {
			typeMatchReps = append(typeMatchReps, rep)
		}
	} else if findCtx.ParsedFindQuery.UsePlaing {
		// PlaingだったらTimeIsRep以外は無視する
		for _, rep := range repositories.TimeIsReps {
			typeMatchReps = append(typeMatchReps, rep)
		}
	} else if findCtx.ParsedFindQuery.UseRepTypes {
		// RepType指定の場合、指定以外は除外する
		for _, repType := range findCtx.ParsedFindQuery.RepTypes {
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
	}

	// Step2: タイプフィルタもUseRepsも指定なし → 全repをそのまま追加して終了
	if !hasTypeFilter && !findCtx.ParsedFindQuery.UseReps {
		for _, rep := range repositories.Reps {
			rep := rep
			repName, err := rep.GetRepName(ctx)
			if err != nil {
				return nil, err
			}
			if _, exist := findCtx.MatchReps[repName]; !exist {
				findCtx.MatchReps[repName] = rep
			}
		}
		return nil, nil
	}

	// タイプフィルタなし（UseRepsのみ指定）→ 全repを候補にする
	if !hasTypeFilter {
		typeMatchReps = append(typeMatchReps, repositories.Reps...)
	}

	// Step3: UseRepsなし → typeMatchReps を全てMatchRepsへ追加して終了
	if !findCtx.ParsedFindQuery.UseReps {
		for _, matchRep := range typeMatchReps {
			repImpls, err := matchRep.UnWrap()
			if err != nil {
				return nil, err
			}
			for _, repImpl := range repImpls {
				repName, err := repImpl.GetRepName(ctx)
				if err != nil {
					return nil, err
				}
				if _, exist := findCtx.MatchReps[repName]; !exist {
					findCtx.MatchReps[repName] = repImpl
				}
			}
		}
		return nil, nil
	}

	// Step4: UseRepsあり → typeMatchRepsをrep名でさらにフィルタ
	targetRepNames := []string{}
	if findCtx.ParsedFindQuery.Reps != nil {
		targetRepNames = findCtx.ParsedFindQuery.Reps
	}

	for _, matchRep := range typeMatchReps {
		repImpls, err := matchRep.UnWrap()
		if err != nil {
			return nil, err
		}
		for _, repImpl := range repImpls {
			repName, err := repImpl.GetRepName(ctx)
			if err != nil {
				return nil, err
			}

		rep_search:
			for _, targetRepName := range targetRepNames {
				if targetRepName == repName {
					if _, exist := findCtx.MatchReps[repName]; !exist {
						findCtx.MatchReps[repName] = repImpl
						continue rep_search
					}
				}
			}
		}
	}
	return nil, nil
}

func (f *FindFilter) updateCache(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	err := findCtx.Repositories.UpdateCache(ctx)
	if err != nil {
		err = fmt.Errorf("error at update repositories cache: %w", err)
		return nil, err
	}
	findCtx.ParsedFindQuery.UpdateCache = false
	return nil, nil
}

func (f *FindFilter) getAllTags(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	// 全タグ取得用検索クエリ
	findTagsQuery := &find.FindQuery{IsDeleted: false}

	allTagsList, err := collectFromRepos([]reps.TagRepository(findCtx.Repositories.TagReps), func(tagRep reps.TagRepository) ([]reps.Tag, error) {
		return tagRep.FindTags(ctx, findTagsQuery)
	})
	if err != nil {
		return nil, fmt.Errorf("error at get all tags: %w", err)
	}

	// Tag集約
	for _, tag := range allTagsList {
		upsertIfNewer(findCtx.AllTags, tag.ID, tag, func(t reps.Tag) time.Time { return t.UpdateTime })
	}

	// タグの対象をリスト
	for _, tag := range findCtx.AllTags {
		if !findCtx.isLatestData(tag.ID, tag.UpdateTime) {
			continue
		}
		if tag.IsDeleted {
			continue
		}
		findCtx.RelatedTagIDs[tag.TargetID] = struct{}{}
	}

	return nil, nil
}

func (f *FindFilter) getAllHideTagsWhenUnChecked(ctx context.Context, findCtx *FindKyouContext, userID string, device string) ([]*message.GkillError, error) {
	hideTagNames := []string{}
	if findCtx.ParsedFindQuery.HideTags != nil {
		hideTagNames = append(hideTagNames, findCtx.ParsedFindQuery.HideTags...)
	}

	for _, hideTagName := range hideTagNames {
		hideTagsInReps, err := findCtx.Repositories.TagReps.GetTagsByTagName(ctx, hideTagName)
		if err != nil {
			err = fmt.Errorf("error at get tags by tagname tagname=%s: %w", hideTagName, err)
			return nil, err
		}
		for _, hideTag := range hideTagsInReps {
			if !findCtx.isLatestData(hideTag.ID, hideTag.UpdateTime) {
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

func (f *FindFilter) getMatchHideTagsWhenUnchecked(
	findCtx *FindKyouContext,
	checkedTagNames []string,
	output map[string]reps.Tag,
) {
	for _, hideTag := range findCtx.AllHideTagsWhenUnchecked {
		if !containsString(checkedTagNames, hideTag.Tag) {
			output[hideTag.ID] = hideTag
		}
	}
}

func (f *FindFilter) getMatchHideTagsWhenUnckedKyou(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if findCtx.ParsedFindQuery.Tags == nil {
		return nil, nil
	}
	f.getMatchHideTagsWhenUnchecked(findCtx, findCtx.ParsedFindQuery.Tags, findCtx.MatchHideTagsWhenUncheckedKyou)
	return nil, nil
}

func (f *FindFilter) getMatchHideTagsWhenUnckedTimeIs(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	f.getMatchHideTagsWhenUnchecked(findCtx, findCtx.ParsedFindQuery.TimeIsTags, findCtx.MatchHideTagsWhenUncheckedTimeIs)
	return nil, nil
}

func (f *FindFilter) findTimeIsTags(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	// タグを使わない場合はnil
	if !(findCtx.ParsedFindQuery.UseTimeIsTags) {
		return nil, nil
	}

	for _, tagName := range findCtx.ParsedFindQuery.TimeIsTags {
		matchTags, err := findCtx.Repositories.TagReps.GetTagsByTagName(ctx, tagName)
		if err != nil {
			err = fmt.Errorf("error at get tags by name %s: %w", tagName, err)
			return nil, err
		}
		for _, tag := range matchTags {
			if !findCtx.isLatestData(tag.ID, tag.UpdateTime) {
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

func (f *FindFilter) findTags(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {

	query := &find.FindQuery{
		// IsDeleted: false, // TagReps.FindTags内に考慮があるため削除
		UseWords: true,
		Words:    findCtx.ParsedFindQuery.Tags,
		WordsAnd: false,
	}
	matchTags, err := findCtx.Repositories.TagReps.FindTags(ctx, query)
	if err != nil {
		err = fmt.Errorf("error at get tags by name %#v: %w", findCtx.ParsedFindQuery.Words, err)
		return nil, err
	}
	for _, tag := range matchTags {
		if !findCtx.isLatestData(tag.ID, tag.UpdateTime) {
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

	matchTextFindByIDQuery := &find.FindQuery{
		UseIDs: true,
		IDs:    targetIDs,
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
	textMatchKyousMap := map[string][]reps.Kyou{}
	if len(targetIDs) != 0 {
		textMatchKyousMap, err = matchReps.FindKyous(ctx, matchTextFindByIDQuery)
		if err != nil {
			return nil, err
		}
	}
	for id, textMatchKyous := range textMatchKyousMap {
		if _, exist := kyousMap[id]; !exist {
			kyousMap[id] = []reps.Kyou{}
		}
		kyousMap[id] = append(kyousMap[id], textMatchKyous...)
	}

	// 削除隅のものは消す
	deleteTargetIDs := []string{}
	for id, kyous := range kyousMap {
		var latestKyou reps.Kyou
		for _, kyou := range kyous {
			if kyou.UpdateTime.After(latestKyou.UpdateTime) {
				latestKyou = kyou
			}
		}
		if latestKyou.IsDeleted {
			deleteTargetIDs = append(deleteTargetIDs, id)
		}
	}
	for _, deleteTargetID := range deleteTargetIDs {
		delete(kyousMap, deleteTargetID)
	}
	findCtx.MatchKyousCurrent = kyousMap
	return nil, nil
}

func (f *FindFilter) sortAndTrimKyousMap(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	resultKyous := map[string][]reps.Kyou{}

	deleteTargetKyouIDs := []string{}
	for id, kyous := range findCtx.MatchKyousCurrent {
		if len(kyous) == 0 {
			deleteTargetKyouIDs = append(deleteTargetKyouIDs, id)
			continue
		}

		trimedKyousMap := map[int64]reps.Kyou{}
		for _, kyou := range kyous {
			if findCtx.ParsedFindQuery.UseCalendar {
				if (findCtx.ParsedFindQuery.CalendarStartDate != nil && kyou.RelatedTime.Before(*findCtx.ParsedFindQuery.CalendarStartDate)) ||
					(findCtx.ParsedFindQuery.CalendarEndDate != nil && kyou.RelatedTime.After(*findCtx.ParsedFindQuery.CalendarEndDate)) {
					continue
				}
			}
			trimedKyousMap[kyou.RelatedTime.Unix()] = kyou
		}

		sortedKyous := make([]reps.Kyou, 0, len(trimedKyousMap))
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

	if (findCtx.ParsedFindQuery.UsePlaing) || (findCtx.ParsedFindQuery.ForMi) {
		for id := range resultKyous {
			resultKyous[id] = []reps.Kyou{resultKyous[id][0]}
		}
	}

	findCtx.MatchKyousCurrent = resultKyous
	return nil, nil
}

func (f *FindFilter) filterMiForMi(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if !(findCtx.ParsedFindQuery.ForMi) {
		return nil, nil
	}

	// Miを取得位する
	// 作成日時以外の条件でmiを取得する。その後、作成日時で取得して追加する。
	allMis := map[string]reps.Mi{}
	withoutCreatedMiFindQuery := *findCtx.ParsedFindQuery
	withoutCreatedMiFindQuery.IncludeCreateMi = false
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
	targetMis := []reps.Mi{}
	for _, mi := range allMis {
		switch string(findCtx.ParsedFindQuery.MiCheckState) {
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

	// 対象MiのKyouのみを抽出する
	filteredKyous := map[string][]reps.Kyou{}
	for _, mi := range targetMis {
		kyous, exist := findCtx.MatchKyousCurrent[mi.ID]
		if exist {
			filteredKyous[mi.ID] = kyous
		}
	}

	findCtx.MatchMisAtFilterMi = allMis
	findCtx.MatchKyousCurrent = filteredKyous
	return nil, nil
}

func (f *FindFilter) filterTagsKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if !(findCtx.ParsedFindQuery.TagsAnd) {
		// ORの場合のフィルタリング処理

		// タグ対象Kyouリスト
		matchOrTagKyousMap := map[string][]reps.Kyou{}
		for _, tag := range findCtx.MatchTags {
			kyou, exist := findCtx.MatchKyousCurrent[tag.TargetID]
			if !exist {
				continue
			}
			matchOrTagKyousMap[tag.TargetID] = kyou
		}

		// タグ無しKyouリスト
		noTagKyous := map[string][]reps.Kyou{}
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
			tags = findCtx.ParsedFindQuery.Tags
		}

		for _, tag := range tags {
			if tag == NoTags {
				existNoTags = true
				break
			}
		}

		// タグフィルタしたものを収める
		filteredByTags := map[string][]reps.Kyou{}
		for id, kyou := range matchOrTagKyousMap {
			filteredByTags[id] = kyou
		}
		if existNoTags {
			for id, kyou := range noTagKyous {
				filteredByTags[id] = kyou
			}
		}

		// 非表示タグの対象を消す
		for _, hideTag := range findCtx.MatchHideTagsWhenUncheckedKyou {
			delete(filteredByTags, hideTag.TargetID)
		}

		findCtx.MatchKyousCurrent = filteredByTags
	} else if findCtx.ParsedFindQuery.TagsAnd {
		// ANDの場合のフィルタリング処理
		tagNameMap := map[string]map[string][]reps.Kyou{} // map[タグ名][kyou.ID（tagTargetID）] = reps.kyou

		for _, tag := range findCtx.MatchTags {
			isTagInQuery := false
			for _, tagName := range findCtx.ParsedFindQuery.Tags {
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
				tagNameMap[tag.Tag] = map[string][]reps.Kyou{}
			}

			tagNameMap[tag.Tag][tag.TargetID] = kyous
		}

		// タグ無しの情報もtagNameMapにいれる
		existNoTags := false
		tags := []string{}
		if findCtx.ParsedFindQuery.Tags != nil {
			tags = findCtx.ParsedFindQuery.Tags
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
						tagNameMap[NoTags] = map[string][]reps.Kyou{}
					}
					tagNameMap[NoTags][id] = kyous
				}
			}
		}

		// tagNameMapの全部のタグ名に存在するKyouだけを抽出
		hasAllMatchTagsKyousMap := map[string]map[string][]reps.Kyou{}
		// 初回は全部いれる
		for tagName, kyouIDMap := range tagNameMap {
			for kyouID, kyous := range kyouIDMap {
				if _, exist := hasAllMatchTagsKyousMap[tagName]; !exist {
					hasAllMatchTagsKyousMap[tagName] = map[string][]reps.Kyou{}
				}
				hasAllMatchTagsKyousMap[tagName][kyouID] = kyous
			}
		}
		for tagName := range tagNameMap {
			matchThisLoopKyousMap := map[string]map[string][]reps.Kyou{}
			// 初回ループ以外は、
			// 以前のタグにマッチしたもの（hasAllMatchTagsKyous）にあり、かつ
			// 今回のタグにマッチしたもの　をいれる。
			if _, exist := matchThisLoopKyousMap[tagName]; !exist {
				matchThisLoopKyousMap[tagName] = map[string][]reps.Kyou{}
			}

			beforeMatchKyous := map[string][]reps.Kyou{}
			for _, kyous := range hasAllMatchTagsKyousMap {
				for kyouID, kyou := range kyous {
					beforeMatchKyous[kyouID] = kyou
				}
			}
			currentMatchKyous := map[string][]reps.Kyou{}
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

		filteredByTags := map[string][]reps.Kyou{}
		for _, matchTagsKyousMap := range hasAllMatchTagsKyousMap {
			for kyouID, kyous := range matchTagsKyousMap {
				filteredByTags[kyouID] = kyous
			}
		}

		// 非表示タグの対象を消す
		for _, hideTag := range findCtx.MatchHideTagsWhenUncheckedKyou {
			delete(filteredByTags, hideTag.TargetID)
		}

		findCtx.MatchKyousCurrent = filteredByTags
	}

	return nil, nil
}

func (f *FindFilter) filterTagsTimeIs(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if !(findCtx.ParsedFindQuery.UseTimeIsTags) {
		for _, timeis := range findCtx.MatchTimeIssAtFindTimeIs {
			if timeis.IsDeleted {
				continue
			}
			findCtx.MatchTimeIssAtFilterTags[timeis.ID] = timeis
		}
		return nil, nil
	}
	if findCtx.ParsedFindQuery.TimeIsTags != nil && !(findCtx.ParsedFindQuery.TimeIsTagsAnd) {
		// ORの場合のフィルタリング処理

		// タグ対象Kyouリスト
		matchOrTagTimeIss := map[string]reps.TimeIs{}
		for _, tag := range findCtx.MatchTimeIsTags {
			matchTimeis, exist := findCtx.MatchTimeIssAtFindTimeIs[tag.TargetID]
			if !exist {
				continue
			}
			matchOrTagTimeIss[matchTimeis.ID] = matchTimeis
		}

		// タグ無しKyouリスト
		noTagTimeIss := map[string]reps.TimeIs{}
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
			tags = findCtx.ParsedFindQuery.TimeIsTags
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

	} else if findCtx.ParsedFindQuery.TimeIsTags != nil && (findCtx.ParsedFindQuery.TimeIsTagsAnd) {
		// ANDの場合のフィルタリング処理

		tagNameMap := map[string]map[string]reps.TimeIs{} // map[タグ名][kyou.ID（tagTargetID）] = reps.TimeIs

		for _, tag := range findCtx.MatchTimeIsTags {
			isTagInQuery := false
			for _, tagName := range findCtx.ParsedFindQuery.TimeIsTags {
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
				tagNameMap[tag.Tag] = map[string]reps.TimeIs{}
			}

			tagNameMap[tag.Tag][timeis.ID] = timeis
		}

		// タグ無しの情報もtagNameMapにいれる
		existNoTags := false
		tags := []string{}
		if findCtx.ParsedFindQuery.TimeIsTags != nil {
			tags = findCtx.ParsedFindQuery.TimeIsTags
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
		hasAllMatchTagsTimeIssMap := map[string]reps.TimeIs{}
		index := 0
		for _, timeisIDMap := range tagNameMap {
			switch index {
			case 0:
				// 初回ループは全部いれる
				for _, timeis := range timeisIDMap {
					hasAllMatchTagsTimeIssMap[timeis.ID] = timeis
				}
			default:
				matchThisLoopTimeIssMap := map[string]reps.TimeIs{}
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
	if !(findCtx.ParsedFindQuery.UseTimeIs) {
		return nil, nil
	}

	filteredByTimeIs := map[string][]reps.Kyou{}
	for _, timeis := range findCtx.MatchTimeIssAtFilterTags {
		for id, kyous := range findCtx.MatchKyousCurrent {
			if (timeis.EndTime != nil && kyous[0].RelatedTime.After(timeis.StartTime) && kyous[0].RelatedTime.Before(*timeis.EndTime)) || (timeis.EndTime == nil && kyous[0].RelatedTime.After(timeis.StartTime)) {
				filteredByTimeIs[id] = kyous
			}
		}
	}
	findCtx.MatchKyousCurrent = filteredByTimeIs
	return nil, nil
}

func (f *FindFilter) findTimeIs(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if !findCtx.ParsedFindQuery.UseTimeIs {
		return nil, nil
	}

	// 対象TimeIs取得用検索クエリ
	timeisFindKyouQuery := &find.FindQuery{
		UseWords:          true,
		Words:             findCtx.ParsedFindQuery.TimeIsWords,
		NotWords:          findCtx.ParsedFindQuery.TimeIsNotWords,
		WordsAnd:          findCtx.ParsedFindQuery.TimeIsWordsAnd,
		UseCalendar:       findCtx.ParsedFindQuery.UseCalendar,
		CalendarStartDate: findCtx.ParsedFindQuery.CalendarStartDate,
		CalendarEndDate:   findCtx.ParsedFindQuery.CalendarEndDate,
		IncludeEndTimeIs:  true,
	}

	// text検索用クエリ
	targetIDs := []string{}
	for _, text := range findCtx.MatchTimeIsTexts {
		targetIDs = append(targetIDs, text.TargetID)
	}
	matchTextFindByIDQuery := &find.FindQuery{
		UseIDs: true,
		IDs:    targetIDs,
	}

	allTimeIss, err := collectFromRepos([]reps.TimeIsRepository(findCtx.Repositories.TimeIsReps), func(rep reps.TimeIsRepository) ([]reps.TimeIs, error) {
		timeiss, err := rep.FindTimeIs(ctx, timeisFindKyouQuery)
		if err != nil {
			return nil, err
		}
		if len(targetIDs) != 0 {
			textMatchTimeiss, err := rep.FindTimeIs(ctx, matchTextFindByIDQuery)
			if err != nil {
				return nil, err
			}
			timeiss = append(timeiss, textMatchTimeiss...)
		}
		return timeiss, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error at find timeiss: %w", err)
	}

	// TimeIs集約
	for _, timeis := range allTimeIss {
		if !findCtx.isLatestData(timeis.ID, timeis.UpdateTime) {
			continue
		}
		upsertIfNewer(findCtx.MatchTimeIssAtFindTimeIs, timeis.ID, timeis, func(t reps.TimeIs) time.Time { return t.UpdateTime })
	}

	deletedIDs := []string{}
	for _, timeis := range findCtx.MatchTimeIssAtFindTimeIs {
		if timeis.IsDeleted {
			deletedIDs = append(deletedIDs, timeis.ID)
		}
	}
	for _, deletedID := range deletedIDs {
		delete(findCtx.MatchTimeIssAtFindTimeIs, deletedID)
	}

	return nil, nil
}

func (f *FindFilter) filterLocationKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if !findCtx.ParsedFindQuery.UseMap {
		return nil, nil
	}

	matchKyous := map[string][]reps.Kyou{}

	// 開始日を取得
	startTime := findCtx.ParsedFindQuery.CalendarStartDate
	endTime := findCtx.ParsedFindQuery.CalendarEndDate

	// radius, latitude, longitudeを取得
	var radius float64
	var latitude float64
	var longitude float64

	if findCtx.ParsedFindQuery.MapRadius != 0 {
		radius = findCtx.ParsedFindQuery.MapRadius / 1000
	}
	latitude = findCtx.ParsedFindQuery.MapLatitude
	longitude = findCtx.ParsedFindQuery.MapLongitude

	// 日付のnil解決 もしくは全部の日付
	isAllDays := false
	if (startTime != nil && endTime == nil) && findCtx.ParsedFindQuery.UseCalendar {
		s := time.Time(*startTime)
		e := time.Time(*startTime).Add(time.Hour*23 + time.Minute*59 + time.Second*59)
		startTime = &s
		endTime = &e
	} else if (startTime != nil && endTime != nil) && findCtx.ParsedFindQuery.UseCalendar {
		s := time.Time(*startTime)
		e := time.Time(*endTime).Add(time.Hour*23 + time.Minute*59 + time.Second*59)
		startTime = &s
		endTime = &e
	} else if (startTime == nil && endTime == nil) || (!findCtx.ParsedFindQuery.UseCalendar) {
		isAllDays = true
	}
	// GPSLogを取得する
	matchGPSLogs, err := collectFromRepos([]reps.GPSLogRepository(findCtx.Repositories.GPSLogReps), func(rep reps.GPSLogRepository) ([]reps.GPSLog, error) {
		if isAllDays {
			return rep.GetAllGPSLogs(ctx)
		}
		return rep.GetGPSLogs(ctx, startTime, endTime)
	})
	if err != nil {
		return nil, fmt.Errorf("error at filter gpslogs: %w", err)
	}

	// 並び替え
	sort.Slice(matchGPSLogs, func(i, j int) bool { return matchGPSLogs[i].RelatedTime.Before(matchGPSLogs[j].RelatedTime) })

	// 該当する時間を出す
	matchGPSLogSetList := [][]reps.GPSLog{}

	preTrue := false // 一つ前の時間でtrueだった
	for i := range matchGPSLogs {
		if preTrue {
			matchGPSLogSetList = append(matchGPSLogSetList, []reps.GPSLog{
				matchGPSLogs[i-1],
				matchGPSLogs[i],
			})
		}

		if calcDistanceKm(latitude, longitude, matchGPSLogs[i].Latitude, matchGPSLogs[i].Longitude) <= radius {
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

	findCtx.MatchKyousCurrent = matchKyous
	return nil, nil
}
func (f *FindFilter) overrideKyous(_ context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if !findCtx.ParsedFindQuery.ForMi {
		// kyou検索の場合は何もしない
		return nil, nil
	}
	// miの場合は
	// 表示したとき、指定日時か作成日時かわかるようにDataTypeを上書きする
	for _, mi := range findCtx.MatchMisAtFilterMi {
		kyous, exist := findCtx.MatchKyousCurrent[mi.ID]
		if exist {
			kyous[0].DataType = mi.DataType
			if string(findCtx.ParsedFindQuery.MiSortType) == string(find.CreateTime) {
				kyous[0].DataType = "mi_create"
				kyous[0].RelatedTime = mi.CreateTime
			} else if string(findCtx.ParsedFindQuery.MiSortType) == string(find.EstimateStartTime) && mi.EstimateStartTime != nil {
				kyous[0].DataType = "mi_start"
				kyous[0].RelatedTime = *mi.EstimateStartTime
			} else if string(findCtx.ParsedFindQuery.MiSortType) == string(find.EstimateEndTime) && mi.EstimateEndTime != nil {
				kyous[0].DataType = "mi_end"
				kyous[0].RelatedTime = *mi.EstimateEndTime
			} else if string(findCtx.ParsedFindQuery.MiSortType) == string(find.LimitTime) && mi.LimitTime != nil {
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
	if !findCtx.ParsedFindQuery.ForMi {
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
	sortType := findCtx.ParsedFindQuery.MiSortType
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

func (f *FindFilter) findTexts(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	return f.findTextsGeneric(ctx, findCtx,
		findCtx.ParsedFindQuery.Words, findCtx.ParsedFindQuery.NotWords, findCtx.ParsedFindQuery.WordsAnd,
		findCtx.MatchTexts)
}

func (f *FindFilter) findTextsGeneric(
	ctx context.Context, findCtx *FindKyouContext,
	words, notWords []string, wordsAnd bool,
	targetMap map[string]reps.Text,
) ([]*message.GkillError, error) {
	// words, notWordsをパースする
	w := []string{}
	nw := []string{}
	if words != nil {
		w = words
	}
	if notWords != nil {
		nw = notWords
	}

	findTextsQuery := &find.FindQuery{
		UseWords: true,
		Words:    w,
		NotWords: nw,
		WordsAnd: wordsAnd,
	}

	repos := make([]reps.TextRepository, 0, len(findCtx.Repositories.TextReps))
	for _, rep := range findCtx.Repositories.TextReps {
		repos = append(repos, rep)
	}

	allTexts, err := collectFromRepos(repos, func(textRep reps.TextRepository) ([]reps.Text, error) {
		return textRep.FindTexts(ctx, findTextsQuery)
	})
	if err != nil {
		return nil, fmt.Errorf("error at find texts: %w", err)
	}

	// Text集約
	for _, text := range allTexts {
		if !findCtx.isLatestData(text.ID, text.UpdateTime) {
			continue
		}
		upsertIfNewer(targetMap, text.ID, text, func(t reps.Text) time.Time { return t.UpdateTime })
	}

	return nil, nil
}

func (f *FindFilter) filterImageKyous(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	if !(findCtx.ParsedFindQuery.IsImageOnly) {
		return nil, nil
	}

	filterdImageKyous := map[string][]reps.Kyou{}
	for id, kyous := range findCtx.MatchKyousCurrent {
		if kyous[0].IsImage || kyous[0].IsVideo {
			filterdImageKyous[id] = kyous
		}
	}
	findCtx.MatchKyousCurrent = filterdImageKyous
	return nil, nil
}

func (f *FindFilter) findTimeIsTexts(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	return f.findTextsGeneric(ctx, findCtx,
		findCtx.ParsedFindQuery.TimeIsWords, findCtx.ParsedFindQuery.TimeIsNotWords, findCtx.ParsedFindQuery.TimeIsWordsAnd,
		findCtx.MatchTimeIsTexts)
}
func (f *FindFilter) replaceLatestKyouInfos(ctx context.Context, findCtx *FindKyouContext) ([]*message.GkillError, error) {
	latestKyousMap := map[string][]reps.Kyou{}

	for id, currentKyou := range findCtx.MatchKyousCurrent {
		if findCtx.DisableLatestDataRepositoryCache {
			sort.Slice(currentKyou, func(i, j int) bool { return currentKyou[i].UpdateTime.After(currentKyou[j].UpdateTime) })

			// UsePlaing時はLatestDataRepositoryAddressと一致するもののみ残す
			if findCtx.ParsedFindQuery.UsePlaing {
				latestData, exist := (findCtx.Repositories.LatestDataRepositoryAddresses)[id]
				if !exist || !currentKyou[0].UpdateTime.Equal(latestData.DataUpdateTime) {
					continue
				}
			}

			// TimeIsやMiは複数Kyou（start/end等）を持つのでそのまま保持する
			isMiData := strings.HasPrefix(currentKyou[0].DataType, "mi") && findCtx.ParsedFindQuery.ForMi
			isTimeIsData := strings.HasPrefix(currentKyou[0].DataType, "timeis")
			if isTimeIsData || isMiData {
				latestKyousMap[id] = currentKyou
			} else {
				latestKyousMap[id] = []reps.Kyou{currentKyou[0]}
			}
			continue
		}

		latestData, exist := (findCtx.Repositories.LatestDataRepositoryAddresses)[id]
		if !exist {
			continue
		}

		isMiData := strings.HasPrefix(currentKyou[0].DataType, "mi") && findCtx.ParsedFindQuery.ForMi
		isTimeIsData := strings.HasPrefix(currentKyou[0].DataType, "timeis")
		isUsePlaing := findCtx.ParsedFindQuery.UsePlaing

		// すでに最新が入っていそうだったらそのままいれる RepNameは運用都合でチェックしない
		// Miもそのままいれる
		if ((currentKyou[0].UpdateTime.Equal(latestData.DataUpdateTime) || isMiData || isTimeIsData) && !isUsePlaing) ||
			((currentKyou[0].UpdateTime.Equal(latestData.DataUpdateTime) || isTimeIsData) && isUsePlaing) {
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
		latestKyousMap[id] = []reps.Kyou{*latestKyou}
	}

	// miの場合は最新以外消す
	isForMi := findCtx.ParsedFindQuery.ForMi
	if isForMi {
		for id, kyous := range latestKyousMap {
			sort.Slice(kyous, func(i, j int) bool {
				return kyous[i].UpdateTime.After(kyous[j].UpdateTime)
			})
			latestKyousMap[id] = []reps.Kyou{kyous[0]}
		}
	}

	findCtx.MatchKyousCurrent = latestKyousMap
	return nil, nil
}

