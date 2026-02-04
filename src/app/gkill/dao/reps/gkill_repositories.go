package reps

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/account_state"

	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/threads"
)

var (
	// 全体で1つだけ起動されるように考慮。
	updateCacheThreadPool = make(chan interface{}, 1)
)

func init() {
	updateCacheThreadPool <- struct{}{}
}

type GkillRepositories struct {
	userID string

	Reps Repositories

	TagReps            TagRepositories
	TagRepsWatchTarget TagRepositories // 監視解除用

	TextReps            TextRepositories
	TextRepsWatchTarget TextRepositories // 監視解除用

	NotificationReps NotificationRepositories

	KmemoReps KmemoRepositories

	KCReps KCRepositories

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

	WriteKCRep KCRepository

	WriteURLogRep URLogRepository

	WriteNlogRep NlogRepository

	WriteTimeIsRep TimeIsRepository

	WriteMiRep MiRepository

	WriteLantanaRep LantanaRepository

	WriteIDFKyouRep IDFKyouRepository

	WriteReKyouRep ReKyouRepository

	WriteGPSLogRep GPSLogRepository

	LatestDataRepositoryAddressDAO account_state.LatestDataRepositoryAddressDAO
	TempReps                       *TempReps

	LatestDataRepositoryAddresses                       map[string]*account_state.LatestDataRepositoryAddress
	LastUpdatedLatestDataRepositoryAddressCacheFindTime time.Time

	IsUpdateCacheNextTick bool
	updateCacheTicker     *time.Ticker
	isClosed              bool

	cancelPreFunc    context.CancelFunc // 一回前で実行されたコンテキスト。キャンセル用
	updateCacheMutex sync.Mutex

	CacheMemoryDBMutex *sync.Mutex
	CacheMemoryDB      *sql.DB
	TempMemoryDBMutex  *sync.Mutex
	TempMemoryDB       *sql.DB
}

// repsとLatestDataRepositoryAddressDAOのみ初期化済みのGkillRepositoriesを返す
func NewGkillRepositories(userID string) (*GkillRepositories, error) {
	if userID == "" {
		err := fmt.Errorf("error at new gkill repositories. userID is blank")
		return nil, err
	}

	configDBRootDir := os.ExpandEnv(gkill_options.CacheDir)
	err := os.MkdirAll(configDBRootDir, fs.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at create directory: %w", err)
		return nil, err
	}

	// memory_dbの初期化
	var CacheMemoryDBMutex *sync.Mutex
	var CacheMemoryDB *sql.DB
	var TempMemoryDBMutex *sync.Mutex
	var TempMemoryDB *sql.DB
	if gkill_options.IsCacheInMemory {
		CacheMemoryDB, err = sql.Open("sqlite3", "file:gkill_memory_db_"+userID+"?mode=memory&cache=shared&_busy_timeout=6000&_txlock=immediate&_journal_mode=MEMORY&_synchronous=OFF")
		if err != nil {
			err = fmt.Errorf("error at open memory database: %w", err)
			gkill_log.Debug.Fatal(err)
		}
		CacheMemoryDB.SetMaxOpenConns(runtime.NumCPU()) // 読み取り並列を許可
		CacheMemoryDB.SetMaxIdleConns(1)                // 0にすると最後が閉じて消える
		CacheMemoryDB.SetConnMaxLifetime(0)             // 無限
		CacheMemoryDB.SetConnMaxIdleTime(0)             // 無限

		TempMemoryDB, err = sql.Open("sqlite3", "file:gkill_temp_db_"+userID+"?mode=memory&cache=shared&_busy_timeout=6000&_txlock=immediate&_journal_mode=MEMORY&_synchronous=OFF")
		if err != nil {
			err = fmt.Errorf("error at open memory database: %w", err)
			gkill_log.Debug.Fatal(err)
		}
		TempMemoryDB.SetMaxOpenConns(runtime.NumCPU()) // 読み取り並列を許可
		TempMemoryDB.SetMaxIdleConns(1)                // 0にすると最後が閉じて消える
		TempMemoryDB.SetConnMaxLifetime(0)             // 無限
		TempMemoryDB.SetConnMaxIdleTime(0)             // 無限

		CacheMemoryDBMutex = &sync.Mutex{}
		TempMemoryDBMutex = &sync.Mutex{}
	} else {
		TempMemoryDB, err = sql.Open("sqlite3", os.ExpandEnv(filepath.Join(gkill_options.CacheDir, userID+"_temp_"+".db?_timeout=6000&_synchronous=2&_journal=WAL")))
		if err != nil {
			err = fmt.Errorf("error at open database: %w", err)
			return nil, err
		}
		CacheMemoryDB, err = sql.Open("sqlite3", os.ExpandEnv(filepath.Join(gkill_options.CacheDir, userID+"_cache_"+".db?_timeout=6000&_synchronous=2&_journal=WAL")))
		if err != nil {
			err = fmt.Errorf("error at open database: %w", err)
			return nil, err
		}

		CacheMemoryDBMutex = &sync.Mutex{}
		TempMemoryDBMutex = &sync.Mutex{}
	}

	// メモリ上でやる
	latestDataRepositoryAddressDAO, err := account_state.NewLatestDataRepositoryAddressSQLite3Impl(userID, CacheMemoryDB, CacheMemoryDBMutex)
	if err != nil {
		err = fmt.Errorf("error at get latest data repository address dao. user id = %s: %w", userID, err)
		return nil, err
	}

	TempReps, err := NewTempReps(TempMemoryDB, TempMemoryDBMutex)
	if err != nil {
		return nil, err
	}

	// UpdateCacheNextTick用のTicker。キャッシュ更新（ファイル監視起動用）
	ticker := time.NewTicker(gkill_options.CacheUpdateDuration)

	repositories := &GkillRepositories{
		Reps:                           Repositories{},
		userID:                         userID,
		LatestDataRepositoryAddressDAO: latestDataRepositoryAddressDAO,

		cancelPreFunc: context.CancelFunc(func() {}),

		TempReps: TempReps,

		updateCacheMutex: sync.Mutex{},

		TempMemoryDB:       TempMemoryDB,
		CacheMemoryDB:      CacheMemoryDB,
		TempMemoryDBMutex:  TempMemoryDBMutex,
		CacheMemoryDBMutex: CacheMemoryDBMutex,

		updateCacheTicker:                                   ticker,
		LatestDataRepositoryAddresses:                       nil,
		LastUpdatedLatestDataRepositoryAddressCacheFindTime: time.Unix(0, 0),
	}

	go func() {
		for !repositories.isClosed {
			<-ticker.C
			if repositories.IsUpdateCacheNextTick {
				err := repositories.UpdateCache(context.Background())
				if err != nil {
					gkill_log.Error.Println(err.Error())
					return
				}
				repositories.IsUpdateCacheNextTick = false
			}
		}
	}()
	return repositories, nil
}

func (g *GkillRepositories) GetUserID(ctx context.Context) (string, error) {
	return g.userID, nil
}

func (g *GkillRepositories) Close(ctx context.Context) error {
	g.isClosed = true
	g.updateCacheTicker.Stop()
	if err := g.TagReps.Close(ctx); err != nil {
		gkill_log.Error.Println(err.Error())
		return err
	}
	if err := g.TextReps.Close(ctx); err != nil {
		gkill_log.Error.Println(err.Error())
		return err
	}
	if err := g.KmemoReps.Close(ctx); err != nil {
		gkill_log.Error.Println(err.Error())
		return err
	}
	if err := g.KCReps.Close(ctx); err != nil {
		gkill_log.Error.Println(err.Error())
		return err
	}
	if err := g.NlogReps.Close(ctx); err != nil {
		gkill_log.Error.Println(err.Error())
		return err
	}
	if err := g.TimeIsReps.Close(ctx); err != nil {
		gkill_log.Error.Println(err.Error())
		return err
	}

	if err := g.MiReps.Close(ctx); err != nil {
		gkill_log.Error.Println(err.Error())
		return err
	}
	if err := g.IDFKyouReps.Close(ctx); err != nil {
		gkill_log.Error.Println(err.Error())
		return err
	}
	if err := g.ReKyouReps.Close(ctx); err != nil {
		gkill_log.Error.Println(err.Error())
		return err
	}
	if err := g.GitCommitLogReps.Close(ctx); err != nil {
		gkill_log.Error.Println(err.Error())
		return err
	}
	if err := g.LantanaReps.Close(ctx); err != nil {
		gkill_log.Error.Println(err.Error())
		return err
	}
	err := g.LatestDataRepositoryAddressDAO.Close(ctx)
	if err != nil {
		gkill_log.Error.Println(err.Error())
	}

	g.CacheMemoryDB.Close()
	g.TempMemoryDB.Close()
	/*
		for _, rep := range g.GPSLogReps {
			err := rep.Close(ctx)
			if err != nil {
			gkill_log.Error.Println(err.Error())
			}
		}
	*/
	g.userID = ""

	return nil
}

func (g *GkillRepositories) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]*Kyou, error) {
	matchKyous := map[string][]*Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan map[string][]*Kyou, len(g.Reps))
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
		_ = threads.Go(ctx, wg, func() {
			func(rep Repository) {
				// jsonからパースする
				queryLatestValue := *query
				queryLatest := &queryLatestValue

				// idsを指定されていなければ、最新であるもののIDのみを対象とする
				if query.IDs == nil || len(*query.IDs) == 0 {
					ids := append([]string{}, *query.IDs...)
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

					for _, latestDataRepositoryAddress := range latestDataRepositoryAddresses {
						ids = append(ids, latestDataRepositoryAddress.TargetID)
					}
					trueValue := true
					queryLatest.IDs = &ids
					queryLatest.UseIDs = &trueValue
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
		})
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
			for _, kyous := range matchKyousInRep {
				for _, kyou := range kyous {
					if _, exist := matchKyous[kyou.ID]; !exist {
						matchKyous[kyou.ID] = []*Kyou{}
					}
					matchKyous[kyou.ID] = append(matchKyous[kyou.ID], kyou)
				}
			}
		default:
			break loop
		}
	}

	return matchKyous, nil
}

func (g *GkillRepositories) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	var matchKyou *Kyou
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
			if matchKyouInRep.UpdateTime.After(matchKyou.UpdateTime) {
				matchKyou = matchKyouInRep
			}
		} else {
			matchKyou = matchKyouInRep
		}
	}
	return matchKyou, nil
}

func (g *GkillRepositories) UpdateCache(ctx context.Context) error {
	<-updateCacheThreadPool
	func() {
		defer func() { updateCacheThreadPool <- struct{}{} }()
		g.updateCacheMutex.Lock()
		defer g.updateCacheMutex.Unlock()

		var cancelFunc context.CancelFunc

		// 一個前でUpdateCacheしてるやつをキャンセルする
		g.cancelPreFunc()
		ctx, cancelFunc = context.WithCancel(ctx)
		g.cancelPreFunc = cancelFunc
	}()

	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	kyousCh := make(chan map[string][]*Kyou, len(g.Reps))
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

	allKyousMap := map[string][]*Kyou{}
	allTags := []*Tag{}
	allTexts := []*Text{}
	allNotifications := []*Notification{}

	// UpdateCache並列処理
	updateCacheTargets := []interface {
		UpdateCache(ctx context.Context) error
	}{}
	updateCacheTargets = append(updateCacheTargets, g.Reps)
	updateCacheTargets = append(updateCacheTargets, g.TagReps)
	updateCacheTargets = append(updateCacheTargets, g.TextReps)
	updateCacheTargets = append(updateCacheTargets, g.NotificationReps)
	for _, rep := range updateCacheTargets {
		func(rep interface {
			UpdateCache(ctx context.Context) error
		}) {
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
		_ = threads.Go(ctx, wg, func() {
			func(rep Repository) {
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
		})
	}

	// tagを集める
	for _, rep := range g.TagReps {
		_ = threads.Go(ctx, wg, func() {
			func(rep TagRepository) {
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
		})
	}

	// textを集める
	for _, rep := range g.TextReps {
		_ = threads.Go(ctx, wg, func() {
			func(rep TextRepository) {
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
		})
	}

	// notificationを集める
	for _, rep := range g.NotificationReps {
		_ = threads.Go(ctx, wg, func() {
			func(rep NotificationRepository) {
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
		})
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
		case kyouMaps := <-kyousCh:
			for id, kyouMap := range kyouMaps {
				if _, exist := allKyousMap[id]; !exist {
					allKyousMap[id] = []*Kyou{}
				}
				allKyousMap[id] = append(allKyousMap[id], kyouMap...)
			}
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
	for id, kyousMap := range allKyousMap {
		for _, kyou := range kyousMap {
			if existKyou, exist := latestKyousMap[id]; exist {
				if kyou.UpdateTime.After(existKyou.UpdateTime) {
					latestKyousMap[kyou.ID] = kyou
				}
			} else {
				latestKyousMap[kyou.ID] = kyou
			}
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
	latestDataRepositoryAddresses := make([]*account_state.LatestDataRepositoryAddress, 0, len(latestKyousMap)+len(latestTagsMap)+len(latestTextsMap)+len(latestNotificationsMap))
	for _, kyou := range latestKyousMap {
		latestDataRepositoryAddress := &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              kyou.IsDeleted,
			TargetID:                               kyou.ID,
			LatestDataRepositoryName:               kyou.RepName,
			DataUpdateTime:                         kyou.UpdateTime,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}
		latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, latestDataRepositoryAddress)
	}
	for _, tag := range latestTagsMap {
		latestDataRepositoryAddress := &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              tag.IsDeleted,
			TargetID:                               tag.ID,
			TargetIDInData:                         &tag.TargetID,
			LatestDataRepositoryName:               tag.RepName,
			DataUpdateTime:                         tag.UpdateTime,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}
		latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, latestDataRepositoryAddress)
	}
	for _, text := range latestTextsMap {
		latestDataRepositoryAddress := &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              text.IsDeleted,
			TargetID:                               text.ID,
			TargetIDInData:                         &text.TargetID,
			LatestDataRepositoryName:               text.RepName,
			DataUpdateTime:                         text.UpdateTime,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}
		latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, latestDataRepositoryAddress)
	}
	for _, notification := range latestNotificationsMap {
		latestDataRepositoryAddress := &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              notification.IsDeleted,
			TargetID:                               notification.ID,
			TargetIDInData:                         &notification.TargetID,
			LatestDataRepositoryName:               notification.RepName,
			DataUpdateTime:                         notification.UpdateTime,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}
		latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, latestDataRepositoryAddress)
	}

	updatedLatestDataRepositoryAddresses, err := g.LatestDataRepositoryAddressDAO.ExtructUpdatedLatestDataRepositoryAddressDatas(ctx, latestDataRepositoryAddresses)
	if err != nil {
		err = fmt.Errorf("error at update latest data repository address cache data: %w", err)
		return err
	}

	if g.LatestDataRepositoryAddresses == nil {
		g.LatestDataRepositoryAddresses = map[string]*account_state.LatestDataRepositoryAddress{}
	}
	for _, updatedLatestDataRepositoryAddress := range updatedLatestDataRepositoryAddresses {
		g.LatestDataRepositoryAddresses[updatedLatestDataRepositoryAddress.TargetID] = updatedLatestDataRepositoryAddress
	}

	_, err = g.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddresses(ctx, updatedLatestDataRepositoryAddresses)
	if err != nil {
		err = fmt.Errorf("error at update latest data repository address cache data: %w", err)
		return err
	}
	g.LastUpdatedLatestDataRepositoryAddressCacheFindTime = time.Now()
	return nil
}

func (g *GkillRepositories) UpdateCacheNextTick() {
	g.IsUpdateCacheNextTick = true
}

func (g *GkillRepositories) GetPath(ctx context.Context, id string) (string, error) {
	// 並列処理
	matchPaths := []string{}
	trueValue := true
	ids := []string{id}
	for _, rep := range g.Reps {
		query := &find.FindQuery{
			IDs:    &ids,
			UseIDs: &trueValue,
		}
		kyous, err := rep.FindKyous(ctx, query)
		if len(kyous) == 0 || err != nil {
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
		_ = threads.Go(ctx, wg, func() {
			func(rep Repository) {
				matchKyousInRep, err := rep.GetKyouHistories(ctx, id)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchKyousInRep
			}(rep)
		})
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
		_ = threads.Go(ctx, wg, func() {
			func(rep TagRepository) {
				queryLatestValue := *query
				queryLatest := &queryLatestValue

				// idsを指定されていなければ、最新であるもののIDのみを対象とする
				if query.IDs == nil || len(*query.IDs) == 0 {
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

					ids := []string{}
					for _, latestDataRepositoryAddress := range latestDataRepositoryAddresses {
						ids = append(ids, latestDataRepositoryAddress.TargetID)
					}

					trueValue := true
					queryLatest.IDs = &ids
					queryLatest.UseIDs = &trueValue
				}

				matchTagsInRep, err := rep.FindTags(ctx, queryLatest)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchTagsInRep
			}(rep)
		})
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

	sort.Slice(matchTagsList, func(i, j int) bool {
		return matchTagsList[i].RelatedTime.After(matchTagsList[j].RelatedTime)
	})
	return matchTagsList, nil
}

func (g *GkillRepositories) GetTag(ctx context.Context, id string, updateTime *time.Time) (*Tag, error) {
	var matchTag *Tag
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Tag, len(g.TagReps))
	errch := make(chan error, len(g.TagReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.TagReps {
		_ = threads.Go(ctx, wg, func() {
			func(rep TagRepository) {
				matchTagInRep, err := rep.GetTag(ctx, id, updateTime)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchTagInRep
			}(rep)
		})
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
				if matchTagInRep.UpdateTime.After(matchTag.UpdateTime) {
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
		_ = threads.Go(ctx, wg, func() {
			func(rep TagRepository) {
				matchTagsInRep, err := rep.GetTagsByTagName(ctx, tagname)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchTagsInRep
			}(rep)
		})
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
		_ = threads.Go(ctx, wg, func() {
			func(rep TagRepository) {
				matchTagsInRep, err := rep.GetTagHistories(ctx, id)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchTagsInRep
			}(rep)
		})
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
	var err error
	repImpls, err := g.Reps.UnWrap()
	if err != nil {
		err = fmt.Errorf("error at unwrap reps: %w", err)
		return nil, err
	}

	repNames := map[string]struct{}{}
	existErr := false
	wg := &sync.WaitGroup{}
	ch := make(chan string, len(repImpls))
	errch := make(chan error, len(repImpls))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range repImpls {
		_ = threads.Go(ctx, wg, func() {
			func(rep Repository) {
				repName, err := rep.GetRepName(ctx)
				if err != nil {
					errch <- err
					return
				}
				ch <- repName
			}(rep)
		})
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
		_ = threads.Go(ctx, wg, func() {
			func(rep TextRepository) {
				// jsonからパースする
				queryLatest := query
				ids := []string{}
				if query.IDs != nil {
					ids = append([]string{}, *query.IDs...)
				}
				// idsを指定されていなければ、最新であるもののIDのみを対象とする
				if query.IDs == nil || len(ids) == 0 {
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

					for _, latestDataRepositoryAddress := range latestDataRepositoryAddresses {
						ids = append(ids, latestDataRepositoryAddress.TargetID)
					}

					trueValue := true
					queryLatest.IDs = &ids
					queryLatest.UseIDs = &trueValue
				}

				matchTextsInRep, err := rep.FindTexts(ctx, queryLatest)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchTextsInRep
			}(rep)
		})
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
				key := text.ID
				if query.OnlyLatestData == nil || !*query.OnlyLatestData {
					key += fmt.Sprintf("%d", text.UpdateTime.Unix())
				}
				if existText, exist := matchTexts[key]; exist {
					if text.UpdateTime.After(existText.UpdateTime) {
						matchTexts[key] = text
					}
				} else {
					matchTexts[key] = text
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
	var matchText *Text
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Text, len(g.TextReps))
	errch := make(chan error, len(g.TextReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.TextReps {
		_ = threads.Go(ctx, wg, func() {
			func(rep TextRepository) {
				matchTextInRep, err := rep.GetText(ctx, id, updateTime)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchTextInRep
			}(rep)
		})
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
				if matchTextInRep.UpdateTime.After(matchText.UpdateTime) {
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
	var matchNotification *Notification
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Notification, len(g.NotificationReps))
	errch := make(chan error, len(g.NotificationReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.NotificationReps {
		_ = threads.Go(ctx, wg, func() {
			func(rep NotificationRepository) {
				matchNotificationInRep, err := rep.GetNotification(ctx, id, updateTime)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchNotificationInRep
			}(rep)
		})
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
				if matchNotificationInRep.UpdateTime.After(matchNotification.UpdateTime) {
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
		_ = threads.Go(ctx, wg, func() {
			func(rep TextRepository) {
				matchTextsInRep, err := rep.GetTextsByTargetID(ctx, target_id)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchTextsInRep
			}(rep)
		})
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
		_ = threads.Go(ctx, wg, func() {
			func(rep NotificationRepository) {
				matchNotificationsInRep, err := rep.GetNotificationsByTargetID(ctx, target_id)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchNotificationsInRep
			}(rep)
		})
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
		_ = threads.Go(ctx, wg, func() {
			func(rep TextRepository) {
				matchTextsInRep, err := rep.GetTextHistories(ctx, id)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchTextsInRep
			}(rep)
		})
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
		_ = threads.Go(ctx, wg, func() {
			func(rep NotificationRepository) {
				matchNotificationsInRep, err := rep.GetNotificationHistories(ctx, id)
				if err != nil {
					errch <- err
					return
				}
				ch <- matchNotificationsInRep
			}(rep)
		})
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
		_ = threads.Go(ctx, wg, func() {
			func(rep Repository) {
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
		})
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
