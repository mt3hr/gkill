package dao

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"sync"

	"github.com/gorilla/mux"
	"github.com/mattn/go-zglob"
	"github.com/mt3hr/gkill/src/server/gkill/dao/account"
	"github.com/mt3hr/gkill/src/server/gkill/dao/account_state"
	"github.com/mt3hr/gkill/src/server/gkill/dao/gkill_notification"
	"github.com/mt3hr/gkill/src/server/gkill/dao/hide_files"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps/rep_cache_updater"
	"github.com/mt3hr/gkill/src/server/gkill/dao/server_config"
	"github.com/mt3hr/gkill/src/server/gkill/dao/share_kyou_info"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

type GkillDAOManager struct {
	// stateMutex は下記4マップ自体へのアクセスを保護する。
	// 同時リクエストでマップの読み書きが競合すると
	// concurrent map read and map write でプロセスが即死するため必須。
	// 時間のかかるリポジトリ構築中は保持しない。構築の排他は initializingMutex が担う。
	stateMutex               sync.Mutex
	initializingMutex        map[string]map[string]*sync.RWMutex
	gkillRepositories        map[string]map[string]*reps.GkillRepositories
	gkillNotificators        map[string]map[string]*GkillNotificator
	fileRepWatchCacheUpdater rep_cache_updater.FileRepCacheUpdater

	// pluginManagers はユーザID別のPluginManager。
	// GetRepositories 時に遅延初期化される。
	pluginManagers map[string]*PluginManager

	ConfigDAOs *ConfigDAOs

	router    *mux.Router
	IDFIgnore []string

	skipUpdateCache *bool
}

func NewGkillDAOManager() (*GkillDAOManager, error) {
	skipUpdateCache := false

	fileRepWatchCacheUpdater, err := rep_cache_updater.NewFileRepCacheUpdater(&skipUpdateCache)
	if err != nil {
		err = fmt.Errorf("error at new file rep cache updater: %w", err)
		return nil, err
	}

	ctx := context.Background()
	gkillDAOManager := &GkillDAOManager{
		router:                   &mux.Router{},
		IDFIgnore:                gkill_options.IDFIgnore,
		fileRepWatchCacheUpdater: fileRepWatchCacheUpdater,
		skipUpdateCache:          &skipUpdateCache,
	}

	configDBRootDir := os.ExpandEnv(gkill_options.ConfigDir)
	err = os.MkdirAll(os.ExpandEnv(configDBRootDir), fs.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at create directory %s: %w", os.ExpandEnv(configDBRootDir), err)
		return nil, err
	}

	gkillDAOManager.ConfigDAOs = &ConfigDAOs{}
	gkillDAOManager.ConfigDAOs.ServerConfigDAO, err = server_config.NewServerConfigDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "server_config.db"))
	if err != nil {
		return nil, err
	}
	gkillDAOManager.ConfigDAOs.AccountDAO, err = account.NewAccountDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "account.db"))
	if err != nil {
		return nil, err
	}
	gkillDAOManager.ConfigDAOs.LoginSessionDAO, err = account_state.NewLoginSessionDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "account_state.db"))
	if err != nil {
		return nil, err
	}
	gkillDAOManager.ConfigDAOs.FileUploadHistoryDAO, err = account_state.NewFileUploadHistoryDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "account_state.db"))
	if err != nil {
		return nil, err
	}
	gkillDAOManager.ConfigDAOs.ShareKyouInfoDAO, err = share_kyou_info.NewShareKyouInfoDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "share_kyou_info.db"))
	if err != nil {
		return nil, err
	}
	gkillDAOManager.ConfigDAOs.ApplicationConfigDAO, err = user_config.NewApplicationConfigDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "user_config.db"))
	if err != nil {
		return nil, err
	}
	gkillDAOManager.ConfigDAOs.RepositoryDAO, err = user_config.NewRepositoryDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "user_config.db"))
	if err != nil {
		return nil, err
	}
	gkillDAOManager.ConfigDAOs.GkillNotificationTargetDAO, err = gkill_notification.NewGkillNotificateTargetDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "gkill_notification_target.db"))
	if err != nil {
		return nil, err
	}

	// plugins/ ベースディレクトリを作成する
	pluginsBaseDir := filepath.Join(os.ExpandEnv(gkill_options.GkillHomeDir), "plugins")
	if err := os.MkdirAll(pluginsBaseDir, fs.ModePerm); err != nil {
		slog.Warn(fmt.Sprintf("plugins base dir create failed: %q", err))
	}

	// 全ユーザの plugins/{userID}/ ディレクトリを作成する
	accounts, err := gkillDAOManager.ConfigDAOs.AccountDAO.GetAllAccounts(ctx)
	if err != nil {
		slog.Warn(fmt.Sprintf("get all accounts failed (plugin dir creation skipped): %q", err))
	} else {
		for _, acc := range accounts {
			userPluginsDir := filepath.Join(pluginsBaseDir, acc.UserID)
			if err := os.MkdirAll(userPluginsDir, fs.ModePerm); err != nil {
				slog.Warn(fmt.Sprintf("plugin user dir create failed for %q: %q", acc.UserID, err))
				continue
			}
		}
	}

	return gkillDAOManager, nil
}

func (g *GkillDAOManager) GetRouter() *mux.Router {
	return g.router
}

// lookupRepositories は userID/device に対応する初期化用ミューテックスと、
// すでに構築済みならそのリポジトリを返す。
// マップが未初期化なら初期化し、ミューテックスがなければ作って登録する。
func (g *GkillDAOManager) lookupRepositories(userID string, device string) (*sync.RWMutex, *reps.GkillRepositories, bool) {
	g.stateMutex.Lock()
	defer g.stateMutex.Unlock()

	if g.initializingMutex == nil {
		g.initializingMutex = map[string]map[string]*sync.RWMutex{}
	}
	if g.gkillRepositories == nil {
		g.gkillRepositories = map[string]map[string]*reps.GkillRepositories{}
	}
	if _, exist := g.initializingMutex[userID]; !exist {
		g.initializingMutex[userID] = map[string]*sync.RWMutex{}
	}
	if _, exist := g.initializingMutex[userID][device]; !exist {
		g.initializingMutex[userID][device] = &sync.RWMutex{}
	}
	if _, exist := g.gkillRepositories[userID]; !exist {
		g.gkillRepositories[userID] = map[string]*reps.GkillRepositories{}
	}

	repositories, exist := g.gkillRepositories[userID][device]
	return g.initializingMutex[userID][device], repositories, exist
}

// storeRepositories は構築済みリポジトリをマップに登録する。
func (g *GkillDAOManager) storeRepositories(userID string, device string, repositories *reps.GkillRepositories) {
	g.stateMutex.Lock()
	defer g.stateMutex.Unlock()

	if g.gkillRepositories == nil {
		g.gkillRepositories = map[string]map[string]*reps.GkillRepositories{}
	}
	if _, exist := g.gkillRepositories[userID]; !exist {
		g.gkillRepositories[userID] = map[string]*reps.GkillRepositories{}
	}
	g.gkillRepositories[userID][device] = repositories
}

func (g *GkillDAOManager) GetRepositories(userID string, device string) (*reps.GkillRepositories, error) {
	if userID == "" || device == "" {
		err := fmt.Errorf("userID or device is blank. userID=%s device=%s", userID, device)
		return nil, err
	}

	ctx := context.Background()
	var err error

	// マップの初期化と参照。マップ自体の操作中だけ stateMutex を保持する
	initializingMutex, repositories, existRepsInDevice := g.lookupRepositories(userID, device)

	if !existRepsInDevice {
		// 初期化中だったら終わるまで待つ
		initializingMutex.Lock()
		defer initializingMutex.Unlock()

		// 初期化がおわり、値が入っていればそれを使う
		if _, repositories, exist := g.lookupRepositories(userID, device); exist {
			return repositories, nil
		}

		// なかったら作っていれる
		repositories, err = reps.NewGkillRepositories(userID)
		if err != nil {
			err = fmt.Errorf("error at new gkill repositories. user id = %s: %w", userID, err)
			return nil, err
		}
		repositories.SkipUpdateCache = g.skipUpdateCache
		repositories.ReKyouReps.GkillRepositories = repositories
		repositories.MiReKyouReps.GkillRepositories = repositories

		repositoriesDefine, err := g.ConfigDAOs.RepositoryDAO.GetRepositories(ctx, userID, device)
		if err != nil {
			err = fmt.Errorf("error at get repositories user=%s device=%s: %w", userID, device, err)
			return nil, err
		}

		// disableはあとから除外する
		disableReps := []string{}
		for _, rep := range repositoriesDefine {
			if rep.IsEnable {
				continue
			}
			disableReps = append(disableReps, filepath.Clean(os.ExpandEnv(rep.File)))
		}

		// 同一設定が二重登録されていたりglobどうしが重なっていたりすると、
		// 同じファイルに対するRepositoryが複数生成されてしまう。
		// そうなるとKyouが重複するうえ、ローカルキャッシュ有効時は
		// 同じキャッシュファイルを複数のRepが同時に開いて削除し合い、UpdateCacheが失敗する。
		// 解決後の (Type, ファイルパス) で重複排除する。
		// 書き込み用repが必ず残るよう、先に書き込み用を前に寄せておく。
		slices.SortStableFunc(repositoriesDefine, func(a, b *user_config.Repository) int {
			if a.UseToWrite == b.UseToWrite {
				return 0
			}
			if a.UseToWrite {
				return -1
			}
			return 1
		})
		loadedRepKeys := map[string]struct{}{}

		for _, rep := range repositoriesDefine {
			if !rep.IsEnable {
				continue
			}
			rep.File = os.ExpandEnv(rep.File)

			matchFiles, _ := zglob.Glob(rep.File)
			sort.Strings(matchFiles)
			for _, filename := range matchFiles {
				filename = filepath.Clean(filename)
				isSkipLoop := false
				for _, disableRep := range disableReps {
					if match, _ := zglob.Match(disableRep, filename); match {
						isSkipLoop = true
						break
					}
				}
				if isSkipLoop {
					continue
				}

				loadedRepKey := rep.Type + "\x00" + filename
				if _, loaded := loadedRepKeys[loadedRepKey]; loaded {
					slog.Log(ctx, gkill_log.Debug, "skip duplicated repository define", "userID", fmt.Sprintf("%q", userID), "device", fmt.Sprintf("%q", device), "type", fmt.Sprintf("%q", rep.Type), "file", fmt.Sprintf("%q", filename))
					continue
				}
				loadedRepKeys[loadedRepKey] = struct{}{}

				parentDir := filepath.Dir(filename)
				err := os.MkdirAll(os.ExpandEnv(parentDir), os.ModePerm)
				if err != nil {
					err = fmt.Errorf("error at make directory %s: %w", parentDir, err)
					return nil, err
				}

				switch rep.Type {
				case "kmemo":
					if gkill_options.LoadIDFRepOnly {
						continue
					}
					var kmemoRep reps.KmemoRepository
					if !rep.UseToWrite && gkill_options.CacheRepsLocalStorage {
						kmemoRep, err = reps.NewKmemoRepositorySQLite3ImplLocalCached(ctx, userID, filename, rep.UseToWrite)
					} else {
						kmemoRep, err = reps.NewKmemoRepositorySQLite3Impl(ctx, filename, rep.UseToWrite)
					}
					if err != nil {
						return nil, err
					}
					repositories.KmemoReps = append(repositories.KmemoReps, kmemoRep)
					if rep.UseToWrite {
						newPath, _ := kmemoRep.GetPath(ctx, "")
						if repositories.WriteKmemoRep != nil {
							existPath, _ := repositories.WriteKmemoRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write kmemo rep %s %s", existPath, newPath)
							return nil, err
						}
						repositories.WriteKmemoRep = kmemoRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if rep.IsWatchTargetForUpdateRep {
						rep := kmemoRep
						enableUpdateRepsCache := false
						enableUpdateLatestDataRepositoryCache := true
						cacheUpdater := rep_cache_updater.NewLatestRepositoryAddressCacheUpdater(rep, repositories, enableUpdateRepsCache, enableUpdateLatestDataRepositoryCache)
						ignoreFileNamePrefixes := []string{}
						repFilename, err := rep.GetPath(ctx, "")
						if err != nil {
							repName, _ := rep.GetRepName(ctx)
							err = fmt.Errorf("error at get path. repname = %s: %w", repName, err)
							return nil, err
						}
						repFilename = filepath.ToSlash(repFilename)

						err = g.fileRepWatchCacheUpdater.RegisterWatchFileRep(cacheUpdater, repFilename, ignoreFileNamePrefixes, userID)
						if err != nil {
							err = fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "kc":
					if gkill_options.LoadIDFRepOnly {
						continue
					}
					var kcRep reps.KCRepository
					if !rep.UseToWrite && gkill_options.CacheRepsLocalStorage {
						kcRep, err = reps.NewKCRepositorySQLite3ImplLocalCached(ctx, userID, filename, rep.UseToWrite)
					} else {
						kcRep, err = reps.NewKCRepositorySQLite3Impl(ctx, filename, rep.UseToWrite)
					}
					if err != nil {
						return nil, err
					}
					repositories.KCReps = append(repositories.KCReps, kcRep)
					if rep.UseToWrite {
						newPath, _ := kcRep.GetPath(ctx, "")
						if repositories.WriteKCRep != nil {
							existPath, _ := repositories.WriteKCRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write kc rep %s %s", existPath, newPath)
							return nil, err
						}
						repositories.WriteKCRep = kcRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if rep.IsWatchTargetForUpdateRep {
						rep := kcRep
						enableUpdateRepsCache := false
						enableUpdateLatestDataRepositoryCache := true
						cacheUpdater := rep_cache_updater.NewLatestRepositoryAddressCacheUpdater(rep, repositories, enableUpdateRepsCache, enableUpdateLatestDataRepositoryCache)
						ignoreFileNamePrefixes := []string{}
						repFilename, err := rep.GetPath(ctx, "")
						if err != nil {
							repName, _ := rep.GetRepName(ctx)
							err = fmt.Errorf("error at get path. repname = %s: %w", repName, err)
							return nil, err
						}
						repFilename = filepath.ToSlash(repFilename)

						err = g.fileRepWatchCacheUpdater.RegisterWatchFileRep(cacheUpdater, repFilename, ignoreFileNamePrefixes, userID)
						if err != nil {
							err = fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "urlog":
					if gkill_options.LoadIDFRepOnly {
						continue
					}
					var urlogRep reps.URLogRepository
					if !rep.UseToWrite && gkill_options.CacheRepsLocalStorage {
						urlogRep, err = reps.NewURLogRepositorySQLite3ImplLocalCached(ctx, userID, filename, rep.UseToWrite)
					} else {
						urlogRep, err = reps.NewURLogRepositorySQLite3Impl(ctx, filename, rep.UseToWrite)
					}
					if err != nil {
						return nil, err
					}
					repositories.URLogReps = append(repositories.URLogReps, urlogRep)
					if rep.UseToWrite {
						newPath, _ := urlogRep.GetPath(ctx, "")
						if repositories.WriteURLogRep != nil {
							existPath, _ := repositories.WriteURLogRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write urlog rep %s %s", existPath, newPath)
							return nil, err
						}
						repositories.WriteURLogRep = urlogRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if rep.IsWatchTargetForUpdateRep {
						rep := urlogRep
						enableUpdateRepsCache := false
						enableUpdateLatestDataRepositoryCache := true
						cacheUpdater := rep_cache_updater.NewLatestRepositoryAddressCacheUpdater(rep, repositories, enableUpdateRepsCache, enableUpdateLatestDataRepositoryCache)
						ignoreFileNamePrefixes := []string{}
						repFilename, err := rep.GetPath(ctx, "")
						if err != nil {
							repName, _ := rep.GetRepName(ctx)
							err = fmt.Errorf("error at get path. repname = %s: %w", repName, err)
							return nil, err
						}
						repFilename = filepath.ToSlash(repFilename)

						err = g.fileRepWatchCacheUpdater.RegisterWatchFileRep(cacheUpdater, repFilename, ignoreFileNamePrefixes, userID)
						if err != nil {
							err = fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "timeis":
					if gkill_options.LoadIDFRepOnly {
						continue
					}
					var timeisRep reps.TimeIsRepository
					if !rep.UseToWrite && gkill_options.CacheRepsLocalStorage {
						timeisRep, err = reps.NewTimeIsRepositorySQLite3ImplLocalCached(ctx, userID, filename, rep.UseToWrite)
					} else {
						timeisRep, err = reps.NewTimeIsRepositorySQLite3Impl(ctx, filename, rep.UseToWrite)
					}
					if err != nil {
						return nil, err
					}
					repositories.TimeIsReps = append(repositories.TimeIsReps, timeisRep)
					if rep.UseToWrite {
						newPath, _ := timeisRep.GetPath(ctx, "")
						if repositories.WriteTimeIsRep != nil {
							existPath, _ := repositories.WriteTimeIsRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write timeis rep %s %s", existPath, newPath)
							return nil, err
						}
						repositories.WriteTimeIsRep = timeisRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if rep.IsWatchTargetForUpdateRep {
						rep := timeisRep
						enableUpdateRepsCache := false
						enableUpdateLatestDataRepositoryCache := true
						cacheUpdater := rep_cache_updater.NewLatestRepositoryAddressCacheUpdater(rep, repositories, enableUpdateRepsCache, enableUpdateLatestDataRepositoryCache)
						ignoreFileNamePrefixes := []string{}
						repFilename, err := rep.GetPath(ctx, "")
						if err != nil {
							repName, _ := rep.GetRepName(ctx)
							err = fmt.Errorf("error at get path. repname = %s: %w", repName, err)
							return nil, err
						}
						repFilename = filepath.ToSlash(repFilename)

						err = g.fileRepWatchCacheUpdater.RegisterWatchFileRep(cacheUpdater, repFilename, ignoreFileNamePrefixes, userID)
						if err != nil {
							err = fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "mi":
					if gkill_options.LoadIDFRepOnly {
						continue
					}
					var miRep reps.MiRepository
					if !rep.UseToWrite && gkill_options.CacheRepsLocalStorage {
						miRep, err = reps.NewMiRepositorySQLite3ImplLocalCached(ctx, userID, filename, rep.UseToWrite)
					} else {
						miRep, err = reps.NewMiRepositorySQLite3Impl(ctx, filename, rep.UseToWrite)
					}
					if err != nil {
						return nil, err
					}
					repositories.MiReps = append(repositories.MiReps, miRep)
					if rep.UseToWrite {
						newPath, _ := miRep.GetPath(ctx, "")
						if repositories.WriteMiRep != nil {
							existPath, _ := repositories.WriteMiRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write mi rep %s %s", existPath, newPath)
							return nil, err
						}
						repositories.WriteMiRep = miRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if rep.IsWatchTargetForUpdateRep {
						rep := miRep
						enableUpdateRepsCache := false
						enableUpdateLatestDataRepositoryCache := true
						cacheUpdater := rep_cache_updater.NewLatestRepositoryAddressCacheUpdater(rep, repositories, enableUpdateRepsCache, enableUpdateLatestDataRepositoryCache)
						ignoreFileNamePrefixes := []string{}
						repFilename, err := rep.GetPath(ctx, "")
						if err != nil {
							repName, _ := rep.GetRepName(ctx)
							err = fmt.Errorf("error at get path. repname = %s: %w", repName, err)
							return nil, err
						}
						repFilename = filepath.ToSlash(repFilename)

						err = g.fileRepWatchCacheUpdater.RegisterWatchFileRep(cacheUpdater, repFilename, ignoreFileNamePrefixes, userID)
						if err != nil {
							err = fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "nlog":
					if gkill_options.LoadIDFRepOnly {
						continue
					}
					var nlogRep reps.NlogRepository
					if !rep.UseToWrite && gkill_options.CacheRepsLocalStorage {
						nlogRep, err = reps.NewNlogRepositorySQLite3ImplLocalCached(ctx, userID, filename, rep.UseToWrite)
					} else {
						nlogRep, err = reps.NewNlogRepositorySQLite3Impl(ctx, filename, rep.UseToWrite)
					}
					if err != nil {
						return nil, err
					}
					repositories.NlogReps = append(repositories.NlogReps, nlogRep)
					if rep.UseToWrite {
						newPath, _ := nlogRep.GetPath(ctx, "")
						if repositories.WriteNlogRep != nil {
							existPath, _ := repositories.WriteNlogRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write nlog rep %s %s", existPath, newPath)
							return nil, err
						}
						repositories.WriteNlogRep = nlogRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if rep.IsWatchTargetForUpdateRep {
						rep := nlogRep
						enableUpdateRepsCache := false
						enableUpdateLatestDataRepositoryCache := true
						cacheUpdater := rep_cache_updater.NewLatestRepositoryAddressCacheUpdater(rep, repositories, enableUpdateRepsCache, enableUpdateLatestDataRepositoryCache)
						ignoreFileNamePrefixes := []string{}
						repFilename, err := rep.GetPath(ctx, "")
						if err != nil {
							repName, _ := rep.GetRepName(ctx)
							err = fmt.Errorf("error at get path. repname = %s: %w", repName, err)
							return nil, err
						}
						repFilename = filepath.ToSlash(repFilename)

						err = g.fileRepWatchCacheUpdater.RegisterWatchFileRep(cacheUpdater, repFilename, ignoreFileNamePrefixes, userID)
						if err != nil {
							err = fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "lantana":
					if gkill_options.LoadIDFRepOnly {
						continue
					}
					var lantanaRep reps.LantanaRepository
					if !rep.UseToWrite && gkill_options.CacheRepsLocalStorage {
						lantanaRep, err = reps.NewLantanaRepositorySQLite3ImplLocalCached(ctx, userID, filename, rep.UseToWrite)
					} else {
						lantanaRep, err = reps.NewLantanaRepositorySQLite3Impl(ctx, filename, rep.UseToWrite)
					}
					if err != nil {
						return nil, err
					}
					repositories.LantanaReps = append(repositories.LantanaReps, lantanaRep)
					if rep.UseToWrite {
						newPath, _ := lantanaRep.GetPath(ctx, "")
						if repositories.WriteLantanaRep != nil {
							existPath, _ := repositories.WriteLantanaRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write lantana rep %s %s", existPath, newPath)
							return nil, err
						}
						repositories.WriteLantanaRep = lantanaRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if rep.IsWatchTargetForUpdateRep {
						rep := lantanaRep
						enableUpdateRepsCache := false
						enableUpdateLatestDataRepositoryCache := true
						cacheUpdater := rep_cache_updater.NewLatestRepositoryAddressCacheUpdater(rep, repositories, enableUpdateRepsCache, enableUpdateLatestDataRepositoryCache)
						ignoreFileNamePrefixes := []string{}
						repFilename, err := rep.GetPath(ctx, "")
						if err != nil {
							repName, _ := rep.GetRepName(ctx)
							err = fmt.Errorf("error at get path. repname = %s: %w", repName, err)
							return nil, err
						}
						repFilename = filepath.ToSlash(repFilename)

						err = g.fileRepWatchCacheUpdater.RegisterWatchFileRep(cacheUpdater, repFilename, ignoreFileNamePrefixes, userID)
						if err != nil {
							err = fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "tag":
					if gkill_options.LoadIDFRepOnly {
						continue
					}
					var tagRep reps.TagRepository
					if !rep.UseToWrite && gkill_options.CacheRepsLocalStorage {
						tagRep, err = reps.NewTagRepositorySQLite3ImplLocalCached(ctx, userID, filename, rep.UseToWrite)
					} else {
						tagRep, err = reps.NewTagRepositorySQLite3Impl(ctx, filename, rep.UseToWrite)
					}
					if err != nil {
						return nil, err
					}
					repositories.TagReps = append(repositories.TagReps, tagRep)
					// 第1引数は TagRepsWatchTarget であること。
					// TagReps を渡すと、直前の行で足した tagRep が重複するうえ、
					// TagReps と backing array を共有して互いに干渉する。
					repositories.TagRepsWatchTarget = append(repositories.TagRepsWatchTarget, tagRep)
					if rep.UseToWrite {
						newPath, _ := tagRep.GetPath(ctx, "")
						if repositories.WriteTagRep != nil {
							existPath, _ := repositories.WriteTagRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write tag rep %s %s", existPath, newPath)
							return nil, err
						}
						repositories.WriteTagRep = tagRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if rep.IsWatchTargetForUpdateRep {
						rep := tagRep
						enableUpdateRepsCache := false
						enableUpdateLatestDataRepositoryCache := true
						cacheUpdater := rep_cache_updater.NewLatestRepositoryAddressCacheUpdater(rep, repositories, enableUpdateRepsCache, enableUpdateLatestDataRepositoryCache)
						ignoreFileNamePrefixes := []string{}
						repFilename, err := rep.GetPath(ctx, "")
						if err != nil {
							repName, _ := rep.GetRepName(ctx)
							err = fmt.Errorf("error at get path. repname = %s: %w", repName, err)
							return nil, err
						}
						repFilename = filepath.ToSlash(repFilename)

						err = g.fileRepWatchCacheUpdater.RegisterWatchFileRep(cacheUpdater, repFilename, ignoreFileNamePrefixes, userID)
						if err != nil {
							err = fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "text":
					if gkill_options.LoadIDFRepOnly {
						continue
					}
					var textRep reps.TextRepository
					if !rep.UseToWrite && gkill_options.CacheRepsLocalStorage {
						textRep, err = reps.NewTextRepositorySQLite3ImplLocalCached(ctx, userID, filename, rep.UseToWrite)
					} else {
						textRep, err = reps.NewTextRepositorySQLite3Impl(ctx, filename, rep.UseToWrite)
					}
					if err != nil {
						return nil, err
					}
					repositories.TextReps = append(repositories.TextReps, textRep)
					// 第1引数は TextRepsWatchTarget であること（TagReps側と同じ理由）
					repositories.TextRepsWatchTarget = append(repositories.TextRepsWatchTarget, textRep)
					if rep.UseToWrite {
						newPath, _ := textRep.GetPath(ctx, "")
						if repositories.WriteTextRep != nil {
							existPath, _ := repositories.WriteTextRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write text rep %s %s", existPath, newPath)
							return nil, err
						}
						repositories.WriteTextRep = textRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if rep.IsWatchTargetForUpdateRep {
						rep := textRep
						enableUpdateRepsCache := false
						enableUpdateLatestDataRepositoryCache := true
						cacheUpdater := rep_cache_updater.NewLatestRepositoryAddressCacheUpdater(rep, repositories, enableUpdateRepsCache, enableUpdateLatestDataRepositoryCache)
						ignoreFileNamePrefixes := []string{}
						repFilename, err := rep.GetPath(ctx, "")
						if err != nil {
							repName, _ := rep.GetRepName(ctx)
							err = fmt.Errorf("error at get path. repname = %s: %w", repName, err)
							return nil, err
						}
						repFilename = filepath.ToSlash(repFilename)

						err = g.fileRepWatchCacheUpdater.RegisterWatchFileRep(cacheUpdater, repFilename, ignoreFileNamePrefixes, userID)
						if err != nil {
							err = fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "notification":
					if gkill_options.LoadIDFRepOnly {
						continue
					}
					var notificationRep reps.NotificationRepository
					if !rep.UseToWrite && gkill_options.CacheRepsLocalStorage {
						notificationRep, err = reps.NewNotificationRepositorySQLite3ImplLocalCached(ctx, userID, filename, rep.UseToWrite)
					} else {
						notificationRep, err = reps.NewNotificationRepositorySQLite3Impl(ctx, filename, rep.UseToWrite)
					}
					if err != nil {
						return nil, err
					}
					repositories.NotificationReps = append(repositories.NotificationReps, notificationRep)
					if rep.UseToWrite {
						newPath, _ := notificationRep.GetPath(ctx, "")
						if repositories.WriteNotificationRep != nil {
							existPath, _ := repositories.WriteNotificationRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write notification rep %s %s", existPath, newPath)
							return nil, err
						}
						repositories.WriteNotificationRep = notificationRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if rep.IsWatchTargetForUpdateRep {
						rep := notificationRep
						enableUpdateRepsCache := false
						enableUpdateLatestDataRepositoryCache := true
						cacheUpdater := rep_cache_updater.NewLatestRepositoryAddressCacheUpdater(rep, repositories, enableUpdateRepsCache, enableUpdateLatestDataRepositoryCache)
						ignoreFileNamePrefixes := []string{}
						repFilename, err := rep.GetPath(ctx, "")
						if err != nil {
							repName, _ := rep.GetRepName(ctx)
							err = fmt.Errorf("error at get path. repname = %s: %w", repName, err)
							return nil, err
						}
						repFilename = filepath.ToSlash(repFilename)

						err = g.fileRepWatchCacheUpdater.RegisterWatchFileRep(cacheUpdater, repFilename, ignoreFileNamePrefixes, userID)
						if err != nil {
							err = fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "rekyou":
					if gkill_options.LoadIDFRepOnly {
						continue
					}
					var reKyouRep reps.ReKyouRepository
					if !rep.UseToWrite && gkill_options.CacheRepsLocalStorage {
						reKyouRep, err = reps.NewReKyouRepositorySQLite3ImplLocalCached(ctx, userID, filename, rep.UseToWrite, repositories)
					} else {
						reKyouRep, err = reps.NewReKyouRepositorySQLite3Impl(ctx, filename, rep.UseToWrite, repositories)
					}
					if err != nil {
						return nil, err
					}
					repositories.ReKyouReps.ReKyouRepositories = append(repositories.ReKyouReps.ReKyouRepositories, reKyouRep)
					if rep.UseToWrite {
						newPath, _ := reKyouRep.GetPath(ctx, "")
						if repositories.WriteReKyouRep != nil {
							existPath, _ := repositories.WriteReKyouRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write reKyou rep %s %s", existPath, newPath)
							return nil, err
						}
						repositories.WriteReKyouRep = reKyouRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if rep.IsWatchTargetForUpdateRep {
						rep := reKyouRep
						enableUpdateRepsCache := false
						enableUpdateLatestDataRepositoryCache := true
						cacheUpdater := rep_cache_updater.NewLatestRepositoryAddressCacheUpdater(rep, repositories, enableUpdateRepsCache, enableUpdateLatestDataRepositoryCache)
						ignoreFileNamePrefixes := []string{}
						repFilename, err := rep.GetPath(ctx, "")
						if err != nil {
							repName, _ := rep.GetRepName(ctx)
							err = fmt.Errorf("error at get path. repname = %s: %w", repName, err)
							return nil, err
						}
						repFilename = filepath.ToSlash(repFilename)

						err = g.fileRepWatchCacheUpdater.RegisterWatchFileRep(cacheUpdater, repFilename, ignoreFileNamePrefixes, userID)
						if err != nil {
							err = fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "mirekyou":
					if gkill_options.LoadIDFRepOnly {
						continue
					}
					var miReKyouRep reps.MiReKyouRepository
					if !rep.UseToWrite && gkill_options.CacheRepsLocalStorage {
						miReKyouRep, err = reps.NewMiReKyouRepositorySQLite3ImplLocalCached(ctx, userID, filename, rep.UseToWrite, repositories)
					} else {
						miReKyouRep, err = reps.NewMiReKyouRepositorySQLite3Impl(ctx, filename, rep.UseToWrite, repositories)
					}
					if err != nil {
						return nil, err
					}
					repositories.MiReKyouReps.MiReKyouRepositories = append(repositories.MiReKyouReps.MiReKyouRepositories, miReKyouRep)
					if rep.UseToWrite {
						newPath, _ := miReKyouRep.GetPath(ctx, "")
						if repositories.WriteMiReKyouRep != nil {
							existPath, _ := repositories.WriteMiReKyouRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write miReKyou rep %s %s", existPath, newPath)
							return nil, err
						}
						repositories.WriteMiReKyouRep = miReKyouRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if rep.IsWatchTargetForUpdateRep {
						rep := miReKyouRep
						enableUpdateRepsCache := false
						enableUpdateLatestDataRepositoryCache := true
						cacheUpdater := rep_cache_updater.NewLatestRepositoryAddressCacheUpdater(rep, repositories, enableUpdateRepsCache, enableUpdateLatestDataRepositoryCache)
						ignoreFileNamePrefixes := []string{}
						repFilename, err := rep.GetPath(ctx, "")
						if err != nil {
							repName, _ := rep.GetRepName(ctx)
							err = fmt.Errorf("error at get path. repname = %s: %w", repName, err)
							return nil, err
						}
						repFilename = filepath.ToSlash(repFilename)

						err = g.fileRepWatchCacheUpdater.RegisterWatchFileRep(cacheUpdater, repFilename, ignoreFileNamePrefixes, userID)
						if err != nil {
							err = fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "directory":
					autoIDF := rep.IsExecuteIDFWhenReload
					parentDir := filepath.Join(filename, ".gkill")
					err := os.MkdirAll(os.ExpandEnv(parentDir), os.ModePerm)
					if err != nil {
						err = fmt.Errorf("error at make directory %s: %w", parentDir, err)
						return nil, err
					}
					hide_files.HideFolder(parentDir)
					idDBFilename := filepath.Join(parentDir, "gkill_id.db")

					var idfKyouRep reps.IDFKyouRepository
					if !rep.UseToWrite && gkill_options.CacheRepsLocalStorage {
						idfKyouRep, err = reps.NewIDFDirRepLocalCached(ctx, userID, filename, idDBFilename, rep.UseToWrite, g.router, autoIDF, &g.IDFIgnore, repositories)
					} else {
						idfKyouRep, err = reps.NewIDFDirRep(ctx, userID, filename, idDBFilename, rep.UseToWrite, g.router, autoIDF, &g.IDFIgnore, repositories)
					}
					if err != nil {
						return nil, err
					}
					repositories.IDFKyouReps = append(repositories.IDFKyouReps, idfKyouRep)
					if rep.UseToWrite {
						newPath, _ := idfKyouRep.GetPath(ctx, "")
						if repositories.WriteIDFKyouRep != nil {
							existPath, _ := repositories.WriteIDFKyouRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write idf kyou rep %s %s", existPath, newPath)
							return nil, err
						}
						repositories.WriteIDFKyouRep = idfKyouRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if rep.IsWatchTargetForUpdateRep {
						rep := idfKyouRep
						enableUpdateRepsCache := true
						enableUpdateLatestDataRepositoryCache := true
						cacheUpdater := rep_cache_updater.NewLatestRepositoryAddressCacheUpdater(rep, repositories, enableUpdateRepsCache, enableUpdateLatestDataRepositoryCache)
						ignoreFileNamePrefixes := []string{}
						repFilename := idDBFilename

						err = g.fileRepWatchCacheUpdater.RegisterWatchFileRep(cacheUpdater, repFilename, ignoreFileNamePrefixes, userID)
						if err != nil {
							err = fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "gpslog":
					if gkill_options.LoadIDFRepOnly {
						continue
					}
					err := os.MkdirAll(os.ExpandEnv(filename), os.ModePerm)
					if err != nil {
						err = fmt.Errorf("error at make directory %s: %w", filename, err)
						return nil, err
					}

					gpslogRep := reps.NewGPXDirRep(filename)
					repositories.GPSLogReps = append(repositories.GPSLogReps, gpslogRep)
					if rep.UseToWrite {
						repositories.WriteGPSLogRep = gpslogRep
					}

				case "git_commit_log":
					if gkill_options.LoadIDFRepOnly {
						continue
					}
					gitCommitLogRep, err := reps.NewGitRep(filename)
					if err != nil {
						return nil, err
					}
					repositories.GitCommitLogReps = append(repositories.GitCommitLogReps, gitCommitLogRep)
				}
			}
		}

		// キャッシュしたRep。
		// XxxRepsをキャッシュrep1個で差し替えると同時に、その実体をCachedRepsへ控える。
		// このあと(プラグイン読み込み時)に型別アダプタがXxxRepsへappendされるので、
		// 「XxxRepsの長さが1ならキャッシュrep」という個数判定は成立しない。
		// 書き込み後の反映はrepositories.WriteThroughXxxCacheを使うこと
		if *gkill_options.CacheKmemoReps {
			cachedKmemoRep, err := reps.NewKmemoRepositoryCachedSQLite3Impl(ctx, repositories.KmemoReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_KMEMO")
			if err != nil {
				err = fmt.Errorf("error at new cached kmemo rep: %w", err)
				return nil, err
			}
			repositories.KmemoReps = []reps.KmemoRepository{cachedKmemoRep}
			repositories.CachedReps.Kmemo = cachedKmemoRep
		}

		if *gkill_options.CacheURLogReps {
			cachedURLogRep, err := reps.NewURLogRepositoryCachedSQLite3Impl(ctx, repositories.URLogReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_URLOG")
			if err != nil {
				err = fmt.Errorf("error at new cached urlog rep: %w", err)
				return nil, err
			}
			repositories.URLogReps = []reps.URLogRepository{cachedURLogRep}
			repositories.CachedReps.URLog = cachedURLogRep
		}

		if *gkill_options.CacheKCReps {
			cachedKCRep, err := reps.NewKCRepositoryCachedSQLite3Impl(ctx, repositories.KCReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_KC")
			if err != nil {
				err = fmt.Errorf("error at new cached kc rep: %w", err)
				return nil, err
			}
			repositories.KCReps = []reps.KCRepository{cachedKCRep}
			repositories.CachedReps.KC = cachedKCRep
		}

		if *gkill_options.CacheIDFKyouReps {
			cachedIDFKyouRep, err := reps.NewIDFCachedRep(ctx, repositories.IDFKyouReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_IDF")
			if err != nil {
				err = fmt.Errorf("error at new cached idf kyou rep: %w", err)
				return nil, err
			}
			repositories.IDFKyouReps = []reps.IDFKyouRepository{cachedIDFKyouRep}
			repositories.CachedReps.IDFKyou = cachedIDFKyouRep
		}

		if *gkill_options.CacheLantanaReps {
			cachedLantanaRep, err := reps.NewLantanaRepositoryCachedSQLite3Impl(ctx, repositories.LantanaReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_LANTANA")
			if err != nil {
				err = fmt.Errorf("error at new cached lantana rep: %w", err)
				return nil, err
			}
			repositories.LantanaReps = []reps.LantanaRepository{cachedLantanaRep}
			repositories.CachedReps.Lantana = cachedLantanaRep
		}

		if *gkill_options.CacheTimeIsReps {
			cachedTimeIsRep, err := reps.NewTimeIsRepositoryCachedSQLite3Impl(ctx, repositories.TimeIsReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_TIMEIS")
			if err != nil {
				err = fmt.Errorf("error at new cached timeis rep: %w", err)
				return nil, err
			}
			repositories.TimeIsReps = []reps.TimeIsRepository{cachedTimeIsRep}
			repositories.CachedReps.TimeIs = cachedTimeIsRep
		}

		if *gkill_options.CacheMiReps {
			cachedMiRep, err := reps.NewMiRepositoryCachedSQLite3Impl(ctx, repositories.MiReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_MI")
			if err != nil {
				err = fmt.Errorf("error at new cached mi rep: %w", err)
				return nil, err
			}
			repositories.MiReps = []reps.MiRepository{cachedMiRep}
			repositories.CachedReps.Mi = cachedMiRep
		}

		if *gkill_options.CacheNlogReps {
			cachedNlogRep, err := reps.NewNlogRepositoryCachedSQLite3Impl(ctx, repositories.NlogReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_NLOG")
			if err != nil {
				err = fmt.Errorf("error at new cached nlog rep: %w", err)
				return nil, err
			}
			repositories.NlogReps = []reps.NlogRepository{cachedNlogRep}
			repositories.CachedReps.Nlog = cachedNlogRep
		}

		if *gkill_options.CacheTagReps {
			cachedTagRep, err := reps.NewTagRepositoryCachedSQLite3Impl(ctx, repositories.TagReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_TAG")
			if err != nil {
				err = fmt.Errorf("error at new cached tag rep: %w", err)
				return nil, err
			}
			repositories.TagReps = []reps.TagRepository{cachedTagRep}
			repositories.CachedReps.Tag = cachedTagRep
		}

		if *gkill_options.CacheTextReps {
			cachedTextRep, err := reps.NewTextRepositoryCachedSQLite3Impl(ctx, repositories.TextReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_TEXT")
			if err != nil {
				err = fmt.Errorf("error at new cached text rep: %w", err)
				return nil, err
			}
			repositories.TextReps = []reps.TextRepository{cachedTextRep}
			repositories.CachedReps.Text = cachedTextRep
		}

		if *gkill_options.CacheNotificationReps {
			cachedNotificationRep, err := reps.NewNotificationRepositoryCachedSQLite3Impl(ctx, repositories.NotificationReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_NOTIFICATION")
			if err != nil {
				err = fmt.Errorf("error at new cached notification rep: %w", err)
				return nil, err
			}
			repositories.NotificationReps = []reps.NotificationRepository{cachedNotificationRep}
			repositories.CachedReps.Notification = cachedNotificationRep
		}
		if *gkill_options.CacheReKyouReps {
			// キャッシュ差し替え前の実体を固定化して自己参照を防ぐ
			originalReKyouReps := append([]reps.ReKyouRepository(nil), repositories.ReKyouReps.ReKyouRepositories...)
			rekyouRepositories := reps.ReKyouRepositories{ReKyouRepositories: originalReKyouReps, GkillRepositories: repositories}
			cachedReKyouRep, err := reps.NewReKyouRepositoryCachedSQLite3Impl(ctx, &rekyouRepositories, repositories, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_REKYOU")
			if err != nil {
				err = fmt.Errorf("error at new cached rekyou rep: %w", err)
				return nil, err
			}
			repositories.ReKyouReps = reps.ReKyouRepositories{ReKyouRepositories: []reps.ReKyouRepository{cachedReKyouRep}, GkillRepositories: repositories}
			repositories.CachedReps.ReKyou = cachedReKyouRep
		}
		if *gkill_options.CacheMiReKyouReps {
			// キャッシュ差し替え前の実体を固定化して自己参照を防ぐ
			originalMiReKyouReps := append([]reps.MiReKyouRepository(nil), repositories.MiReKyouReps.MiReKyouRepositories...)
			mirekyouRepositories := reps.MiReKyouRepositories{MiReKyouRepositories: originalMiReKyouReps, GkillRepositories: repositories}
			cachedMiReKyouRep, err := reps.NewMiReKyouRepositoryCachedSQLite3Impl(ctx, &mirekyouRepositories, repositories, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_MIREKYOU")
			if err != nil {
				err = fmt.Errorf("error at new cached mirekyou rep: %w", err)
				return nil, err
			}
			repositories.MiReKyouReps = reps.MiReKyouRepositories{MiReKyouRepositories: []reps.MiReKyouRepository{cachedMiReKyouRep}, GkillRepositories: repositories}
			repositories.CachedReps.MiReKyou = cachedMiReKyouRep
		}
		if *gkill_options.CacheGitCommitLogReps {
			// Phase 1: GitCommitLog専用の永続ファイルベースSQLite DBを使用
			// Phase 4: 初回フルリビルドはバックグラウンドで実行
			cacheDir := os.ExpandEnv(gkill_options.CacheDir)
			gitCommitLogCacheSubDir := filepath.Join(cacheDir, "git_commit_log_cache")
			os.MkdirAll(gitCommitLogCacheSubDir, os.ModePerm)
			gitCommitLogCacheDBPath := filepath.Join(gitCommitLogCacheSubDir, userID+"_git_commit_log_cache.db")
			cachedGitCommitLogRep, err := reps.NewGitRepCachedSQLite3ImplPersistent(ctx, repositories.GitCommitLogReps, gitCommitLogCacheDBPath, userID+"_GIT_COMMIT_LOG", true)
			if err != nil {
				err = fmt.Errorf("error at new cached git commit log rep: %w", err)
				return nil, err
			}
			repositories.GitCommitLogReps = []reps.GitCommitLogRepository{cachedGitCommitLogRep}
		}

		// Repsへの追加
		for _, rep := range repositories.KmemoReps {
			repositories.Reps = append(repositories.Reps, rep)
		}
		for _, rep := range repositories.KCReps {
			repositories.Reps = append(repositories.Reps, rep)
		}
		for _, rep := range repositories.URLogReps {
			repositories.Reps = append(repositories.Reps, rep)
		}
		for _, rep := range repositories.TimeIsReps {
			repositories.Reps = append(repositories.Reps, rep)
		}
		for _, rep := range repositories.MiReps {
			repositories.Reps = append(repositories.Reps, rep)
		}
		for _, rep := range repositories.NlogReps {
			repositories.Reps = append(repositories.Reps, rep)
		}
		for _, rep := range repositories.LantanaReps {
			repositories.Reps = append(repositories.Reps, rep)
		}
		for _, rep := range repositories.IDFKyouReps {
			repositories.Reps = append(repositories.Reps, rep)
		}
		for _, rep := range repositories.GitCommitLogReps {
			repositories.Reps = append(repositories.Reps, rep)
		}
		for _, rep := range repositories.ReKyouReps.ReKyouRepositories {
			repositories.Reps = append(repositories.Reps, rep)
		}
		for _, rep := range repositories.MiReKyouReps.MiReKyouRepositories {
			repositories.Reps = append(repositories.Reps, rep)
		}

		// プラグインリポジトリを追加（デバイス非依存、ユーザ別）
		pm := g.getOrCreatePluginManager(userID)
		if discoverErr := pm.DiscoverPlugins(ctx); discoverErr != nil {
			slog.Warn(fmt.Sprintf("plugin discovery error for user %q: %q", userID, discoverErr))
		}
		for _, pluginRepo := range pm.GetRepositories() {
			pluginRep := pluginRepo.(reps.PluginRepository)
			repositories.PluginReps = append(repositories.PluginReps, pluginRep)

			// manifest.jsonのemits_kyouがfalseのプラグインはRepsに入れない。
			//
			// GPSログだけを提供するプラグインがこれにあたる。Kyouを1件も返さないのに
			// Repsに居ると「記録保管場所」の一覧（GetAllRepNames）に並び、
			// 選んでも0件の項目になる。検索のたびに空振りの往復も1回発生する。
			//
			// PluginReps（設定・死活確認）とGPSLogReps（下で登録）には入るので、
			// GPSログは rep の選択状態と無関係に常に使われる。
			if pluginRep.GetManifest().EmitsKyouOrDefault() {
				repositories.Reps = append(repositories.Reps, pluginRepo)
			}

			// manifest.jsonのprovidesに応じて型別/付随データのアダプタを登録する。
			//
			// ここが上のRepsへのコピーループより後にあることが要点。
			// 先に足すとKCReps→Repsのコピーでアダプタまで Reps に入り、
			// プラグイン本体と二重に検索されて同じ記録が2回出る。
			//
			// WriteXxxRepには絶対に入れない（プラグインは読み取り専用）。
			// TagRepsWatchTarget / TextRepsWatchTargetにも入れない
			// （ファイル実体が無くfsnotifyの監視対象にならないため）。
			adapters := reps.NewPluginTypedRepositories(pluginRep)
			if adapters.Kmemo != nil {
				repositories.KmemoReps = append(repositories.KmemoReps, adapters.Kmemo)
			}
			if adapters.KC != nil {
				repositories.KCReps = append(repositories.KCReps, adapters.KC)
			}
			if adapters.URLog != nil {
				repositories.URLogReps = append(repositories.URLogReps, adapters.URLog)
			}
			if adapters.Nlog != nil {
				repositories.NlogReps = append(repositories.NlogReps, adapters.Nlog)
			}
			if adapters.Lantana != nil {
				repositories.LantanaReps = append(repositories.LantanaReps, adapters.Lantana)
			}
			if adapters.TimeIs != nil {
				repositories.TimeIsReps = append(repositories.TimeIsReps, adapters.TimeIs)
			}
			if adapters.Mi != nil {
				repositories.MiReps = append(repositories.MiReps, adapters.Mi)
			}
			if adapters.Tag != nil {
				repositories.TagReps = append(repositories.TagReps, adapters.Tag)
			}
			if adapters.Text != nil {
				repositories.TextReps = append(repositories.TextReps, adapters.Text)
			}
			if adapters.Notification != nil {
				repositories.NotificationReps = append(repositories.NotificationReps, adapters.Notification)
			}

			// GPSログを提供するプラグインは GPSLogRepository としても登録する。
			// 書き込み口を持たないので WriteGPSLogRep には決してしない。
			if gpsLogRep, ok := reps.NewGPSLogPluginRepIfProvided(pluginRep); ok {
				repositories.GPSLogReps = append(repositories.GPSLogReps, gpsLogRep)
			}
		}

		err = repositories.UpdateCache(ctx)
		if err != nil {
			err = fmt.Errorf("error at update cache in get repositories: %w", err)
			return nil, err
		}
		g.storeRepositories(userID, device, repositories)

		if _, err := g.GetNotificator(userID, device); err != nil {
			slog.Log(context.Background(), gkill_log.Warn, "error at get notificator", "error", fmt.Sprintf("%q", err), "userID", fmt.Sprintf("%q", userID), "device", fmt.Sprintf("%q", device))
		}
	}

	return repositories, nil
}

// getOrCreatePluginManager はユーザID別のPluginManagerを取得または作成する。
func (g *GkillDAOManager) getOrCreatePluginManager(userID string) *PluginManager {
	g.stateMutex.Lock()
	defer g.stateMutex.Unlock()

	if g.pluginManagers == nil {
		g.pluginManagers = map[string]*PluginManager{}
	}
	if pm, ok := g.pluginManagers[userID]; ok {
		return pm
	}
	pm := newPluginManager(userID)
	g.pluginManagers[userID] = pm
	return pm
}

// GetPluginManager はユーザID別のPluginManagerを返す。APIハンドラからプラグイン操作時に使用する。
// まだ走査していなければ走査してから返す。
//
// かつてはリクエストのたびに DiscoverPlugins を呼んでいたが、
// プラグインKyouの本文取得は一覧の行数ぶん飛んでくるため、
// ディレクトリ列挙と manifest.json の読み込みがその回数ぶん繰り返されていた。
// リポジトリ構築時(GetRepositories)には引き続き再走査するので、
// プラグインを置き直したあとも再ログインやリロードで拾える。
func (g *GkillDAOManager) GetPluginManager(userID string) *PluginManager {
	pm := g.getOrCreatePluginManager(userID)
	if discoverErr := pm.EnsureDiscovered(context.Background()); discoverErr != nil {
		slog.Warn(fmt.Sprintf("plugin discovery error for user %q: %q", userID, discoverErr))
	}
	return pm
}

func (g *GkillDAOManager) GetNotificator(userID string, device string) (*GkillNotificator, error) {
	// すでに存在していればそれを返す。マップ操作中だけ stateMutex を保持する
	if notificator, exist := g.lookupNotificator(userID, device); exist {
		return notificator, nil
	}

	// Notificatorの初期化
	gkillRepositories, err := g.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories in get notificator: %w", err)
		return nil, err
	}

	gkillNotificator, err := NewGkillNotificator(context.Background(), g, gkillRepositories)
	if err != nil {
		err = fmt.Errorf("error at new gkill notificator: %w", err)
		return nil, err
	}

	g.stateMutex.Lock()
	defer g.stateMutex.Unlock()
	if g.gkillNotificators == nil {
		g.gkillNotificators = map[string]map[string]*GkillNotificator{}
	}
	if _, exist := g.gkillNotificators[userID]; !exist {
		g.gkillNotificators[userID] = map[string]*GkillNotificator{}
	}
	// 並行して先に作られていたらそちらを使い、自分が作ったほうは閉じる
	if existing, exist := g.gkillNotificators[userID][device]; exist {
		if closeErr := gkillNotificator.Close(context.Background()); closeErr != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at close duplicated notificator", "error", closeErr)
		}
		return existing, nil
	}
	g.gkillNotificators[userID][device] = gkillNotificator
	return gkillNotificator, nil
}

// lookupNotificator は構築済みNotificatorがあれば返す。
func (g *GkillDAOManager) lookupNotificator(userID string, device string) (*GkillNotificator, bool) {
	g.stateMutex.Lock()
	defer g.stateMutex.Unlock()

	if g.gkillNotificators == nil {
		return nil, false
	}
	notificatorInUser, exist := g.gkillNotificators[userID]
	if !exist {
		return nil, false
	}
	notificator, exist := notificatorInUser[device]
	return notificator, exist
}

func (g *GkillDAOManager) Close() error {
	ctx := context.Background()
	var allErrors error
	var err error

	if e := g.fileRepWatchCacheUpdater.Close(); e != nil {
		err = fmt.Errorf("error at close file rep watch cache updater. : %w : %w", e, err)
	}

	// マップをいったん切り離してからクローズする。
	// クローズには時間がかかるので stateMutex は保持しない
	g.stateMutex.Lock()
	gkillRepositories := g.gkillRepositories
	gkillNotificators := g.gkillNotificators
	pluginManagers := g.pluginManagers
	g.gkillRepositories = nil
	g.gkillNotificators = nil
	g.pluginManagers = nil
	g.initializingMutex = nil
	g.stateMutex.Unlock()

	for userID, repInDevices := range gkillRepositories {
		for repName, repInDevice := range repInDevices {
			err = repInDevice.Close(ctx)
			if err != nil {
				if allErrors != nil {
					allErrors = fmt.Errorf("error at close repository user id = %s rep name %s: %w", userID, repName, err)
				} else {
					allErrors = fmt.Errorf("error at close repository user id = %s rep name %s", userID, repName)
				}
			}

		}
	}

	for userID, notificatorInDevices := range gkillNotificators {
		for _, notificator := range notificatorInDevices {
			err = notificator.Close(ctx)
			if err != nil {
				if allErrors != nil {
					allErrors = fmt.Errorf("error at close gkill notificator user id = %s: %w", userID, err)
				} else {
					allErrors = fmt.Errorf("error at close gkill notificator user id = %s", userID)
				}
			}
		}
	}

	for userID, pm := range pluginManagers {
		if e := pm.CloseAll(ctx); e != nil {
			if allErrors != nil {
				allErrors = fmt.Errorf("error at close plugins user id = %s: %w: %w", userID, e, allErrors)
			} else {
				allErrors = fmt.Errorf("error at close plugins user id = %s: %w", userID, e)
			}
		}
	}

	if g.ConfigDAOs != nil {
		err = g.ConfigDAOs.AccountDAO.Close(ctx)
		if err != nil {
			return err
		}
		err = g.ConfigDAOs.LoginSessionDAO.Close(ctx)
		if err != nil {
			return err
		}
		err = g.ConfigDAOs.FileUploadHistoryDAO.Close(ctx)
		if err != nil {
			return err
		}
		err = g.ConfigDAOs.ShareKyouInfoDAO.Close(ctx)
		if err != nil {
			return err
		}
		err = g.ConfigDAOs.ServerConfigDAO.Close(ctx)
		if err != nil {
			return err
		}
		err = g.ConfigDAOs.ApplicationConfigDAO.Close(ctx)
		if err != nil {
			return err
		}
		err = g.ConfigDAOs.RepositoryDAO.Close(ctx)
		if err != nil {
			return err
		}
		g.ConfigDAOs.AccountDAO = nil
		g.ConfigDAOs.LoginSessionDAO = nil
		g.ConfigDAOs.FileUploadHistoryDAO = nil
		g.ConfigDAOs.ShareKyouInfoDAO = nil
		g.ConfigDAOs.ServerConfigDAO = nil
		g.ConfigDAOs.ApplicationConfigDAO = nil
		g.ConfigDAOs.RepositoryDAO = nil
	}

	g.ConfigDAOs = nil
	g.router = nil
	g.IDFIgnore = []string{}

	return err
}

func (g *GkillDAOManager) SetSkipIDF(skip bool) {
	*g.skipUpdateCache = skip
}

func (g *GkillDAOManager) CloseUserRepositories(userID string, device string) (bool, error) {
	var err error
	ctx := context.TODO()

	g.stateMutex.Lock()
	repsInDevices, exist := g.gkillRepositories[userID]
	var reps *reps.GkillRepositories
	if exist {
		reps, exist = repsInDevices[device]
	}
	if exist {
		// 以降のクローズ処理中に他のリクエストが同じRepを掴まないよう、先にマップから外す
		delete(g.gkillRepositories[userID], device)
		if len(g.gkillRepositories[userID]) == 0 {
			delete(g.gkillRepositories, userID)
		}
	}
	g.stateMutex.Unlock()

	if !exist {
		return false, nil
	}

	// Reps, TagReps, TextReps, GPSLogRepsの監視をやめる
	removeWatchTargetReps := []rep_cache_updater.CacheUpdatable{}
	for _, reps := range reps.Reps {
		unwrapedReps, err := reps.UnWrap()
		if err != nil {
			return false, err
		}
		for _, unwrapedRep := range unwrapedReps {
			removeWatchTargetReps = append(removeWatchTargetReps, unwrapedRep)
		}
	}
	for _, tagReps := range reps.TagRepsWatchTarget {
		unwrapedTagReps, err := tagReps.UnWrapTyped()
		if err != nil {
			return false, err
		}
		for _, unwrapedTagRep := range unwrapedTagReps {
			removeWatchTargetReps = append(removeWatchTargetReps, unwrapedTagRep)
		}
	}
	for _, textReps := range reps.TextRepsWatchTarget {
		unwrapedTextReps, err := textReps.UnWrapTyped()
		if err != nil {
			return false, err
		}
		for _, unwrapedTextRep := range unwrapedTextReps {
			removeWatchTargetReps = append(removeWatchTargetReps, unwrapedTextRep)
		}
	}
	for _, notificationReps := range reps.NotificationReps {
		unwrapedNotificationReps, err := notificationReps.UnWrapTyped()
		if err != nil {
			return false, err
		}
		for _, unwrapedNotificationRep := range unwrapedNotificationReps {
			removeWatchTargetReps = append(removeWatchTargetReps, unwrapedNotificationRep)
		}
	}
	for _, gpsLogReps := range reps.GPSLogReps {
		unwrapedGPSLogReps, err := gpsLogReps.UnWrapTyped()
		if err != nil {
			return false, err
		}
		for _, unwrapedGPSLogRep := range unwrapedGPSLogReps {
			removeWatchTargetReps = append(removeWatchTargetReps, unwrapedGPSLogRep)
		}
	}

	for _, rep := range removeWatchTargetReps {
		filename, err := rep.GetPath(ctx, "")
		if err != nil {
			repName, _ := rep.GetRepName(ctx)
			err = fmt.Errorf("error at get path. repname = %s: %w", repName, err)
			return false, err
		}
		filename = filepath.ToSlash(filename)

		err = g.fileRepWatchCacheUpdater.RemoveWatchFileRep(filename, userID)
		if err != nil {
			err = fmt.Errorf("error at remove watch file rep. filename = %s userID = %s: %w", filename, userID, err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		}
	}

	// 全Repを閉じる
	err = reps.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close repositories: %w", err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
	}
	return true, nil
}
