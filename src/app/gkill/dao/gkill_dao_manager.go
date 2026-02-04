package dao

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/gorilla/mux"
	"github.com/mattn/go-zglob"
	"github.com/mt3hr/gkill/src/app/gkill/dao/account"
	"github.com/mt3hr/gkill/src/app/gkill/dao/account_state"
	"github.com/mt3hr/gkill/src/app/gkill/dao/gkill_notification"

	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps/rep_cache_updater"
	"github.com/mt3hr/gkill/src/app/gkill/dao/server_config"
	"github.com/mt3hr/gkill/src/app/gkill/dao/share_kyou_info"
	"github.com/mt3hr/gkill/src/app/gkill/dao/user_config"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

type GkillDAOManager struct {
	initializingMutex        map[string]map[string]*sync.Mutex
	gkillRepositories        map[string]map[string]*reps.GkillRepositories
	gkillNotificators        map[string]map[string]*GkillNotificator
	fileRepWatchCacheUpdater rep_cache_updater.FileRepCacheUpdater

	ConfigDAOs *ConfigDAOs

	router    *mux.Router
	IDFIgnore []string

	enableOutputLogs bool
	infoLogFile      *os.File
	errorLogFile     *os.File
	debugLogFile     *os.File
	traceLogFile     *os.File
	traceSQLLogFile  *os.File

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
	gkillDAOManager.ConfigDAOs.AppllicationConfigDAO, err = user_config.NewApplicationConfigDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "user_config.db"))
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

	// ログ出力先設定
	gkillDAOManager.enableOutputLogs = gkill_options.IsOutputLog
	if gkillDAOManager.enableOutputLogs {
		logRootDir := os.ExpandEnv(gkill_options.LogDir)
		err := os.MkdirAll(logRootDir, os.ModePerm)
		if err != nil {
			err = fmt.Errorf("error at mkdir %s: %w", logRootDir, err)
			return nil, err
		}
		infoLogFileName := filepath.Join(logRootDir, "gkill_info.log")
		infoLogFile, err := os.OpenFile(infoLogFileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.ModePerm)
		if err != nil {
			err = fmt.Errorf("error at create info log file %s: %w", infoLogFile.Name(), err)
			return nil, err
		}
		gkillDAOManager.infoLogFile = infoLogFile
		gkill_log.Info.SetOutput(infoLogFile)

		errorLogFileName := filepath.Join(logRootDir, "gkill_error.log")
		errorLogFile, err := os.OpenFile(errorLogFileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.ModePerm)
		if err != nil {
			err = fmt.Errorf("error at create error log file %s: %w", errorLogFile.Name(), err)
			return nil, err
		}
		gkillDAOManager.errorLogFile = errorLogFile
		gkill_log.Error.SetOutput(errorLogFile)

		debugLogFileName := filepath.Join(logRootDir, "gkill_debug.log")
		debugLogFile, err := os.OpenFile(debugLogFileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.ModePerm)
		if err != nil {
			err = fmt.Errorf("error at create debug log file %s: %w", debugLogFile.Name(), err)
			return nil, err
		}
		gkillDAOManager.debugLogFile = debugLogFile
		gkill_log.Debug.SetOutput(debugLogFile)

		traceLogFileName := filepath.Join(logRootDir, "gkill_trage.log")
		traceLogFile, err := os.OpenFile(traceLogFileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.ModePerm)
		if err != nil {
			err = fmt.Errorf("error at create trage log file %s: %w", traceLogFile.Name(), err)
			return nil, err
		}
		gkillDAOManager.traceLogFile = traceLogFile
		gkill_log.Trace.SetOutput(traceLogFile)

		traceSQLLogFileName := filepath.Join(logRootDir, "gkill_traceSQL.log")
		traceSQLLogFile, err := os.OpenFile(traceSQLLogFileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.ModePerm)
		if err != nil {
			err = fmt.Errorf("error at create traceSQL log file %s: %w", traceSQLLogFile.Name(), err)
			return nil, err
		}
		gkillDAOManager.traceSQLLogFile = traceSQLLogFile
		gkill_log.TraceSQL.SetOutput(traceSQLLogFile)
	} else {
		gkill_log.Info = nil
		gkill_log.Error = nil
		gkill_log.Debug = nil
		gkill_log.Trace = nil
		gkill_log.TraceSQL = nil
	}

	return gkillDAOManager, nil
}

func (g *GkillDAOManager) GetRouter() *mux.Router {
	return g.router
}

func (g *GkillDAOManager) GetRepositories(userID string, device string) (*reps.GkillRepositories, error) {
	if userID == "" || device == "" {
		err := fmt.Errorf("userID or device is blank. userID=%s device=%s", userID, device)
		return nil, err
	}

	ctx := context.Background()
	var err error

	// nilだったら初期化する
	if g.initializingMutex == nil {
		g.initializingMutex = map[string]map[string]*sync.Mutex{}
	}
	if g.gkillRepositories == nil {
		g.gkillRepositories = map[string]map[string]*reps.GkillRepositories{}
	}

	// 初期化中だったらちょっとまつ
	initializeMutexInUser, existRepsInUsers := g.initializingMutex[userID]
	if !existRepsInUsers {
		g.initializingMutex[userID] = map[string]*sync.Mutex{}
		initializeMutexInUser = g.initializingMutex[userID]
	}
	_, existMutexsInDevice := initializeMutexInUser[device]
	if !existMutexsInDevice {
		g.initializingMutex[userID][device] = &sync.Mutex{}
	}

	// すでに存在していればそれを、存在していなければ作っていれる。Rep
	repositoriesInUser, existRepsInUsers := g.gkillRepositories[userID]
	if !existRepsInUsers {
		g.gkillRepositories[userID] = map[string]*reps.GkillRepositories{}
		repositoriesInUser = g.gkillRepositories[userID]
	}

	repositories, existRepsInDevice := repositoriesInUser[device]
	if !existRepsInDevice {
		// 初期化中だったら終わるまで待つ
		g.initializingMutex[userID][device].Lock()
		defer g.initializingMutex[userID][device].Unlock()

		// 初期化がおわり、値が入っていればそれを使う
		if repositories, exist := g.gkillRepositories[userID][device]; exist {
			return repositories, nil
		}

		// なかったら作っていれる
		repositories, err = reps.NewGkillRepositories(userID)
		if err != nil {
			err = fmt.Errorf("error at new gkill repositories. user id = %s: %w", userID, err)
			return nil, err
		}
		repositories.ReKyouReps.GkillRepositories = repositories

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

				parentDir := filepath.Dir(filename)
				err := os.MkdirAll(os.ExpandEnv(parentDir), os.ModePerm)
				if err != nil {
					err = fmt.Errorf("error at make directory %s: %w", parentDir, err)
					return nil, err
				}

				switch rep.Type {
				case "kmemo":
					kmemoRep, err := reps.NewKmemoRepositorySQLite3Impl(ctx, filename, rep.UseToWrite)
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
					kcRep, err := reps.NewKCRepositorySQLite3Impl(ctx, filename, rep.UseToWrite)
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
					urlogRep, err := reps.NewURLogRepositorySQLite3Impl(ctx, filename, rep.UseToWrite)
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
					timeisRep, err := reps.NewTimeIsRepositorySQLite3Impl(ctx, filename, rep.UseToWrite)
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
					miRep, err := reps.NewMiRepositorySQLite3Impl(ctx, filename, rep.UseToWrite)
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
					nlogRep, err := reps.NewNlogRepositorySQLite3Impl(ctx, filename, rep.UseToWrite)
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
					lantanaRep, err := reps.NewLantanaRepositorySQLite3Impl(ctx, filename, rep.UseToWrite)
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
					tagRep, err := reps.NewTagRepositorySQLite3Impl(ctx, filename, rep.UseToWrite)
					if err != nil {
						return nil, err
					}
					repositories.TagReps = append(repositories.TagReps, tagRep)
					repositories.TagRepsWatchTarget = append(repositories.TagReps, tagRep)
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
					textRep, err := reps.NewTextRepositorySQLite3Impl(ctx, filename, rep.UseToWrite)
					if err != nil {
						return nil, err
					}
					repositories.TextReps = append(repositories.TextReps, textRep)
					repositories.TextRepsWatchTarget = append(repositories.TextReps, textRep)
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
					notificationRep, err := reps.NewNotificationRepositorySQLite3Impl(ctx, filename, rep.UseToWrite)
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
					reKyouRep, err := reps.NewReKyouRepositorySQLite3Impl(ctx, filename, rep.UseToWrite, repositories)
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

				case "directory":
					autoIDF := rep.IsExecuteIDFWhenReload
					parentDir := filepath.Join(filename, ".gkill")
					err := os.MkdirAll(os.ExpandEnv(parentDir), os.ModePerm)
					if err != nil {
						err = fmt.Errorf("error at make directory %s: %w", parentDir, err)
						return nil, err
					}

					idDBFilename := filepath.Join(parentDir, "gkill_id.db")
					idfKyouRep, err := reps.NewIDFDirRep(ctx, filename, idDBFilename, rep.UseToWrite, g.router, autoIDF, &g.IDFIgnore, repositories)
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
					gitCommitLogRep, err := reps.NewGitRep(filename)
					if err != nil {
						return nil, err
					}
					repositories.GitCommitLogReps = append(repositories.GitCommitLogReps, gitCommitLogRep)
				}
			}
		}

		// キャッシュしたRep
		if *gkill_options.CacheKmemoReps {
			cachedKmemoRep, err := reps.NewKmemoRepositoryCachedSQLite3Impl(ctx, repositories.KmemoReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_KMEMO")
			if err != nil {
				err = fmt.Errorf("error at new cached tag rep: %w", err)
				return nil, err
			}
			repositories.KmemoReps = []reps.KmemoRepository{cachedKmemoRep}
		}

		if *gkill_options.CacheURLogReps {
			cachedURLogRep, err := reps.NewURLogRepositoryCachedSQLite3Impl(ctx, repositories.URLogReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_URLOG")
			if err != nil {
				err = fmt.Errorf("error at new cached tag rep: %w", err)
				return nil, err
			}
			repositories.URLogReps = []reps.URLogRepository{cachedURLogRep}
		}

		if *gkill_options.CacheKCReps {
			cachedKCRep, err := reps.NewKCRepositoryCachedSQLite3Impl(ctx, repositories.KCReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_KC")
			if err != nil {
				err = fmt.Errorf("error at new cached tag rep: %w", err)
				return nil, err
			}
			repositories.KCReps = []reps.KCRepository{cachedKCRep}
		}

		if *gkill_options.CacheIDFKyouReps {
			cachedIDFKyouRep, err := reps.NewIDFCachedRep(ctx, repositories.IDFKyouReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_IDF")
			if err != nil {
				err = fmt.Errorf("error at new cached tag rep: %w", err)
				return nil, err
			}
			repositories.IDFKyouReps = []reps.IDFKyouRepository{cachedIDFKyouRep}
		}

		if *gkill_options.CacheLantanaReps {
			cachedLantanaRep, err := reps.NewLantanaRepositoryCachedSQLite3Impl(ctx, repositories.LantanaReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_LANTANA")
			if err != nil {
				err = fmt.Errorf("error at new cached tag rep: %w", err)
				return nil, err
			}
			repositories.LantanaReps = []reps.LantanaRepository{cachedLantanaRep}
		}

		if *gkill_options.CacheTimeIsReps {
			cachedTimeIsRep, err := reps.NewTimeIsRepositoryCachedSQLite3Impl(ctx, repositories.TimeIsReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_TIMEIS")
			if err != nil {
				err = fmt.Errorf("error at new cached tag rep: %w", err)
				return nil, err
			}
			repositories.TimeIsReps = []reps.TimeIsRepository{cachedTimeIsRep}
		}

		if *gkill_options.CacheMiReps {
			cachedMiRep, err := reps.NewMiRepositoryCachedSQLite3Impl(ctx, repositories.MiReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_MI")
			if err != nil {
				err = fmt.Errorf("error at new cached tag rep: %w", err)
				return nil, err
			}
			repositories.MiReps = []reps.MiRepository{cachedMiRep}
		}

		if *gkill_options.CacheNlogReps {
			cachedNlogRep, err := reps.NewNlogRepositoryCachedSQLite3Impl(ctx, repositories.NlogReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_NLOG")
			if err != nil {
				err = fmt.Errorf("error at new cached tag rep: %w", err)
				return nil, err
			}
			repositories.NlogReps = []reps.NlogRepository{cachedNlogRep}
		}

		if *gkill_options.CacheTagReps {
			cachedTagRep, err := reps.NewTagRepositoryCachedSQLite3Impl(ctx, repositories.TagReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_TAG")
			if err != nil {
				err = fmt.Errorf("error at new cached tag rep: %w", err)
				return nil, err
			}
			repositories.TagReps = []reps.TagRepository{cachedTagRep}
		}

		if *gkill_options.CacheTextReps {
			cachedTextRep, err := reps.NewTextRepositoryCachedSQLite3Impl(ctx, repositories.TextReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_TEXT")
			if err != nil {
				err = fmt.Errorf("error at new cached tag rep: %w", err)
				return nil, err
			}
			repositories.TextReps = []reps.TextRepository{cachedTextRep}
		}

		if *gkill_options.CacheNotificationReps {
			cachedNotificationRep, err := reps.NewNotificationRepositoryCachedSQLite3Impl(ctx, repositories.NotificationReps, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_NOTIFICATION")
			if err != nil {
				err = fmt.Errorf("error at new cached tag rep: %w", err)
				return nil, err
			}
			repositories.NotificationReps = []reps.NotificationRepository{cachedNotificationRep}
		}
		if *gkill_options.CacheReKyouReps {
			rekyouRepositories := reps.ReKyouRepositories{ReKyouRepositories: []reps.ReKyouRepository{&repositories.ReKyouReps}, GkillRepositories: repositories}
			cachedReKyouRep, err := reps.NewReKyouRepositoryCachedSQLite3Impl(ctx, &rekyouRepositories, repositories, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, userID+"_REKYOU")
			if err != nil {
				err = fmt.Errorf("error at new cached tag rep: %w", err)
				return nil, err
			}
			repositories.ReKyouReps = reps.ReKyouRepositories{ReKyouRepositories: []reps.ReKyouRepository{cachedReKyouRep}, GkillRepositories: repositories}
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

		err = repositories.UpdateCache(ctx)
		if err != nil {
			err = fmt.Errorf("error at update cache in get repositories: %w", err)
			return nil, err
		}
		g.gkillRepositories[userID][device] = repositories
		repositories = repositoriesInUser[device]

		_, _ = g.GetNotificator(userID, device)
	}

	return repositories, nil
}

func (g *GkillDAOManager) GetNotificator(userID string, device string) (*GkillNotificator, error) {
	// nilだったら初期化する
	if g.gkillNotificators == nil {
		g.gkillNotificators = map[string]map[string]*GkillNotificator{}
	}

	// すでに存在していればそれを、存在していなければ作る。Notificator
	notificatorInUser, existNotificatorsInUsers := g.gkillNotificators[userID]
	if !existNotificatorsInUsers {
		g.gkillNotificators[userID] = map[string]*GkillNotificator{}
		notificatorInUser = g.gkillNotificators[userID]
	}

	notificator, existNotificatorsInDevice := notificatorInUser[device]
	if !existNotificatorsInDevice {
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
		g.gkillNotificators[userID][device] = gkillNotificator
		notificator = g.gkillNotificators[userID][device]
	}
	return notificator, nil
}

func (g *GkillDAOManager) Close() error {
	ctx := context.Background()
	var allErrors error
	var err error

	if e := g.fileRepWatchCacheUpdater.Close(); e != nil {
		err = fmt.Errorf("error at close file rep watch cache updater. : %w : %w", e, err)
	}

	for userID, repInDevices := range g.gkillRepositories {
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
	g.gkillRepositories = nil

	for userID, notificatorInDevices := range g.gkillNotificators {
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
	g.gkillNotificators = nil

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
		err = g.ConfigDAOs.AppllicationConfigDAO.Close(ctx)
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
		g.ConfigDAOs.AppllicationConfigDAO = nil
		g.ConfigDAOs.RepositoryDAO = nil
	}

	g.ConfigDAOs = nil
	g.router = nil
	g.IDFIgnore = []string{}

	if g.enableOutputLogs {
		if e := g.infoLogFile.Close(); e != nil {
			err = fmt.Errorf("error at close info log file %s: %w: %w", g.infoLogFile.Name(), e, err)
		}

		if e := g.errorLogFile.Close(); e != nil {
			err = fmt.Errorf("error at close error log file %s: %w: %w", g.errorLogFile.Name(), e, err)
		}

		if e := g.debugLogFile.Close(); e != nil {
			err = fmt.Errorf("error at close debug log file %s: %w : %w", g.debugLogFile.Name(), e, err)
		}

		if e := g.traceLogFile.Close(); e != nil {
			err = fmt.Errorf("error at close trace log file %s: %w : %w", g.traceLogFile.Name(), e, err)
		}

		if e := g.traceSQLLogFile.Close(); e != nil {
			err = fmt.Errorf("error at close trace sql log file %s: %w : %w", g.traceSQLLogFile.Name(), e, err)
		}
	}

	return err
}

func (g *GkillDAOManager) SetSkipIDF(skip bool) {
	*g.skipUpdateCache = skip
}

func (g *GkillDAOManager) CloseUserRepositories(userID string, device string) (bool, error) {
	var err error
	ctx := context.TODO()

	repsInDevices, exist := g.gkillRepositories[userID]
	if !exist {
		return false, nil
	}

	reps, exist := repsInDevices[device]
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
			gkill_log.Debug.Println(err.Error())
		}
	}

	// 全Repを閉じる
	err = reps.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close repositories: %w", err)
		gkill_log.Debug.Println(err.Error())
	}
	delete(g.gkillRepositories[userID], device)
	delete(g.gkillRepositories, userID)
	return true, nil
}
