package reps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/account_state"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
)

type GkillRepositories struct {
	userID string

	Reps Repositories

	TagReps TagRepositories

	TextReps TextRepositories

	KmemoReps KmemoRepositories

	URLogReps URLogRepositories

	NlogReps NlogRepositories

	TimeIsReps TimeIsRepositories

	MiReps MiRepositories

	LantanaReps LantanaRepositories

	IDFKyouReps IDFKyouRepositories

	ReKyouReps ReKyouRepositories

	GitCommitLogReps GitCommitLogRepositories

	GPSLogReps GPSLogRepositories

	WriteTagRep TagRepository

	WriteTextRep TextRepository

	WriteKmemoRep KmemoRepository

	WriteURLogRep URLogRepository

	WriteNlogRep NlogRepository

	WriteTimeIsRep TimeIsRepository

	WriteMiRep MiRepository

	WriteLantanaRep LantanaRepository

	WriteIDFKyouRep IDFKyouRepository

	WriteReKyouRep ReKyouRepository

	WriteGPSLogRep GPSLogRepository

	LatestDataRepositoryAddressDAO account_state.LatestDataRepositoryAddressDAO
}

// repsとLatestDataRepositoryAddressDAOのみ初期化済みのGkillRepositoriesを返す
func NewGkillRepositories(userID string) (*GkillRepositories, error) {
	configDBRootDir := os.ExpandEnv("$HOME/gkill/configs")
	err := os.MkdirAll(configDBRootDir, fs.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at create directory %s: %w", err)
		return nil, err
	}

	latestDataRepositoryAddressDAO, err := account_state.NewLatestDataRepositoryAddressSQLite3Impl(context.Background(), filepath.Join(configDBRootDir, fmt.Sprintf("latest_data_repository_address_%s.db", userID)))
	if err != nil {
		err = fmt.Errorf("error at get latest data repository address dao. user id = %s: %w", userID, err)
		return nil, err
	}

	return &GkillRepositories{
		Reps:                           Repositories{},
		userID:                         userID,
		LatestDataRepositoryAddressDAO: latestDataRepositoryAddressDAO,
	}, nil
}

func (g *GkillRepositories) GetUserID(ctx context.Context) (string, error) {
	return g.userID, nil
}

func (g *GkillRepositories) Close(ctx context.Context) error {
	for _, rep := range g.TagReps {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}
	for _, rep := range g.TextReps {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}
	for _, rep := range g.KmemoReps {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}
	for _, rep := range g.URLogReps {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}

	for _, rep := range g.NlogReps {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}
	for _, rep := range g.TimeIsReps {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}
	for _, rep := range g.MiReps {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}
	for _, rep := range g.LantanaReps {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}
	for _, rep := range g.IDFKyouReps {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}
	for _, rep := range g.ReKyouReps.ReKyouRepositories {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}
	for _, rep := range g.GitCommitLogReps {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}
	err := g.LatestDataRepositoryAddressDAO.Close(ctx)
	if err != nil {
		return err
	}

	/*
		for _, rep := range g.GPSLogReps {
			err := rep.Close(ctx)
			if err != nil {
				return err
			}
		}
	*/
	g.userID = ""

	return nil
}

func (g *GkillRepositories) FindKyous(ctx context.Context, query *find.FindQuery) ([]*Kyou, error) {
	matchKyous := map[string]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kyou, len(g.Reps))
	errch := make(chan error, len(g.Reps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.Reps {
		wg.Add(1)
		
		go func(rep Repository) {
			defer wg.Done()
			// jsonからパースする
			queryLatest := query

			// idsを指定されていなければ、最新であるもののIDのみを対象とする
			if query.IDs != nil {
				ids := *query.IDs
				repName, err := rep.GetRepName(ctx)
				if err != nil {
					err = fmt.Errorf("error at get rep name: %w", err)
					errch <- err
					return
				}

				latestDataRepositoryAddresses, err := g.LatestDataRepositoryAddressDAO.GetLatestDataRepositoryAddressesByRepName(ctx, repName)
				if err != nil {
					err = fmt.Errorf("error at get latest data repository addresses by rep name %s: %w", repName, err)
					errch <- err
					return
				}

				idsStrBuf := bytes.NewBufferString("")
				for _, latestDataRepositoryAddress := range latestDataRepositoryAddresses {
					ids = append(ids, latestDataRepositoryAddress.TargetID)
				}

				err = json.NewEncoder(idsStrBuf).Encode(ids)
				if err != nil {
					err = fmt.Errorf("error at latest ids in rep marshal json %#v: %w", ids, err)
					errch <- err
					return
				}
				trueValue := true

				query.IDs = &ids
				query.UseIDs = &trueValue

				queryLatest = query
			}

			matchKyousInRep, err := rep.FindKyous(ctx, queryLatest)
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
}

func (g *GkillRepositories) GetKyou(ctx context.Context, id string) (*Kyou, error) {
	matchKyou := &Kyou{}
	matchKyou = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Kyou, len(g.Reps))
	errch := make(chan error, len(g.Reps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.Reps {
		wg.Add(1)
		
		go func(rep Repository) {
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

func (g *GkillRepositories) UpdateCache(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	kyousCh := make(chan []*Kyou, len(g.Reps))
	tagsCh := make(chan []*Tag, len(g.TagReps))
	textsCh := make(chan []*Text, len(g.TextReps))
	errch := make(chan error, len(g.Reps))
	defer close(kyousCh)
	defer close(tagsCh)
	defer close(textsCh)
	defer close(errch)

	allKyous := []*Kyou{}
	allTags := []*Tag{}
	allTexts := []*Text{}

	// UpdateCache並列処理
	for _, rep := range g.Reps {
		wg.Add(1)
		
		go func(rep Repository) {
			defer wg.Done()
			err = rep.UpdateCache(ctx)
			if err != nil {
				errch <- err
				return
			}
		}(rep)
	}
	wg.Wait()

	// kyouを集める
	for _, rep := range g.Reps {
		wg.Add(1)
		
		go func(rep Repository) {
			defer wg.Done()
			repName, err := rep.GetRepName(ctx)
			if err != nil {
				err = fmt.Errorf("error at get rep name: %w", err)
				errch <- err
				return
			}

			reps := []string{repName}
			kyous, err := rep.FindKyous(ctx, &find.FindQuery{Reps: &reps})
			if err != nil {
				errch <- err
				return
			}
			kyousCh <- kyous
		}(rep)
	}

	// tagを集める
	for _, rep := range g.TagReps {
		wg.Add(1)
		
		go func(rep TagRepository) {
			defer wg.Done()
			repName, err := rep.GetRepName(ctx)
			if err != nil {
				err = fmt.Errorf("error at get rep name: %w", err)
				errch <- err
				return
			}

			reps := []string{repName}
			tags, err := rep.FindTags(ctx, &find.FindQuery{Reps: &reps})
			if err != nil {
				errch <- err
				return
			}
			tagsCh <- tags
		}(rep)
	}

	// textを集める
	for _, rep := range g.TextReps {
		wg.Add(1)
		
		go func(rep TextRepository) {
			defer wg.Done()
			repName, err := rep.GetRepName(ctx)
			if err != nil {
				err = fmt.Errorf("error at get rep name: %w", err)
				errch <- err
				return
			}

			reps := []string{repName}
			texts, err := rep.FindTexts(ctx, &find.FindQuery{Reps: &reps})
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
			err = fmt.Errorf("error at update cache: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return err
	}

	// kyou集約
kyousloop:
	for {
		select {
		case kyous := <-kyousCh:
			allKyous = append(allKyous, kyous...)
		default:
			break kyousloop
		}
	}

	// tag集約
tagsloop:
	for {
		select {
		case tags := <-tagsCh:
			allTags = append(allTags, tags...)
		default:
			break tagsloop
		}
	}

	// text集約
textsloop:
	for {
		select {
		case texts := <-textsCh:
			allTexts = append(allTexts, texts...)
		default:
			break textsloop
		}
	}

	// 最新のKyou, tag, textのみにする
	latestKyousMap := map[string]*Kyou{}
	for _, kyou := range allKyous {
		if existKyou, exist := latestKyousMap[kyou.ID]; exist {
			if kyou.UpdateTime.Before(existKyou.UpdateTime) {
				latestKyousMap[kyou.ID] = kyou
			}
		} else {
			latestKyousMap[kyou.ID] = kyou
		}
	}
	latestTagsMap := map[string]*Tag{}
	for _, tag := range allTags {
		if existTag, exist := latestTagsMap[tag.ID]; exist {
			if tag.UpdateTime.Before(existTag.UpdateTime) {
				latestTagsMap[tag.ID] = tag
			}
		} else {
			latestTagsMap[tag.ID] = tag
		}
	}
	latestTextsMap := map[string]*Text{}
	for _, text := range allTexts {
		if existText, exist := latestTextsMap[text.ID]; exist {
			if text.UpdateTime.Before(existText.UpdateTime) {
				latestTextsMap[text.ID] = text
			}
		} else {
			latestTextsMap[text.ID] = text
		}
	}

	// 最新のKyou, Tag, Textの状態をLatestDataRepositoryAddressにいれる
	latestDataRepositoryAddresses := []*account_state.LatestDataRepositoryAddress{}
	for _, kyou := range latestKyousMap {
		latestDataRepositoryAddress := &account_state.LatestDataRepositoryAddress{
			TargetID:                 kyou.ID,
			LatestDataRepositoryName: kyou.RepName,
			DataUpdateTime:           kyou.UpdateTime,
		}
		latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, latestDataRepositoryAddress)
	}
	for _, tag := range latestTagsMap {
		latestDataRepositoryAddress := &account_state.LatestDataRepositoryAddress{
			TargetID:                 tag.ID,
			LatestDataRepositoryName: tag.RepName,
			DataUpdateTime:           tag.UpdateTime,
		}
		latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, latestDataRepositoryAddress)
	}
	for _, text := range latestTextsMap {
		latestDataRepositoryAddress := &account_state.LatestDataRepositoryAddress{
			TargetID:                 text.ID,
			LatestDataRepositoryName: text.RepName,
			DataUpdateTime:           text.UpdateTime,
		}
		latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, latestDataRepositoryAddress)
	}

	_, err = g.LatestDataRepositoryAddressDAO.DeleteAllLatestDataRepositoryAddress(ctx)
	if err != nil {
		err = fmt.Errorf("error at delete all latest data repository address cache: %w", err)
		return err
	}

	_, err = g.LatestDataRepositoryAddressDAO.UpdateOrAddLatestDataRepositoryAddresses(ctx, latestDataRepositoryAddresses)
	if err != nil {
		err = fmt.Errorf("error at add latest data repository address cache: %w", err)
		return err
	}

	return nil
}

func (g *GkillRepositories) GetPath(ctx context.Context, id string) (string, error) {
	err := fmt.Errorf("not implements GetPath")
	return "", err
}

func (g *GkillRepositories) GetRepName(ctx context.Context) (string, error) {
	return "Reps", nil
}

func (g *GkillRepositories) GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error) {
	kyouHistories := map[string]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Kyou, len(g.Reps))
	errch := make(chan error, len(g.Reps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.Reps {
		wg.Add(1)
		
		go func(rep Repository) {
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

func (g *GkillRepositories) FindTags(ctx context.Context, query *find.FindQuery) ([]*Tag, error) {
	matchTags := map[string]*Tag{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Tag, len(g.TagReps))
	errch := make(chan error, len(g.TagReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.TagReps {
		wg.Add(1)
		
		go func(rep TagRepository) {
			defer wg.Done()
			// jsonからパースする
			queryLatest := query

			// idsを指定されていなければ、最新であるもののIDのみを対象とする
			if query.IDs != nil {
				repName, err := rep.GetRepName(ctx)
				if err != nil {
					err = fmt.Errorf("error at get rep name: %w", err)
					errch <- err
					return
				}

				latestDataRepositoryAddresses, err := g.LatestDataRepositoryAddressDAO.GetLatestDataRepositoryAddressesByRepName(ctx, repName)
				if err != nil {
					err = fmt.Errorf("error at get latest data repository addresses by rep name %s: %w", repName, err)
					errch <- err
					return
				}

				idsStrBuf := bytes.NewBufferString("")
				ids := []string{}
				for _, latestDataRepositoryAddress := range latestDataRepositoryAddresses {
					ids = append(ids, latestDataRepositoryAddress.TargetID)
				}

				err = json.NewEncoder(idsStrBuf).Encode(ids)
				if err != nil {
					err = fmt.Errorf("error at latest ids in rep marshal json %#v: %w", ids, err)
					errch <- err
					return
				}
				trueValue := true
				query.IDs = &ids
				query.UseIDs = &trueValue

				queryLatest = query
			}

			matchTagsInRep, err := rep.FindTags(ctx, queryLatest)
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
				if existTag, exist := matchTags[tag.ID]; exist {
					if tag.UpdateTime.Before(existTag.UpdateTime) {
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

	matchTagsList := []*Tag{}
	for _, tag := range matchTags {
		if tag == nil {
			continue
		}
		matchTagsList = append(matchTagsList, tag)
	}

	sort.Slice(matchTagsList, func(i, j int) bool {
		return matchTagsList[i].RelatedTime.After(matchTagsList[j].RelatedTime)
	})
	return matchTagsList, nil
}

func (g *GkillRepositories) GetTag(ctx context.Context, id string) (*Tag, error) {
	matchTag := &Tag{}
	matchTag = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Tag, len(g.TagReps))
	errch := make(chan error, len(g.TagReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.TagReps {
		wg.Add(1)
		
		go func(rep TagRepository) {
			defer wg.Done()
			matchTagInRep, err := rep.GetTag(ctx, id)
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

func (g *GkillRepositories) GetTagsByTagName(ctx context.Context, tagname string) ([]*Tag, error) {
	matchTags := map[string]*Tag{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Tag, len(g.TagReps))
	errch := make(chan error, len(g.TagReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.TagReps {
		wg.Add(1)
		
		go func(rep TagRepository) {
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
					if tag.UpdateTime.Before(existTag.UpdateTime) {
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
		tagHistoriesList = append(tagHistoriesList, tag)
	}

	sort.Slice(tagHistoriesList, func(i, j int) bool {
		return tagHistoriesList[i].UpdateTime.After(tagHistoriesList[j].UpdateTime)
	})

	return tagHistoriesList, nil

}

func (g *GkillRepositories) GetTagsByTargetID(ctx context.Context, target_id string) ([]*Tag, error) {
	matchTags := map[string]*Tag{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Tag, len(g.TagReps))
	errch := make(chan error, len(g.TagReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.TagReps {
		wg.Add(1)
		
		go func(rep TagRepository) {
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
					if tag.UpdateTime.Before(existTag.UpdateTime) {
						matchTags[tag.ID] = tag
					}
				} else {
					matchTags[tag.ID+tag.UpdateTime.Format(sqlite3impl.TimeLayout)] = tag
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
		tagHistoriesList = append(tagHistoriesList, tag)
	}

	sort.Slice(tagHistoriesList, func(i, j int) bool {
		return tagHistoriesList[i].UpdateTime.After(tagHistoriesList[j].UpdateTime)
	})

	return tagHistoriesList, nil
}

func (g *GkillRepositories) GetTagHistories(ctx context.Context, id string) ([]*Tag, error) {
	tagHistories := map[string]*Tag{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Tag, len(g.TagReps))
	errch := make(chan error, len(g.TagReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.TagReps {
		wg.Add(1)
		
		go func(rep TagRepository) {
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
					if tag.UpdateTime.Before(existTag.UpdateTime) {
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

func (g *GkillRepositories) AddTagInfo(ctx context.Context, tag *Tag) error {
	err := fmt.Errorf("not implements GkillRepositories.AddTagInfo")
	return err
}

func (g *GkillRepositories) GetAllTagNames(ctx context.Context) ([]string, error) {
	tagNames := map[string]struct{}{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []string, len(g.TagReps))
	errch := make(chan error, len(g.TagReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.TagReps {
		wg.Add(1)
		
		go func(rep TagRepository) {
			defer wg.Done()
			matchTagNamesInRep, err := rep.GetAllTagNames(ctx)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTagNamesInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get all tagnames: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// タグ名集約
loop:
	for {
		select {
		case tagNamesInRep := <-ch:
			if tagNamesInRep == nil {
				continue loop
			}
			for _, tagName := range tagNamesInRep {
				tagNames[tagName] = struct{}{}
			}
		default:
			break loop
		}
	}

	tagNamesList := []string{}
	for tagName := range tagNames {
		tagNamesList = append(tagNamesList, tagName)
	}

	sort.Slice(tagNamesList, func(i, j int) bool {
		return tagNamesList[i] < tagNamesList[j]
	})

	return tagNamesList, nil
}

func (g *GkillRepositories) GetAllRepNames(ctx context.Context) ([]string, error) {
	repNames := map[string]struct{}{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan string, len(g.Reps))
	errch := make(chan error, len(g.Reps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.Reps {
		wg.Add(1)
		
		go func(rep Repository) {
			defer wg.Done()
			repName, err := rep.GetRepName(ctx)
			if err != nil {
				errch <- err
				return
			}
			ch <- repName
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get all repnames: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// タグ名集約
loop:
	for {
		select {
		case repName := <-ch:
			repNames[repName] = struct{}{}
		default:
			break loop
		}
	}

	repNamesList := []string{}
	for repName := range repNames {
		repNamesList = append(repNamesList, repName)
	}

	sort.Slice(repNamesList, func(i, j int) bool {
		return repNamesList[i] < repNamesList[j]
	})

	return repNamesList, nil
}

func (g *GkillRepositories) FindTexts(ctx context.Context, query *find.FindQuery) ([]*Text, error) {
	matchTexts := map[string]*Text{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Text, len(g.TextReps))
	errch := make(chan error, len(g.TextReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.TextReps {
		wg.Add(1)
		
		go func(rep TextRepository) {
			defer wg.Done()
			// jsonからパースする
			queryLatest := query
			ids := []string{}
			if query.IDs != nil {
				ids = *query.IDs
			}
			// idsを指定されていなければ、最新であるもののIDのみを対象とする
			if len(ids) == 0 {
				repName, err := rep.GetRepName(ctx)
				if err != nil {
					err = fmt.Errorf("error at get rep name: %w", err)
					errch <- err
					return
				}

				latestDataRepositoryAddresses, err := g.LatestDataRepositoryAddressDAO.GetLatestDataRepositoryAddressesByRepName(ctx, repName)
				if err != nil {
					err = fmt.Errorf("error at get latest data repository addresses by rep name %s: %w", repName, err)
					errch <- err
					return
				}

				idsStrBuf := bytes.NewBufferString("")
				for _, latestDataRepositoryAddress := range latestDataRepositoryAddresses {
					ids = append(ids, latestDataRepositoryAddress.TargetID)
				}

				err = json.NewEncoder(idsStrBuf).Encode(ids)
				if err != nil {
					err = fmt.Errorf("error at latest ids in rep marshal json %#v: %w", ids, err)
					errch <- err
					return
				}
				trueValue := true
				query.IDs = &ids
				query.UseIDs = &trueValue

				queryLatest = query
			}

			matchTextsInRep, err := rep.FindTexts(ctx, queryLatest)
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
					if text.UpdateTime.Before(existText.UpdateTime) {
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
		matchTextsList = append(matchTextsList, text)
	}

	sort.Slice(matchTextsList, func(i, j int) bool {
		return matchTextsList[i].RelatedTime.After(matchTextsList[j].RelatedTime)
	})
	return matchTextsList, nil
}

func (g *GkillRepositories) GetText(ctx context.Context, id string) (*Text, error) {
	matchText := &Text{}
	matchText = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Text, len(g.TextReps))
	errch := make(chan error, len(g.TextReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.TextReps {
		wg.Add(1)
		
		go func(rep TextRepository) {
			defer wg.Done()
			matchTextInRep, err := rep.GetText(ctx, id)
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

func (g *GkillRepositories) GetTextsByTargetID(ctx context.Context, target_id string) ([]*Text, error) {
	matchTexts := map[string]*Text{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Text, len(g.TextReps))
	errch := make(chan error, len(g.TextReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.TextReps {
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
					if text.UpdateTime.Before(existText.UpdateTime) {
						matchTexts[text.ID] = text
					}
				} else {
					matchTexts[text.ID+text.UpdateTime.Format(sqlite3impl.TimeLayout)] = text
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

func (g *GkillRepositories) GetTextHistories(ctx context.Context, id string) ([]*Text, error) {
	textHistories := map[string]*Text{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Text, len(g.TextReps))
	errch := make(chan error, len(g.TextReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.TextReps {
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
					if text.UpdateTime.Before(existText.UpdateTime) {
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

func (g *GkillRepositories) AddTextInfo(ctx context.Context, text *Text) error {
	err := fmt.Errorf("not implements GkillRepositories.AddTextInfo")
	return err
}
