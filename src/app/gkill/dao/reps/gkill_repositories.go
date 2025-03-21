package reps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/account_state"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

type GkillRepositories struct {
	userID string

	Reps Repositories

	TagReps TagRepositories

	TextReps TextRepositories

	NotificationReps NotificationRepositories

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

	WriteNotificationRep NotificationRepository

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

	cancelPreFunc context.CancelFunc // 一回前で実行されたコンテキスト。キャンセル用

	m sync.Mutex
}

// repsとLatestDataRepositoryAddressDAOのみ初期化済みのGkillRepositoriesを返す
func NewGkillRepositories(userID string) (*GkillRepositories, error) {
	if userID == "" {
		err := fmt.Errorf("error at new gkill repositories. userID is blank.")
		return nil, err
	}

	configDBRootDir := os.ExpandEnv(gkill_options.CacheDir)
	err := os.MkdirAll(configDBRootDir, fs.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at create directory %s: %w", err)
		return nil, err
	}

	// メモリ上でやる
	latestDataRepositoryAddressDAO, err := account_state.NewLatestDataRepositoryAddressSQLite3Impl(userID)
	if err != nil {
		err = fmt.Errorf("error at get latest data repository address dao. user id = %s: %w", userID, err)
		return nil, err
	}

	return &GkillRepositories{
		Reps:                           Repositories{},
		userID:                         userID,
		LatestDataRepositoryAddressDAO: latestDataRepositoryAddressDAO,

		cancelPreFunc: context.CancelFunc(func() {}),

		m: sync.Mutex{},
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
	for _, rep := range g.NotificationReps {
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

	matchReps, err := g.selectMatchRepsFromQuery(ctx, query)
	if err != nil {
		err = fmt.Errorf("error at select match reps from query: %w", err)
		return nil, err
	}

	// 並列処理
	for _, rep := range matchReps {
		wg.Add(1)

		go func(rep Repository) {
			defer wg.Done()
			// jsonからパースする
			queryLatestValue := *query
			queryLatest := &queryLatestValue

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
				repName, _ := rep.GetRepName(ctx)
				err = fmt.Errorf("error at %s: %w", repName, err)
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
					if kyou.UpdateTime.After(existKyou.UpdateTime) {
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
		if kyou.IsDeleted {
			continue
		}
		matchKyousList = append(matchKyousList, kyou)
	}

	sort.Slice(matchKyousList, func(i, j int) bool {
		return matchKyousList[i].RelatedTime.After(matchKyousList[j].RelatedTime)
	})
	return matchKyousList, nil
}

func (g *GkillRepositories) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	matchKyou := &Kyou{}
	matchKyou = nil
	matchKyousInRep := []*Kyou{}
	var err error
	ch := make(chan *Kyou, len(g.Reps))
	errch := make(chan error, len(g.Reps))
	defer close(ch)
	defer close(errch)

	latestDataRepositoryAddress, err := g.LatestDataRepositoryAddressDAO.GetLatestDataRepositoryAddress(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get latest data repository addresses by id %s: %w", id, err)
		return nil, err
	}

	for _, rep := range g.Reps {
		repName, err := rep.GetRepName(ctx)
		if err != nil {
			return nil, err
		}

		if repName != latestDataRepositoryAddress.LatestDataRepositoryName {
			continue
		}

		matchKyouInRep, err := rep.GetKyou(ctx, id, updateTime)
		if err != nil {
			continue
		}
		matchKyousInRep = append(matchKyousInRep, matchKyouInRep)
	}

	// Kyou集約。UpdateTimeが最新のものを収める
	for _, matchKyouInRep := range matchKyousInRep {
		if matchKyou != nil {
			if matchKyouInRep.UpdateTime.Before(matchKyou.UpdateTime) {
				matchKyou = matchKyouInRep
			}
		} else {
			matchKyou = matchKyouInRep
		}
	}
	return matchKyou, nil
}

func (g *GkillRepositories) UpdateCache(ctx context.Context) error {
	func() {
		g.m.Lock()
		defer g.m.Unlock()

		var cancelFunc context.CancelFunc

		// 一個前でUpdateCacheしてるやつをキャンセルする
		g.cancelPreFunc()
		ctx, cancelFunc = context.WithCancel(ctx)
		g.cancelPreFunc = cancelFunc
	}()

	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	kyousCh := make(chan []*Kyou, len(g.Reps))
	tagsCh := make(chan []*Tag, len(g.TagReps))
	textsCh := make(chan []*Text, len(g.TextReps))
	notificationsCh := make(chan []*Notification, len(g.NotificationReps))
	rekyousCh := make(chan []*Kyou, len(g.ReKyouReps.ReKyouRepositories))
	errch := make(chan error, len(g.Reps)+len(g.TagReps)+len(g.TextReps)+len(g.NotificationReps))
	rekyouErrch := make(chan error, len(g.ReKyouReps.ReKyouRepositories))
	defer close(kyousCh)
	defer close(tagsCh)
	defer close(textsCh)
	defer close(notificationsCh)
	defer close(errch)
	defer close(rekyousCh)
	defer close(rekyouErrch)

	allKyous := []*Kyou{}
	allTags := []*Tag{}
	allTexts := []*Text{}
	allNotifications := []*Notification{}

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
			select {
			case <-ctx.Done():
				e := ctx.Err()
				if e != nil {
					err = fmt.Errorf("error at update cache: %w", e)
					errch <- err
					return
				}
			default:
				repName, err := rep.GetRepName(ctx)
				if err != nil {
					err = fmt.Errorf("error at get rep name: %w", err)
					errch <- err
					return
				}

				reps := []string{repName}
				kyous, err := rep.FindKyous(ctx, &find.FindQuery{Reps: &reps})
				if err != nil {
					repName, _ := rep.GetRepName(ctx)
					err = fmt.Errorf("error at %s: %w", repName, err)
					errch <- err
					return
				}
				kyousCh <- kyous
			}
		}(rep)
	}

	// tagを集める
	for _, rep := range g.TagReps {
		wg.Add(1)

		go func(rep TagRepository) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				e := ctx.Err()
				if e != nil {
					err = fmt.Errorf("error at update cache: %w", e)
					errch <- err
					return
				}
			default:
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
			}
		}(rep)
	}

	// textを集める
	for _, rep := range g.TextReps {
		wg.Add(1)

		go func(rep TextRepository) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				e := ctx.Err()
				if e != nil {
					err = fmt.Errorf("error at update cache: %w", e)
					errch <- err
					return
				}
			default:
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
			}
		}(rep)
	}

	// notificationを集める
	for _, rep := range g.NotificationReps {
		wg.Add(1)

		go func(rep NotificationRepository) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				e := ctx.Err()
				if e != nil {
					err = fmt.Errorf("error at update cache: %w", e)
					errch <- err
					return
				}
			default:
				repName, err := rep.GetRepName(ctx)
				if err != nil {
					err = fmt.Errorf("error at get rep name: %w", err)
					errch <- err
					return
				}

				reps := []string{repName}
				notifications, err := rep.FindNotifications(ctx, &find.FindQuery{Reps: &reps})
				if err != nil {
					errch <- err
					return
				}
				notificationsCh <- notifications
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

	// notification集約
notificationsloop:
	for {
		select {
		case notifications := <-notificationsCh:
			allNotifications = append(allNotifications, notifications...)
		default:
			break notificationsloop
		}
	}

	// 最新のKyou, tag, textのみにする
	latestKyousMap := map[string]*Kyou{}
	for _, kyou := range allKyous {
		if existKyou, exist := latestKyousMap[kyou.ID]; exist {
			if kyou.UpdateTime.After(existKyou.UpdateTime) {
				latestKyousMap[kyou.ID] = kyou
			}
		} else {
			latestKyousMap[kyou.ID] = kyou
		}
	}
	latestTagsMap := map[string]*Tag{}
	for _, tag := range allTags {
		if existTag, exist := latestTagsMap[tag.ID]; exist {
			if tag.UpdateTime.After(existTag.UpdateTime) {
				latestTagsMap[tag.ID] = tag
			}
		} else {
			latestTagsMap[tag.ID] = tag
		}
	}
	latestTextsMap := map[string]*Text{}
	for _, text := range allTexts {
		if existText, exist := latestTextsMap[text.ID]; exist {
			if text.UpdateTime.After(existText.UpdateTime) {
				latestTextsMap[text.ID] = text
			}
		} else {
			latestTextsMap[text.ID] = text
		}
	}
	latestNotificationsMap := map[string]*Notification{}
	for _, notification := range allNotifications {
		if existNotification, exist := latestNotificationsMap[notification.ID]; exist {
			if notification.UpdateTime.After(existNotification.UpdateTime) {
				latestNotificationsMap[notification.ID] = notification
			}
		} else {
			latestNotificationsMap[notification.ID] = notification
		}
	}

	// 最新のKyou, Tag, Text, Notificationの状態をLatestDataRepositoryAddressにいれる
	latestDataRepositoryAddresses := []*account_state.LatestDataRepositoryAddress{}
	for _, kyou := range latestKyousMap {
		latestDataRepositoryAddress := &account_state.LatestDataRepositoryAddress{
			IsDeleted:                kyou.IsDeleted,
			TargetID:                 kyou.ID,
			LatestDataRepositoryName: kyou.RepName,
			DataUpdateTime:           kyou.UpdateTime,
		}
		latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, latestDataRepositoryAddress)
	}
	for _, tag := range latestTagsMap {
		latestDataRepositoryAddress := &account_state.LatestDataRepositoryAddress{
			IsDeleted:                tag.IsDeleted,
			TargetID:                 tag.ID,
			LatestDataRepositoryName: tag.RepName,
			DataUpdateTime:           tag.UpdateTime,
		}
		latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, latestDataRepositoryAddress)
	}
	for _, text := range latestTextsMap {
		latestDataRepositoryAddress := &account_state.LatestDataRepositoryAddress{
			IsDeleted:                text.IsDeleted,
			TargetID:                 text.ID,
			LatestDataRepositoryName: text.RepName,
			DataUpdateTime:           text.UpdateTime,
		}
		latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, latestDataRepositoryAddress)
	}
	for _, notification := range latestNotificationsMap {
		latestDataRepositoryAddress := &account_state.LatestDataRepositoryAddress{
			IsDeleted:                notification.IsDeleted,
			TargetID:                 notification.ID,
			LatestDataRepositoryName: notification.RepName,
			DataUpdateTime:           notification.UpdateTime,
		}
		latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, latestDataRepositoryAddress)
	}

	_, err = g.LatestDataRepositoryAddressDAO.DeleteAllLatestDataRepositoryAddress(ctx)
	if err != nil {
		err = fmt.Errorf("error at delete all latest data repository address cache: %w", err)
		return err
	}

	_, err = g.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddresses(ctx, latestDataRepositoryAddresses)
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
				if kyou == nil {
					continue
				}
				if existKyou, exist := kyouHistories[kyou.ID+kyou.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if kyou.UpdateTime.After(existKyou.UpdateTime) {
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
			queryLatestValue := *query
			queryLatest := &queryLatestValue

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

	sort.Slice(matchTagsList, func(i, j int) bool {
		return matchTagsList[i].RelatedTime.After(matchTagsList[j].RelatedTime)
	})
	return matchTagsList, nil
}

func (g *GkillRepositories) GetTag(ctx context.Context, id string, updateTime *time.Time) (*Tag, error) {
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

	sort.Slice(tagHistoriesList, func(i, j int) bool {
		return tagHistoriesList[i].UpdateTime.After(tagHistoriesList[j].UpdateTime)
	})

	return tagHistoriesList, nil

}

func (g *GkillRepositories) GetTagsByTargetID(ctx context.Context, target_id string) ([]*Tag, error) {
	matchTags, err := g.TagReps.GetTagsByTargetID(ctx, target_id)
	if err != nil {
		err = fmt.Errorf("error at get tags by target id: %w", err)
		return nil, err
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
		if tag.IsDeleted {
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
	return g.TagReps.GetAllTagNames(ctx)
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

func (g *GkillRepositories) GetText(ctx context.Context, id string, updateTime *time.Time) (*Text, error) {
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

func (g *GkillRepositories) GetNotification(ctx context.Context, id string, updateTime *time.Time) (*Notification, error) {
	matchNotification := &Notification{}
	matchNotification = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Notification, len(g.NotificationReps))
	errch := make(chan error, len(g.NotificationReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.NotificationReps {
		wg.Add(1)

		go func(rep NotificationRepository) {
			defer wg.Done()
			matchNotificationInRep, err := rep.GetNotification(ctx, id, updateTime)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchNotificationInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get notification: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Notification集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchNotificationInRep := <-ch:
			if matchNotificationInRep == nil {
				continue loop
			}
			if matchNotification != nil {
				if matchNotificationInRep.UpdateTime.Before(matchNotification.UpdateTime) {
					matchNotification = matchNotificationInRep
				}
			} else {
				matchNotification = matchNotificationInRep
			}
		default:
			break loop
		}
	}

	return matchNotification, nil
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
					if text.UpdateTime.After(existText.UpdateTime) {
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
		if text.IsDeleted {
			continue
		}
		textHistoriesList = append(textHistoriesList, text)
	}

	sort.Slice(textHistoriesList, func(i, j int) bool {
		return textHistoriesList[i].UpdateTime.After(textHistoriesList[j].UpdateTime)
	})

	return textHistoriesList, nil
}

func (g *GkillRepositories) GetNotificationsByTargetID(ctx context.Context, target_id string) ([]*Notification, error) {
	matchNotifications := map[string]*Notification{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Notification, len(g.NotificationReps))
	errch := make(chan error, len(g.NotificationReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.NotificationReps {
		wg.Add(1)

		go func(rep NotificationRepository) {
			defer wg.Done()
			matchNotificationsInRep, err := rep.GetNotificationsByTargetID(ctx, target_id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchNotificationsInRep
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

	// Notification集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchNotificationsInRep := <-ch:
			if matchNotificationsInRep == nil {
				continue loop
			}
			for _, notification := range matchNotificationsInRep {
				if existNotification, exist := matchNotifications[notification.ID]; exist {
					if notification.UpdateTime.After(existNotification.UpdateTime) {
						matchNotifications[notification.ID] = notification
					}
				} else {
					matchNotifications[notification.ID+notification.UpdateTime.Format(sqlite3impl.TimeLayout)] = notification
				}
			}
		default:
			break loop
		}
	}

	notificationHistoriesList := []*Notification{}
	for _, notification := range matchNotifications {
		if notification == nil {
			continue
		}
		if notification.IsDeleted {
			continue
		}
		notificationHistoriesList = append(notificationHistoriesList, notification)
	}

	sort.Slice(notificationHistoriesList, func(i, j int) bool {
		return notificationHistoriesList[i].UpdateTime.After(notificationHistoriesList[j].UpdateTime)
	})

	return notificationHistoriesList, nil
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
		if text.IsDeleted {
			continue
		}
		textHistoriesList = append(textHistoriesList, text)
	}

	sort.Slice(textHistoriesList, func(i, j int) bool {
		return textHistoriesList[i].UpdateTime.After(textHistoriesList[j].UpdateTime)
	})

	return textHistoriesList, nil
}

func (g *GkillRepositories) GetNotificationHistories(ctx context.Context, id string) ([]*Notification, error) {
	notificationHistories := map[string]*Notification{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Notification, len(g.NotificationReps))
	errch := make(chan error, len(g.NotificationReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.NotificationReps {
		wg.Add(1)

		go func(rep NotificationRepository) {
			defer wg.Done()
			matchNotificationsInRep, err := rep.GetNotificationHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchNotificationsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get notification histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Notification集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchNotificationsInRep := <-ch:
			if matchNotificationsInRep == nil {
				continue loop
			}
			for _, notification := range matchNotificationsInRep {
				if existNotification, exist := notificationHistories[notification.ID+notification.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if notification.UpdateTime.After(existNotification.UpdateTime) {
						notificationHistories[notification.ID+notification.UpdateTime.Format(sqlite3impl.TimeLayout)] = notification
					}
				} else {
					notificationHistories[notification.ID+notification.UpdateTime.Format(sqlite3impl.TimeLayout)] = notification
				}
			}
		default:
			break loop
		}
	}

	notificationHistoriesList := []*Notification{}
	for _, notification := range notificationHistories {
		if notification == nil {
			continue
		}
		if notification.IsDeleted {
			continue
		}
		notificationHistoriesList = append(notificationHistoriesList, notification)
	}

	sort.Slice(notificationHistoriesList, func(i, j int) bool {
		return notificationHistoriesList[i].UpdateTime.After(notificationHistoriesList[j].UpdateTime)
	})

	return notificationHistoriesList, nil
}

func (g *GkillRepositories) AddTextInfo(ctx context.Context, text *Text) error {
	err := fmt.Errorf("not implements GkillRepositories.AddTextInfo")
	return err
}

func (g *GkillRepositories) selectMatchRepsFromQuery(ctx context.Context, query *find.FindQuery) (map[string]Repository, error) {
	matchReps := map[string]Repository{}

	var err error
	existErr := false
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(g.Reps))
	defer close(errch)

	// 並列処理
	m := &sync.Mutex{}
	targetReps := g.Reps
	if query.UsePlaing != nil && *query.UsePlaing {
		matchReps["timeis"] = g.TimeIsReps
		return matchReps, nil
	}
	for _, rep := range targetReps {
		wg.Add(1)
		go func(rep Repository) {
			defer wg.Done()

			repName, err := rep.GetRepName(ctx)
			if err != nil {
				errch <- err
				return
			}

			if query.Reps != nil {
				for _, targetRepName := range *query.Reps {
					if targetRepName == repName {
						m.Lock()
						if _, exist := matchReps[repName]; !exist {
							matchReps[repName] = rep
						}
						m.Unlock()
					}
				}
			} else if query.Reps == nil || len(*query.Reps) == 0 {
				m.Lock()
				if _, exist := matchReps[repName]; !exist {
					matchReps[repName] = rep
				}
				m.Unlock()
			}
			errch <- nil
		}(rep)
	}
	wg.Wait()
	// エラー集約
	for range len(g.Reps) {
		e := <-errch
		if e != nil {
			err = fmt.Errorf("error at update cache: %w: %w", e, err)
			existErr = true
		}
	}
	if existErr {
		return nil, err
	}
	return matchReps, nil
}
