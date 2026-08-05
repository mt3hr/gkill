package reps

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"

	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/threads"
)

var (
	// 全体で1つだけ起動されるように考慮。
	updateCacheThreadPool = make(chan any, 1)
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

	MiReKyouReps MiReKyouRepositories

	GitCommitLogReps GitCommitLogRepositories

	GPSLogReps GPSLogRepositories

	// PluginReps はプラグインリポジトリ一覧。Repsにも含まれる。
	// APIハンドラがGetContentHTML等を呼ぶ際に使用する。
	PluginReps []PluginRepository

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

	WriteMiReKyouRep MiReKyouRepository

	WriteGPSLogRep GPSLogRepository

	LatestDataRepositoryAddressDAO gkill_cache.LatestDataRepositoryAddressDAO
	TempReps                       *TempReps

	// latestDataRepositoryAddressesはユーザ+デバイス単位で共有されるGkillRepositoriesが持つ。
	// 検索(FindKyous)と追加/更新(usecase, handle_commit_tx)の両方から触られるので、
	// 素のmapのまま公開すると concurrent map read and map write でプロセスごと落ちる
	// (この種のfatal errorはrecoverできない)。
	// 直接触らず、syncStateのlatestDataAddressesMutexで保護したアクセサ経由で読み書きすること。
	// 非公開にしているのは、新しいアクセス箇所が増えたときにコンパイルエラーで気づけるようにするため。
	latestDataRepositoryAddresses                       map[string]gkill_cache.LatestDataRepositoryAddress
	lastUpdatedLatestDataRepositoryAddressCacheFindTime time.Time

	updateCacheTicker *time.Ticker

	cancelPreFunc context.CancelFunc // 一回前で実行されたコンテキスト。キャンセル用

	// 値コピーしてはいけない同期状態。cloneRepositoriesWithoutReKyouがGkillRepositoriesを
	// 値コピーするため、ポインタで保持して複製ではなく共有されるようにしている。
	syncStateRef *gkillRepositoriesSync

	CacheMemoryDBMutex *sync.RWMutex
	CacheMemoryDB      *sql.DB
	TempMemoryDBMutex  *sync.RWMutex
	TempMemoryDB       *sql.DB

	SkipUpdateCache *bool // fsnotifyによるUpdateCacheの再帰トリガーを防止するためのフラグ。GkillDAOManagerのskipUpdateCacheと同じポインタを共有する。
}

// gkillRepositoriesSyncはGkillRepositoriesの同期状態。
// sync.RWMutexとatomic.Uint64は値コピーしてはいけないため、この構造体にまとめて
// GkillRepositoriesからはポインタで参照する。
type gkillRepositoriesSync struct {
	updateCacheMutex sync.RWMutex

	// latestDataAddressesMutexはlatestDataRepositoryAddressesと
	// lastUpdatedLatestDataRepositoryAddressCacheFindTimeを保護する。
	latestDataAddressesMutex sync.RWMutex

	// isClosedとisUpdateCacheNextTickはCloseや各ハンドラと
	// キャッシュ更新tickerのgoroutineの間で共有されるためatomicで持つ。
	isClosed              atomic.Bool
	isUpdateCacheNextTick atomic.Bool

	// closeChはCloseを検知してtickerのgoroutineを即座に終わらせるためのもの。
	// 停止したtickerは二度と発火しないので、<-ticker.C だけで待っていると
	// Close後にgoroutineが永久にブロックして残り続ける。
	closeCh chan struct{}

	updateReqGen atomic.Uint64
	lastHandled  atomic.Uint64
}

func newGkillRepositoriesSync() *gkillRepositoriesSync {
	return &gkillRepositoriesSync{
		closeCh: make(chan struct{}),
	}
}

// gkillRepositoriesSyncInitMutexはsyncStateの遅延初期化を保護する。
var gkillRepositoriesSyncInitMutex sync.Mutex

// syncStateは同期状態を返す。
// GkillRepositories{}リテラルで生成された場合に備えて遅延初期化する。
func (g *GkillRepositories) syncState() *gkillRepositoriesSync {
	gkillRepositoriesSyncInitMutex.Lock()
	defer gkillRepositoriesSyncInitMutex.Unlock()
	if g.syncStateRef == nil {
		g.syncStateRef = newGkillRepositoriesSync()
	}
	return g.syncStateRef
}

// GetLatestDataRepositoryAddress はIDに対応する最新版アドレスを返す。
func (g *GkillRepositories) GetLatestDataRepositoryAddress(id string) (gkill_cache.LatestDataRepositoryAddress, bool) {
	sync := g.syncState()
	sync.latestDataAddressesMutex.RLock()
	defer sync.latestDataAddressesMutex.RUnlock()

	addr, exist := g.latestDataRepositoryAddresses[id]
	return addr, exist
}

// SetLatestDataRepositoryAddress はIDに対応する最新版アドレスを1件書き込む。
func (g *GkillRepositories) SetLatestDataRepositoryAddress(id string, addr gkill_cache.LatestDataRepositoryAddress) {
	sync := g.syncState()
	sync.latestDataAddressesMutex.Lock()
	defer sync.latestDataAddressesMutex.Unlock()

	if g.latestDataRepositoryAddresses == nil {
		g.latestDataRepositoryAddresses = map[string]gkill_cache.LatestDataRepositoryAddress{}
	}
	g.latestDataRepositoryAddresses[id] = addr
}

// SetLatestDataRepositoryAddresses は最新版アドレスをまとめて差し替える。
func (g *GkillRepositories) SetLatestDataRepositoryAddresses(addresses map[string]gkill_cache.LatestDataRepositoryAddress) {
	sync := g.syncState()
	sync.latestDataAddressesMutex.Lock()
	defer sync.latestDataAddressesMutex.Unlock()

	g.latestDataRepositoryAddresses = addresses
}

// RefreshLatestDataRepositoryAddresses は最新版アドレスのキャッシュを最新化する。
// 未取得なら全件を読み込み、取得済みなら前回時刻以降の差分だけをマージする。
// 「件数を見て分岐し、読み込んで、書き戻す」までを1つのロックで囲む必要があるため、
// 呼び出し側にmapを渡さずこのメソッドの中で完結させている。
func (g *GkillRepositories) RefreshLatestDataRepositoryAddresses(ctx context.Context) error {
	sync := g.syncState()
	sync.latestDataAddressesMutex.Lock()
	defer sync.latestDataAddressesMutex.Unlock()

	if len(g.latestDataRepositoryAddresses) == 0 {
		latestDatas, err := g.LatestDataRepositoryAddressDAO.GetAllLatestDataRepositoryAddresses(ctx)
		if err != nil {
			return fmt.Errorf("error at get all latest data repository addresses: %w", err)
		}
		g.latestDataRepositoryAddresses = latestDatas
	} else {
		updatedLatestDatas, err := g.LatestDataRepositoryAddressDAO.GetLatestDataRepositoryAddressByUpdateTimeAfter(ctx, g.lastUpdatedLatestDataRepositoryAddressCacheFindTime, math.MaxInt)
		if err != nil {
			return fmt.Errorf("error at get updated latest data repository addresses: %w", err)
		}
		for _, latestData := range updatedLatestDatas {
			g.latestDataRepositoryAddresses[latestData.TargetID] = latestData
		}
	}
	g.lastUpdatedLatestDataRepositoryAddressCacheFindTime = time.Now()
	return nil
}

// EnsureLatestDataRepositoryAddresses は最新版アドレスのキャッシュが空なら全件読み込む。
//
// 検索経路では find_filter が頭で RefreshLatestDataRepositoryAddresses を呼ぶので
// ここは何もせずに返る。ReKyou/MiReKyouのターゲット解決は
// 検索経路以外(TX中の一時rep・単体テストなど)からも呼ばれるため、
// キャッシュが未取得のまま読まれて「ターゲットが全部消えている」ことにならないようにする。
func (g *GkillRepositories) EnsureLatestDataRepositoryAddresses(ctx context.Context) error {
	sync := g.syncState()
	sync.latestDataAddressesMutex.RLock()
	loaded := len(g.latestDataRepositoryAddresses) != 0
	sync.latestDataAddressesMutex.RUnlock()
	if loaded {
		return nil
	}
	return g.RefreshLatestDataRepositoryAddresses(ctx)
}

// collectTargetDataRepositories はReKyou/MiReKyouのターゲットになりうる
// 実データrepを集める。ReKyouもMiReKyouも含まない。
//
// collectNonReKyouRepositories と GetRepositoriesWithoutMiReKyouRep の
// 共通部分。片方だけにrep種別を足す事故を防ぐため1箇所にまとめてある。
// 「ReKyouの委譲先 == ここ + MiReKyou」という関係は
// ワード委譲検索のメモ(target_resolution_memo.go)が前提にしている。
func (g *GkillRepositories) collectTargetDataRepositories() Repositories {
	if g == nil {
		return nil
	}
	reps := Repositories{}
	for _, rep := range g.KmemoReps {
		reps = append(reps, rep)
	}
	for _, rep := range g.KCReps {
		reps = append(reps, rep)
	}
	for _, rep := range g.URLogReps {
		reps = append(reps, rep)
	}
	for _, rep := range g.NlogReps {
		reps = append(reps, rep)
	}
	for _, rep := range g.TimeIsReps {
		reps = append(reps, rep)
	}
	for _, rep := range g.MiReps {
		reps = append(reps, rep)
	}
	for _, rep := range g.LantanaReps {
		reps = append(reps, rep)
	}
	for _, rep := range g.IDFKyouReps {
		reps = append(reps, rep)
	}
	for _, rep := range g.GitCommitLogReps {
		reps = append(reps, rep)
	}
	return reps
}

// collectMiReKyouRepositories はMiReKyou repだけを集める。
func (g *GkillRepositories) collectMiReKyouRepositories() Repositories {
	if g == nil {
		return nil
	}
	reps := Repositories{}
	for _, rep := range g.MiReKyouReps.MiReKyouRepositories {
		reps = append(reps, rep)
	}
	return reps
}

// repsとLatestDataRepositoryAddressDAOのみ初期化済みのGkillRepositoriesを返す
func NewGkillRepositories(userID string) (*GkillRepositories, error) {
	ctx := context.Background()
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
	var CacheMemoryDBMutex *sync.RWMutex
	var CacheMemoryDB *sql.DB
	var TempMemoryDBMutex *sync.RWMutex
	var TempMemoryDB *sql.DB
	if gkill_options.IsCacheInMemory {
		CacheMemoryDB, err = sql.Open("sqlite", "file:gkill_memory_db_"+userID+"?mode=memory&cache=shared&_txlock=immediate&_pragma=busy_timeout(6000)&_pragma=journal_mode(MEMORY)&_pragma=synchronous(OFF)")
		if err != nil {
			err = fmt.Errorf("error at open memory database: %w", err)
			slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
		}
		CacheMemoryDB.SetMaxOpenConns(runtime.NumCPU()) // 読み取り並列を許可
		CacheMemoryDB.SetMaxIdleConns(1)                // 0にすると最後が閉じて消える
		CacheMemoryDB.SetConnMaxLifetime(0)             // 無限
		CacheMemoryDB.SetConnMaxIdleTime(0)             // 無限

		TempMemoryDB, err = sql.Open("sqlite", "file:gkill_temp_db_"+userID+"?mode=memory&cache=shared&_txlock=immediate&_pragma=busy_timeout(6000)&_pragma=journal_mode(MEMORY)&_pragma=synchronous(OFF)")
		if err != nil {
			err = fmt.Errorf("error at open memory database: %w", err)
			slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
		}
		TempMemoryDB.SetMaxOpenConns(runtime.NumCPU()) // 読み取り並列を許可
		TempMemoryDB.SetMaxIdleConns(1)                // 0にすると最後が閉じて消える
		TempMemoryDB.SetConnMaxLifetime(0)             // 無限
		TempMemoryDB.SetConnMaxIdleTime(0)             // 無限

		CacheMemoryDBMutex = &sync.RWMutex{}
		TempMemoryDBMutex = &sync.RWMutex{}
	} else {
		tempCacheSubDir := os.ExpandEnv(filepath.Join(gkill_options.CacheDir, "temp_cache"))
		os.MkdirAll(tempCacheSubDir, os.ModePerm)
		TempMemoryDB, err = sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(tempCacheSubDir, userID+"_temp_.db"))+"?_pragma=busy_timeout(6000)&_pragma=synchronous(FULL)&_pragma=journal_mode(WAL)")
		if err != nil {
			err = fmt.Errorf("error at open database: %w", err)
			return nil, err
		}
		latestDataRepAddrCacheSubDir := os.ExpandEnv(filepath.Join(gkill_options.CacheDir, "latest_data_repository_address_cache"))
		os.MkdirAll(latestDataRepAddrCacheSubDir, os.ModePerm)
		CacheMemoryDB, err = sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(latestDataRepAddrCacheSubDir, userID+"_cache_.db"))+"?_pragma=busy_timeout(6000)&_pragma=synchronous(FULL)&_pragma=journal_mode(WAL)")
		if err != nil {
			err = fmt.Errorf("error at open database: %w", err)
			return nil, err
		}

		CacheMemoryDBMutex = &sync.RWMutex{}
		TempMemoryDBMutex = &sync.RWMutex{}
	}

	// メモリ上でやる
	latestDataRepositoryAddressDAO, err := gkill_cache.NewLatestDataRepositoryAddressSQLite3Impl(userID, CacheMemoryDB, CacheMemoryDBMutex)
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

		syncStateRef: newGkillRepositoriesSync(),

		TempMemoryDB:       TempMemoryDB,
		CacheMemoryDB:      CacheMemoryDB,
		TempMemoryDBMutex:  TempMemoryDBMutex,
		CacheMemoryDBMutex: CacheMemoryDBMutex,

		updateCacheTicker:                                   ticker,
		latestDataRepositoryAddresses:                       nil,
		lastUpdatedLatestDataRepositoryAddressCacheFindTime: time.Unix(0, 0),
	}

	syncState := repositories.syncState()
	go func() {
		defer ticker.Stop()
		for {
			// Closeで停止したtickerは二度と発火しないため、
			// <-ticker.C だけで待つとこのgoroutineが永久に残る。
			// closeChも一緒に待って確実に抜ける。
			select {
			case <-syncState.closeCh:
				return
			case <-ticker.C:
			}
			if syncState.isClosed.Load() {
				return
			}
			currentGen := syncState.updateReqGen.Load()
			lastGen := syncState.lastHandled.Load()
			if currentGen == lastGen {
				continue
			}
			if syncState.isUpdateCacheNextTick.Load() {
				tickCtx, tickCancel := context.WithTimeout(context.Background(), 1*time.Hour)
				err := repositories.UpdateCache(tickCtx)
				tickCancel()
				if err != nil {
					slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
					continue
				}
				syncState.isUpdateCacheNextTick.Store(false)
				syncState.lastHandled.Store(currentGen)
			}
		}
	}()
	return repositories, nil
}

func (g *GkillRepositories) GetUserID(ctx context.Context) (string, error) {
	return g.userID, nil
}

func (g *GkillRepositories) Close(ctx context.Context) error {
	syncState := g.syncState()
	// 二重Closeでcloseチャネルを2回閉じてpanicしないよう、CASで1回だけ通す
	if syncState.isClosed.CompareAndSwap(false, true) {
		close(syncState.closeCh)
	}
	if g.updateCacheTicker != nil {
		g.updateCacheTicker.Stop()
	}
	if err := g.TagReps.Close(ctx); err != nil {
		slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
		return err
	}
	if err := g.TextReps.Close(ctx); err != nil {
		slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
		return err
	}
	if err := g.KmemoReps.Close(ctx); err != nil {
		slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
		return err
	}
	if err := g.KCReps.Close(ctx); err != nil {
		slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
		return err
	}
	if err := g.NlogReps.Close(ctx); err != nil {
		slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
		return err
	}
	if err := g.TimeIsReps.Close(ctx); err != nil {
		slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
		return err
	}

	if err := g.MiReps.Close(ctx); err != nil {
		slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
		return err
	}
	if err := g.IDFKyouReps.Close(ctx); err != nil {
		slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
		return err
	}
	if err := g.ReKyouReps.Close(ctx); err != nil {
		slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
		return err
	}
	if err := g.MiReKyouReps.Close(ctx); err != nil {
		slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
		return err
	}
	if err := g.GitCommitLogReps.Close(ctx); err != nil {
		slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
		return err
	}
	if err := g.LantanaReps.Close(ctx); err != nil {
		slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
		return err
	}
	err := g.LatestDataRepositoryAddressDAO.Close(ctx)
	if err != nil {
		slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
	}

	g.CacheMemoryDB.Close()
	g.TempMemoryDB.Close()
	/*
		for _, rep := range g.GPSLogReps {
			err := rep.Close(ctx)
			if err != nil {
			slog.Log(ctx, gkill_log.Error, "error",  err)
			}
		}
	*/
	g.userID = ""

	return nil
}

func (g *GkillRepositories) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	var matchKyou *Kyou
	matchKyousInRep := []Kyou{}
	var err error

	latestDataRepositoryAddress, err := g.LatestDataRepositoryAddressDAO.GetLatestDataRepositoryAddress(ctx, id)
	if err != nil {
		err = fmt.Errorf("error at get latest data repository addresses by id %s: %w", id, err)
		return nil, err
	}

	repImpls, err := g.Reps.UnWrap()
	if err != nil {
		return nil, err
	}

	for _, rep := range repImpls {
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
		matchKyousInRep = append(matchKyousInRep, *matchKyouInRep)
	}

	// Kyou集約。UpdateTimeが最新のものを収める
	for _, matchKyouInRep := range matchKyousInRep {
		if matchKyou != nil {
			if matchKyouInRep.UpdateTime.After(matchKyou.UpdateTime) {
				matchKyou = &matchKyouInRep
			}
		} else {
			matchKyou = &matchKyouInRep
		}
	}
	return matchKyou, nil
}

func (g *GkillRepositories) UpdateCache(ctx context.Context) error {
	select {
	case <-updateCacheThreadPool:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { updateCacheThreadPool <- struct{}{} }()
	updateCacheMutex := &g.syncState().updateCacheMutex
	updateCacheMutex.Lock()
	defer updateCacheMutex.Unlock()

	// UpdateCache実行中にfsnotifyがファイル変更を検出してUpdateCacheを再帰的にトリガーするのを防ぐ
	if g.SkipUpdateCache != nil {
		*g.SkipUpdateCache = true
		defer func() { *g.SkipUpdateCache = false }()
	}

	var cancelFunc context.CancelFunc

	// 一個前でUpdateCacheしてるやつをキャンセルする
	g.cancelPreFunc()
	ctx, cancelFunc = context.WithCancel(ctx)
	g.cancelPreFunc = cancelFunc

	var err error
	persistLatestDataRepositoryAddresses := func(addrs []gkill_cache.LatestDataRepositoryAddress, now time.Time) error {
		allLatestDataRepositoryAddresses := map[string]gkill_cache.LatestDataRepositoryAddress{}
		for _, addr := range addrs {
			if existing, exist := allLatestDataRepositoryAddresses[addr.TargetID]; !exist || existing.DataUpdateTime.Before(addr.DataUpdateTime) {
				addr.LatestDataRepositoryAddressUpdatedTime = now
				allLatestDataRepositoryAddresses[addr.TargetID] = addr
			}
		}

		latestDataRepositoryAddresses := make([]gkill_cache.LatestDataRepositoryAddress, 0, len(allLatestDataRepositoryAddresses))
		for _, addr := range allLatestDataRepositoryAddresses {
			latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, addr)
		}

		updatedLatestDataRepositoryAddresses, err := g.LatestDataRepositoryAddressDAO.ExtractUpdatedLatestDataRepositoryAddressDatas(ctx, latestDataRepositoryAddresses)
		if err != nil {
			return fmt.Errorf("error at update latest data repository address cache data: %w", err)
		}

		func() {
			sync := g.syncState()
			sync.latestDataAddressesMutex.Lock()
			defer sync.latestDataAddressesMutex.Unlock()

			if g.latestDataRepositoryAddresses == nil {
				g.latestDataRepositoryAddresses = map[string]gkill_cache.LatestDataRepositoryAddress{}
			}
			for _, updatedLatestDataRepositoryAddress := range updatedLatestDataRepositoryAddresses {
				g.latestDataRepositoryAddresses[updatedLatestDataRepositoryAddress.TargetID] = updatedLatestDataRepositoryAddress
			}
		}()

		_, err = g.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddresses(ctx, updatedLatestDataRepositoryAddresses)
		if err != nil {
			return fmt.Errorf("error at update latest data repository address cache data: %w", err)
		}

		func() {
			sync := g.syncState()
			sync.latestDataAddressesMutex.Lock()
			defer sync.latestDataAddressesMutex.Unlock()
			g.lastUpdatedLatestDataRepositoryAddressCacheFindTime = time.Now()
		}()
		return nil
	}

	// フェーズ別の所要時間を測る。実データでしか再現しない性能問題を
	// 本番のログだけで切り分けられるようにするため。
	timer := newUpdateCachePhaseTimer()
	defer timer.logUpdateCacheDone(ctx)

	// Phase 1: UpdateCacheは逐次実行する。並列にするとcachedDBのmutexでデッドロックが発生するため、意図的に逐次処理にしている。
	updateCacheTargets := []interface {
		UpdateCache(ctx context.Context) error
	}{}
	updateCacheTargets = append(updateCacheTargets, g.Reps)
	updateCacheTargets = append(updateCacheTargets, g.TagReps)
	updateCacheTargets = append(updateCacheTargets, g.TextReps)
	updateCacheTargets = append(updateCacheTargets, g.NotificationReps)
	for _, rep := range updateCacheTargets {
		err = rep.UpdateCache(ctx)
		if err != nil {
			return err
		}
	}
	timer.mark("phase1")

	// Phase 2: GetLatestDataRepositoryAddressで最新データアドレスを収集
	allLatestDataRepositoryAddresses := map[string]gkill_cache.LatestDataRepositoryAddress{}
	now := time.Now()

	getAddrTargets := []interface {
		GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error)
	}{}
	getAddrTargets = append(getAddrTargets, g.Reps)
	getAddrTargets = append(getAddrTargets, g.TagReps)
	getAddrTargets = append(getAddrTargets, g.TextReps)
	getAddrTargets = append(getAddrTargets, g.NotificationReps)

	for _, target := range getAddrTargets {
		addrs, err := target.GetLatestDataRepositoryAddress(ctx, false)
		if err != nil {
			return err
		}
		for _, addr := range addrs {
			if existing, exist := allLatestDataRepositoryAddresses[addr.TargetID]; !exist || existing.DataUpdateTime.Before(addr.DataUpdateTime) {
				addr.LatestDataRepositoryAddressUpdatedTime = now
				allLatestDataRepositoryAddresses[addr.TargetID] = addr
			}
		}
	}

	latestDataRepositoryAddresses := make([]gkill_cache.LatestDataRepositoryAddress, 0, len(allLatestDataRepositoryAddresses))
	for _, addr := range allLatestDataRepositoryAddresses {
		latestDataRepositoryAddresses = append(latestDataRepositoryAddresses, addr)
	}
	if err := persistLatestDataRepositoryAddresses(latestDataRepositoryAddresses, now); err != nil {
		return err
	}
	timer.mark("phase2")

	if gkill_options.CacheReKyouReps != nil && *gkill_options.CacheReKyouReps && len(g.ReKyouReps.ReKyouRepositories) > 0 {
		if err := g.ReKyouReps.UpdateCache(ctx); err != nil {
			return err
		}

		reKyouAddrs, err := g.ReKyouReps.GetLatestDataRepositoryAddress(ctx, false)
		if err != nil {
			return err
		}
		if err := persistLatestDataRepositoryAddresses(reKyouAddrs, time.Now()); err != nil {
			return err
		}
	}
	timer.mark("rekyou")

	// MiReKyouもReKyouと同じくターゲット解決のため他repのアドレス確定後に更新する
	if gkill_options.CacheMiReKyouReps != nil && *gkill_options.CacheMiReKyouReps && len(g.MiReKyouReps.MiReKyouRepositories) > 0 {
		if err := g.MiReKyouReps.UpdateCache(ctx); err != nil {
			return err
		}

		miReKyouAddrs, err := g.MiReKyouReps.GetLatestDataRepositoryAddress(ctx, false)
		if err != nil {
			return err
		}
		if err := persistLatestDataRepositoryAddresses(miReKyouAddrs, time.Now()); err != nil {
			return err
		}
	}
	timer.mark("mirekyou")

	if _, err := g.CacheMemoryDB.Exec(`ANALYZE;`); err != nil {
		return fmt.Errorf("ANALYZE: %w", err)
	}
	if _, err := g.CacheMemoryDB.Exec(`PRAGMA optimize;`); err != nil {
		return fmt.Errorf("PRAGMA optimize: %w", err)
	}
	timer.mark("analyze")
	return nil
}

func (g *GkillRepositories) UpdateCacheNextTick() {
	syncState := g.syncState()
	syncState.updateReqGen.Add(1)
	syncState.isUpdateCacheNextTick.Store(true)
}

func (g *GkillRepositories) GetPath(ctx context.Context, id string) (string, error) {
	// 並列処理
	matchPaths := []string{}

	ids := []string{id}
	for _, rep := range g.Reps {
		query := &find.FindQuery{
			IDs:    ids,
			UseIDs: true,
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

func (g *GkillRepositories) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	kyouHistories := map[string]Kyou{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Kyou, len(g.Reps))
	errch := make(chan error, len(g.Reps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.Reps {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchKyousInRep, err := rep.GetKyouHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchKyousInRep
		})
		if err != nil {
			errch <- err
		}
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

	kyouHistoriesList := []Kyou{}
	for _, kyou := range kyouHistories {

		kyouHistoriesList = append(kyouHistoriesList, kyou)
	}

	slices.SortFunc(kyouHistoriesList, func(a, b Kyou) int {
		return b.UpdateTime.Compare(a.UpdateTime)
	})

	return kyouHistoriesList, nil
}

func (g *GkillRepositories) FindTags(ctx context.Context, query *find.FindQuery) ([]Tag, error) {
	matchTags := map[string]Tag{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Tag, len(g.TagReps))
	errch := make(chan error, len(g.TagReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.TagReps {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			// 各repのgoroutineが同じ*FindQueryを書き換えると競合するので、構造体を値コピーする
			clonedQuery := *query
			queryLatest := &clonedQuery

			// idsを指定されていなければ、最新であるもののIDのみを対象とする
			if query.IDs == nil || len(query.IDs) == 0 {
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

				queryLatest.IDs = ids
				queryLatest.UseIDs = true
			}

			matchTagsInRep, err := rep.FindTags(ctx, queryLatest)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTagsInRep
		})
		if err != nil {
			errch <- err
		}
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
				if !query.OnlyLatestData {
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

	matchTagsList := []Tag{}
	for _, tag := range matchTags {
		if tag.IsDeleted {
			continue
		}
		matchTagsList = append(matchTagsList, tag)
	}

	slices.SortFunc(matchTagsList, func(a, b Tag) int {
		return b.RelatedTime.Compare(a.RelatedTime)
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
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchTagInRep, err := rep.GetTag(ctx, id, updateTime)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTagInRep
		})
		if err != nil {
			errch <- err
		}
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

func (g *GkillRepositories) GetTagsByTagName(ctx context.Context, tagname string) ([]Tag, error) {
	matchTags := map[string]Tag{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Tag, len(g.TagReps))
	errch := make(chan error, len(g.TagReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.TagReps {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchTagsInRep, err := rep.GetTagsByTagName(ctx, tagname)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTagsInRep
		})
		if err != nil {
			errch <- err
		}
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get tags by tag name: %w", e)
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

	tagHistoriesList := []Tag{}
	for _, tag := range matchTags {
		if tag.IsDeleted {
			continue
		}
		tagHistoriesList = append(tagHistoriesList, tag)
	}

	slices.SortFunc(tagHistoriesList, func(a, b Tag) int {
		return b.UpdateTime.Compare(a.UpdateTime)
	})

	return tagHistoriesList, nil

}

func (g *GkillRepositories) GetTagsByTargetID(ctx context.Context, target_id string) ([]Tag, error) {
	matchTags, err := g.TagReps.GetTagsByTargetID(ctx, target_id)
	if err != nil {
		err = fmt.Errorf("error at get tags by target id: %w", err)
		return nil, err
	}

	tagHistoriesList := []Tag{}
	for _, tag := range matchTags {
		if tag.IsDeleted {
			continue
		}
		tagHistoriesList = append(tagHistoriesList, tag)
	}

	slices.SortFunc(tagHistoriesList, func(a, b Tag) int {
		return b.UpdateTime.Compare(a.UpdateTime)
	})

	return tagHistoriesList, nil
}

func (g *GkillRepositories) GetTagHistories(ctx context.Context, id string) ([]Tag, error) {
	tagHistories := map[string]Tag{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Tag, len(g.TagReps))
	errch := make(chan error, len(g.TagReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.TagReps {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchTagsInRep, err := rep.GetTagHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTagsInRep
		})
		if err != nil {
			errch <- err
		}
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

	tagHistoriesList := []Tag{}
	for _, tag := range tagHistories {
		if tag.IsDeleted {
			continue
		}
		tagHistoriesList = append(tagHistoriesList, tag)
	}

	slices.SortFunc(tagHistoriesList, func(a, b Tag) int {
		return b.UpdateTime.Compare(a.UpdateTime)
	})

	return tagHistoriesList, nil
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
		rep := rep
		err := threads.Go(ctx, wg, func() {
			repName, err := rep.GetRepName(ctx)
			if err != nil {
				errch <- err
				return
			}
			ch <- repName
		})
		if err != nil {
			errch <- err
		}
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

	slices.SortFunc(repNamesList, func(a, b string) int {
		return cmp.Compare(a, b)
	})

	return repNamesList, nil
}

func (g *GkillRepositories) FindTexts(ctx context.Context, query *find.FindQuery) ([]Text, error) {
	matchTexts := map[string]Text{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Text, len(g.TextReps))
	errch := make(chan error, len(g.TextReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.TextReps {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			// jsonからパースする
			// 各repのgoroutineが同じ*FindQueryを書き換えると競合するので、構造体を値コピーする
			clonedQuery := *query
			queryLatest := &clonedQuery
			ids := []string{}
			if query.IDs != nil {
				ids = append([]string{}, query.IDs...)
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

				queryLatest.IDs = ids
				queryLatest.UseIDs = true
			}

			matchTextsInRep, err := rep.FindTexts(ctx, queryLatest)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTextsInRep
		})
		if err != nil {
			errch <- err
		}
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
				if !query.OnlyLatestData {
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

	matchTextsList := []Text{}
	for _, text := range matchTexts {
		if text.IsDeleted {
			continue
		}
		matchTextsList = append(matchTextsList, text)
	}

	slices.SortFunc(matchTextsList, func(a, b Text) int {
		return b.RelatedTime.Compare(a.RelatedTime)
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
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchTextInRep, err := rep.GetText(ctx, id, updateTime)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTextInRep
		})
		if err != nil {
			errch <- err
		}
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
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchNotificationInRep, err := rep.GetNotification(ctx, id, updateTime)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchNotificationInRep
		})
		if err != nil {
			errch <- err
		}
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

func (g *GkillRepositories) GetTextsByTargetID(ctx context.Context, target_id string) ([]Text, error) {
	matchTexts := map[string]Text{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Text, len(g.TextReps))
	errch := make(chan error, len(g.TextReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.TextReps {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchTextsInRep, err := rep.GetTextsByTargetID(ctx, target_id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTextsInRep
		})
		if err != nil {
			errch <- err
		}
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get texts by target id: %w", e)
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
					// キーはIDだけにする。ID+UpdateTimeにすると版ごとに別エントリになり、
					// 上の「UpdateTimeが新しい方で上書き」がいつまでも成立しないため、
					// 編集すると新旧が両方出て、削除しても削除前の版が残っていた。
					matchTexts[text.ID] = text
				}
			}
		default:
			break loop
		}
	}

	textHistoriesList := []Text{}
	for _, text := range matchTexts {
		if text.IsDeleted {
			continue
		}
		textHistoriesList = append(textHistoriesList, text)
	}

	slices.SortFunc(textHistoriesList, func(a, b Text) int {
		return b.UpdateTime.Compare(a.UpdateTime)
	})

	return textHistoriesList, nil
}

func (g *GkillRepositories) GetNotificationsByTargetID(ctx context.Context, target_id string) ([]Notification, error) {
	matchNotifications := map[string]Notification{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Notification, len(g.NotificationReps))
	errch := make(chan error, len(g.NotificationReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.NotificationReps {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchNotificationsInRep, err := rep.GetNotificationsByTargetID(ctx, target_id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchNotificationsInRep
		})
		if err != nil {
			errch <- err
		}
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get notifications by target id: %w", e)
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
					// キーはIDだけにする。理由は GetTextsByTargetID と同じ。
					matchNotifications[notification.ID] = notification
				}
			}
		default:
			break loop
		}
	}

	notificationHistoriesList := []Notification{}
	for _, notification := range matchNotifications {
		if notification.IsDeleted {
			continue
		}
		notificationHistoriesList = append(notificationHistoriesList, notification)
	}

	slices.SortFunc(notificationHistoriesList, func(a, b Notification) int {
		return b.UpdateTime.Compare(a.UpdateTime)
	})

	return notificationHistoriesList, nil
}

func (g *GkillRepositories) GetTextHistories(ctx context.Context, id string) ([]Text, error) {
	textHistories := map[string]Text{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Text, len(g.TextReps))
	errch := make(chan error, len(g.TextReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.TextReps {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchTextsInRep, err := rep.GetTextHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchTextsInRep
		})
		if err != nil {
			errch <- err
		}
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

	textHistoriesList := []Text{}
	for _, text := range textHistories {
		if text.IsDeleted {
			continue
		}
		textHistoriesList = append(textHistoriesList, text)
	}

	slices.SortFunc(textHistoriesList, func(a, b Text) int {
		return b.UpdateTime.Compare(a.UpdateTime)
	})

	return textHistoriesList, nil
}

func (g *GkillRepositories) GetNotificationHistories(ctx context.Context, id string) ([]Notification, error) {
	notificationHistories := map[string]Notification{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []Notification, len(g.NotificationReps))
	errch := make(chan error, len(g.NotificationReps))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g.NotificationReps {
		rep := rep
		err := threads.Go(ctx, wg, func() {
			matchNotificationsInRep, err := rep.GetNotificationHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchNotificationsInRep
		})
		if err != nil {
			errch <- err
		}
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

	notificationHistoriesList := []Notification{}
	for _, notification := range notificationHistories {
		if notification.IsDeleted {
			continue
		}
		notificationHistoriesList = append(notificationHistoriesList, notification)
	}

	slices.SortFunc(notificationHistoriesList, func(a, b Notification) int {
		return b.UpdateTime.Compare(a.UpdateTime)
	})

	return notificationHistoriesList, nil
}
