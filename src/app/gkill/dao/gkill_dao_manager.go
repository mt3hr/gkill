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
	"github.com/mt3hr/gkill/src/app/gkill/dao/mi_share_info"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps/rep_cache_updater"
	"github.com/mt3hr/gkill/src/app/gkill/dao/server_config"
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
	autoIDF   *bool
	IDFIgnore []string

	enableOutputLogs bool
	infoLogFile      *os.File
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

	autoIDF := true

	ctx := context.Background()
	gkillDAOManager := &GkillDAOManager{
		autoIDF:                  &autoIDF,
		router:                   &mux.Router{},
		IDFIgnore:                gkill_options.IDFIgnore,
		fileRepWatchCacheUpdater: fileRepWatchCacheUpdater,
		skipUpdateCache:          &skipUpdateCache,
	}

	configDBRootDir := os.ExpandEnv(gkill_options.ConfigDir)
	err = os.MkdirAll(os.ExpandEnv(configDBRootDir), fs.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at create directory %s: %w", err)
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
	gkillDAOManager.ConfigDAOs.MiShareInfoDAO, err = mi_share_info.NewMiShareInfoDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "mi_share_info.db"))
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
	gkillDAOManager.ConfigDAOs.KFTLTemplateDAO, err = user_config.NewKFTLTemplateDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "user_config.db"))
	if err != nil {
		return nil, err
	}
	gkillDAOManager.ConfigDAOs.TagStructDAO, err = user_config.NewTagStructDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "user_config.db"))
	if err != nil {
		return nil, err
	}
	gkillDAOManager.ConfigDAOs.RepStructDAO, err = user_config.NewRepStructDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "user_config.db"))
	if err != nil {
		return nil, err
	}
	gkillDAOManager.ConfigDAOs.DeviceStructDAO, err = user_config.NewDeviceStructDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "user_config.db"))
	if err != nil {
		return nil, err
	}
	gkillDAOManager.ConfigDAOs.RepTypeStructDAO, err = user_config.NewRepTypeStructDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "user_config.db"))
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
		}
		infoLogFileName := filepath.Join(logRootDir, "gkill_info.log")
		infoLogFile, err := os.OpenFile(infoLogFileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.ModePerm)
		if err != nil {
			err = fmt.Errorf("error at create info log file %s: %w", infoLogFile, err)
			return nil, err
		}
		gkillDAOManager.infoLogFile = infoLogFile
		gkill_log.Info.SetOutput(infoLogFile)

		debugLogFileName := filepath.Join(logRootDir, "gkill_debug.log")
		debugLogFile, err := os.OpenFile(debugLogFileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.ModePerm)
		if err != nil {
			err = fmt.Errorf("error at create debug log file %s: %w", debugLogFile, err)
			return nil, err
		}
		gkillDAOManager.debugLogFile = debugLogFile
		gkill_log.Debug.SetOutput(debugLogFile)

		trageLogFileName := filepath.Join(logRootDir, "gkill_trage.log")
		traceLogFile, err := os.OpenFile(trageLogFileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.ModePerm)
		if err != nil {
			err = fmt.Errorf("error at create trage log file %s: %w", traceLogFile, err)
			return nil, err
		}
		gkillDAOManager.traceLogFile = traceLogFile
		gkill_log.TraceSQL.SetOutput(traceLogFile)

		traceSQLLogFileName := filepath.Join(logRootDir, "gkill_traceSQL.log")
		traceSQLLogFile, err := os.OpenFile(traceSQLLogFileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.ModePerm)
		if err != nil {
			err = fmt.Errorf("error at create traceSQL log file %s: %w", traceSQLLogFile, err)
			return nil, err
		}
		gkillDAOManager.traceSQLLogFile = traceSQLLogFile
		gkill_log.TraceSQL.SetOutput(traceSQLLogFile)
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

	ctx := context.TODO()
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
		for _, rep := range repositoriesDefine {
			if !rep.IsEnable {
				continue
			}
			rep.File = os.ExpandEnv(rep.File)
			matchFiles, _ := zglob.Glob(rep.File)
			sort.Strings(matchFiles)
			for _, filename := range matchFiles {
				parentDir := filepath.Dir(filename)
				err := os.MkdirAll(os.ExpandEnv(parentDir), os.ModePerm)
				if err != nil {
					err = fmt.Errorf("error at make directory %s: %w", parentDir)
					return nil, err
				}

				switch rep.Type {
				case "kmemo":
					kmemoRep, err := reps.NewKmemoRepositorySQLite3Impl(ctx, filename)
					if err != nil {
						return nil, err
					}
					repositories.KmemoReps = append(repositories.KmemoReps, kmemoRep)
					repositories.Reps = append(repositories.Reps, kmemoRep)
					if rep.UseToWrite {
						newPath, _ := kmemoRep.GetPath(ctx, "")
						if repositories.WriteKmemoRep != nil {
							existPath, _ := repositories.WriteKmemoRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write kmemo rep %s %s: %w", existPath, newPath)
							return nil, err
						}
						repositories.WriteKmemoRep = kmemoRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if *g.autoIDF {
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
							fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "urlog":
					urlogRep, err := reps.NewURLogRepositorySQLite3Impl(ctx, filename)
					if err != nil {
						return nil, err
					}
					repositories.URLogReps = append(repositories.URLogReps, urlogRep)
					repositories.Reps = append(repositories.Reps, urlogRep)
					if rep.UseToWrite {
						newPath, _ := urlogRep.GetPath(ctx, "")
						if repositories.WriteURLogRep != nil {
							existPath, _ := repositories.WriteURLogRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write urlog rep %s %s: %w", existPath, newPath)
							return nil, err
						}
						repositories.WriteURLogRep = urlogRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if *g.autoIDF {
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
							fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "timeis":
					timeisRep, err := reps.NewTimeIsRepositorySQLite3Impl(ctx, filename)
					if err != nil {
						return nil, err
					}
					repositories.TimeIsReps = append(repositories.TimeIsReps, timeisRep)
					repositories.Reps = append(repositories.Reps, timeisRep)
					if rep.UseToWrite {
						newPath, _ := timeisRep.GetPath(ctx, "")
						if repositories.WriteTimeIsRep != nil {
							existPath, _ := repositories.WriteTimeIsRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write timeis rep %s %s: %w", existPath, newPath)
							return nil, err
						}
						repositories.WriteTimeIsRep = timeisRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if *g.autoIDF {
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
							fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "mi":
					miRep, err := reps.NewMiRepositorySQLite3Impl(ctx, filename)
					if err != nil {
						return nil, err
					}
					repositories.MiReps = append(repositories.MiReps, miRep)
					repositories.Reps = append(repositories.Reps, miRep)
					if rep.UseToWrite {
						newPath, _ := miRep.GetPath(ctx, "")
						if repositories.WriteMiRep != nil {
							existPath, _ := repositories.WriteMiRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write mi rep %s %s: %w", existPath, newPath)
							return nil, err
						}
						repositories.WriteMiRep = miRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if *g.autoIDF {
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
							fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "nlog":
					nlogRep, err := reps.NewNlogRepositorySQLite3Impl(ctx, filename)
					if err != nil {
						return nil, err
					}
					repositories.NlogReps = append(repositories.NlogReps, nlogRep)
					repositories.Reps = append(repositories.Reps, nlogRep)
					if rep.UseToWrite {
						newPath, _ := nlogRep.GetPath(ctx, "")
						if repositories.WriteNlogRep != nil {
							existPath, _ := repositories.WriteNlogRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write nlog rep %s %s: %w", existPath, newPath)
							return nil, err
						}
						repositories.WriteNlogRep = nlogRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if *g.autoIDF {
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
							fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "lantana":
					lantanaRep, err := reps.NewLantanaRepositorySQLite3Impl(ctx, filename)
					if err != nil {
						return nil, err
					}
					repositories.LantanaReps = append(repositories.LantanaReps, lantanaRep)
					repositories.Reps = append(repositories.Reps, lantanaRep)
					if rep.UseToWrite {
						newPath, _ := lantanaRep.GetPath(ctx, "")
						if repositories.WriteLantanaRep != nil {
							existPath, _ := repositories.WriteLantanaRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write lantana rep %s %s: %w", existPath, newPath)
							return nil, err
						}
						repositories.WriteLantanaRep = lantanaRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if *g.autoIDF {
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
							fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "tag":
					tagRep, err := reps.NewTagRepositorySQLite3Impl(ctx, filename)
					if err != nil {
						return nil, err
					}
					repositories.TagReps = append(repositories.TagReps, tagRep)
					if rep.UseToWrite {
						newPath, _ := tagRep.GetPath(ctx, "")
						if repositories.WriteTagRep != nil {
							existPath, _ := repositories.WriteTagRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write tag rep %s %s: %w", existPath, newPath)
							return nil, err
						}
						repositories.WriteTagRep = tagRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if *g.autoIDF {
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
							fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "text":
					textRep, err := reps.NewTextRepositorySQLite3Impl(ctx, filename)
					if err != nil {
						return nil, err
					}
					repositories.TextReps = append(repositories.TextReps, textRep)
					if rep.UseToWrite {
						newPath, _ := textRep.GetPath(ctx, "")
						if repositories.WriteTextRep != nil {
							existPath, _ := repositories.WriteTextRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write text rep %s %s: %w", existPath, newPath)
							return nil, err
						}
						repositories.WriteTextRep = textRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if *g.autoIDF {
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
							fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "notification":
					notificationRep, err := reps.NewNotificationRepositorySQLite3Impl(ctx, filename)
					if err != nil {
						return nil, err
					}
					repositories.NotificationReps = append(repositories.NotificationReps, notificationRep)
					if rep.UseToWrite {
						newPath, _ := notificationRep.GetPath(ctx, "")
						if repositories.WriteNotificationRep != nil {
							existPath, _ := repositories.WriteNotificationRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write notification rep %s %s: %w", existPath, newPath)
							return nil, err
						}
						repositories.WriteNotificationRep = notificationRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if *g.autoIDF {
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
							fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "rekyou":
					reKyouRep, err := reps.NewReKyouRepositorySQLite3Impl(ctx, filename, repositories)
					if err != nil {
						return nil, err
					}
					repositories.ReKyouReps.ReKyouRepositories = append(repositories.ReKyouReps.ReKyouRepositories, reKyouRep)
					repositories.Reps = append(repositories.Reps, reKyouRep)
					if rep.UseToWrite {
						newPath, _ := reKyouRep.GetPath(ctx, "")
						if repositories.WriteReKyouRep != nil {
							existPath, _ := repositories.WriteReKyouRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write reKyou rep %s %s: %w", existPath, newPath)
							return nil, err
						}
						repositories.WriteReKyouRep = reKyouRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if *g.autoIDF {
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
							fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
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
					idfKyouRep, err := reps.NewIDFDirRep(ctx, filename, idDBFilename, g.router, &autoIDF, &g.IDFIgnore, repositories)
					if err != nil {
						return nil, err
					}
					repositories.IDFKyouReps = append(repositories.IDFKyouReps, idfKyouRep)
					repositories.Reps = append(repositories.Reps, idfKyouRep)
					if rep.UseToWrite {
						newPath, _ := idfKyouRep.GetPath(ctx, "")
						if repositories.WriteIDFKyouRep != nil {
							existPath, _ := repositories.WriteIDFKyouRep.GetPath(ctx, "")
							err := fmt.Errorf("error conflict write idf kyou rep %s %s: %w", existPath, newPath)
							return nil, err
						}
						repositories.WriteIDFKyouRep = idfKyouRep
					}

					// ファイル更新があったときにキャッシュを更新する
					if *g.autoIDF {
						rep := idfKyouRep
						enableUpdateRepsCache := true
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

						ignoreFileNamePrefixes = append(ignoreFileNamePrefixes, filepath.ToSlash(filepath.Join(repFilename, ".gkill")))

						err = g.fileRepWatchCacheUpdater.RegisterWatchFileRep(cacheUpdater, repFilename, ignoreFileNamePrefixes, userID)
						if err != nil {
							fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
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

					// ファイル更新があったときにキャッシュを更新する
					rep := gpslogRep
					if *g.autoIDF {
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
							fmt.Errorf("error at register watch file rep. repfilename = %s userID = %s: %w", repFilename, userID, err)
							return nil, err
						}
					}

				case "git_commit_log":
					gitCommitLogRep, err := reps.NewGitRep(filename)
					if err != nil {
						return nil, err
					}
					repositories.GitCommitLogReps = append(repositories.GitCommitLogReps, gitCommitLogRep)
					repositories.Reps = append(repositories.Reps, gitCommitLogRep)
				}
			}
		}
		repositories.UpdateCache(ctx)
		g.gkillRepositories[userID][device] = repositories
		repositories, _ = repositoriesInUser[device]

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
		err = g.ConfigDAOs.MiShareInfoDAO.Close(ctx)
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
		err = g.ConfigDAOs.KFTLTemplateDAO.Close(ctx)
		if err != nil {
			return err
		}
		err = g.ConfigDAOs.TagStructDAO.Close(ctx)
		if err != nil {
			return err
		}
		err = g.ConfigDAOs.RepStructDAO.Close(ctx)
		if err != nil {
			return err
		}
		err = g.ConfigDAOs.DeviceStructDAO.Close(ctx)
		if err != nil {
			return err
		}
		err = g.ConfigDAOs.RepTypeStructDAO.Close(ctx)
		if err != nil {
			return err
		}

		g.ConfigDAOs.AccountDAO = nil
		g.ConfigDAOs.LoginSessionDAO = nil
		g.ConfigDAOs.FileUploadHistoryDAO = nil
		g.ConfigDAOs.MiShareInfoDAO = nil
		g.ConfigDAOs.ServerConfigDAO = nil
		g.ConfigDAOs.AppllicationConfigDAO = nil
		g.ConfigDAOs.RepositoryDAO = nil
		g.ConfigDAOs.KFTLTemplateDAO = nil
		g.ConfigDAOs.TagStructDAO = nil
		g.ConfigDAOs.RepStructDAO = nil
		g.ConfigDAOs.DeviceStructDAO = nil
		g.ConfigDAOs.RepTypeStructDAO = nil
	}

	g.ConfigDAOs = nil
	g.router = nil
	g.autoIDF = nil
	g.IDFIgnore = []string{}

	if g.enableOutputLogs {
		if e := g.infoLogFile.Close(); e != nil {
			err = fmt.Errorf("error at close info log file %s: %w: %w", g.infoLogFile.Name(), e, err)
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
	for _, rep := range reps.Reps {
		removeWatchTargetReps = append(removeWatchTargetReps, rep)
	}
	for _, tagRep := range reps.TagReps {
		removeWatchTargetReps = append(removeWatchTargetReps, tagRep)
	}
	for _, textRep := range reps.TextReps {
		removeWatchTargetReps = append(removeWatchTargetReps, textRep)
	}
	for _, notificationRep := range reps.NotificationReps {
		removeWatchTargetReps = append(removeWatchTargetReps, notificationRep)
	}
	for _, gpsLogRep := range reps.GPSLogReps {
		removeWatchTargetReps = append(removeWatchTargetReps, gpsLogRep)
	}

	for _, rep := range removeWatchTargetReps {
		filename, err := rep.GetPath(ctx, "")
		if err != nil {
			repName, _ := rep.GetRepName(ctx)
			fmt.Errorf("error at get path. repname = %s: %w", repName, err)
		}
		filename = filepath.ToSlash(filename)

		err = g.fileRepWatchCacheUpdater.RemoveWatchFileRep(filename, userID)
		if err != nil {
			fmt.Errorf("error at remove watch file rep. filename = %s userID = %s: %w", filename, userID, err)
		}
	}

	// 全Repを閉じる
	err = reps.Close(ctx)
	if err != nil {
		fmt.Errorf("error at close repositories: %w", err)
	}
	delete(g.gkillRepositories, userID)
	return true, nil
}

type closable interface {
	Close(ctx context.Context) error
}
