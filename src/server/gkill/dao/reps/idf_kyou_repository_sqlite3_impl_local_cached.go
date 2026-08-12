package reps

import (
	"context"
	"fmt"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

func NewIDFDirRepLocalCached(ctx context.Context, userID string, dir, dbFilename string, fullConnect bool, r *mux.Router, autoIDF bool, idfIgnore *[]string, repositoriesRef *GkillRepositories) (IDFKyouRepository, error) {
	localCacheDBFileName, err := localRepCacheDBFileName(userID, dbFilename)
	if err != nil {
		return nil, err
	}
	localCacheDBParentDirName, _ := filepath.Split(localCacheDBFileName)

	err = os.MkdirAll(localCacheDBParentDirName, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mk dir %s: %w", localCacheDBParentDirName, err)
		return nil, err
	}

	cacheStat, cacheStatErr := os.Stat(localCacheDBFileName)
	originalStat, originalStatErr := os.Stat(dbFilename)
	updateCache := originalStatErr != nil || cacheStatErr != nil || !originalStat.ModTime().Equal(cacheStat.ModTime()) || originalStat.Size() != cacheStat.Size()
	if updateCache {
		originalDBFile, err := os.Open(dbFilename)
		if err != nil {
			err = fmt.Errorf("error at open file %s: %w", dbFilename, err)
			return nil, err
		}
		defer func() {
			err := originalDBFile.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
		cacheDBFile, err := os.Create(localCacheDBFileName)
		if err != nil {
			err = fmt.Errorf("error at open file %s: %w", localCacheDBFileName, err)
			return nil, err
		}
		defer func() {
			err := cacheDBFile.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
		_, err = io.Copy(cacheDBFile, originalDBFile)
		if err != nil {
			err = fmt.Errorf("error at copy local cache db %s to %s: %w", dbFilename, localCacheDBFileName, err)
			return nil, err
		}
		os.Chtimes(localCacheDBFileName, originalStat.ModTime(), originalStat.ModTime())
	}

	originalRep, err := NewIDFDirRep(ctx, userID, dir, dbFilename, false, r, autoIDF, idfIgnore, repositoriesRef)
	if err != nil {
		err = fmt.Errorf("error at new idf dir rep: %w", err)
		return nil, err
	}

	localCachedRep, err := NewIDFDirRep(ctx, userID, dir, localCacheDBFileName, fullConnect, r, false, idfIgnore, repositoriesRef)
	if err != nil {
		err = fmt.Errorf("error at new idf dir rep: %w", err)
		return nil, err
	}

	cachedRep := &idfKyouRepositorySQLite3ImplLocalCached{
		userID:               userID,
		originalDBFileName:   dbFilename,
		localCacheDBFileName: localCacheDBFileName,
		originalRep:          originalRep,
		localCachedRep:       localCachedRep,

		repositoriesRef: repositoriesRef,
		r:               r,
		contentDir:      dir,
		fullConnect:     fullConnect,
		autoIDF:         autoIDF,
		idfIgnore:       idfIgnore,

		cacheChange: &dbFileChangeDetector{},

		m: sync.RWMutex{},
	}
	return cachedRep, nil
}

type idfKyouRepositorySQLite3ImplLocalCached struct {
	userID               string
	originalDBFileName   string
	localCacheDBFileName string
	originalRep          IDFKyouRepository
	localCachedRep       IDFKyouRepository
	m                    sync.RWMutex

	repositoriesRef *GkillRepositories
	r               *mux.Router
	contentDir      string
	fullConnect     bool
	autoIDF         bool
	idfIgnore       *[]string
	// cacheChange は元DBファイルが前回のキャッシュ再構築から変わったかを見ます。
	// 上位のキャッシュrepはこれが false のときフルリビルドを飛ばします。
	cacheChange *dbFileChangeDetector

	// skipRebuild はローカルキャッシュを更新できなかった回だけ立ちます。
	// 古いコピーのまま上位に再構築させると、そのあと基準が進んで
	// 新しい内容が二度と取り込まれなくなるため、その回だけ再構築を見送らせます。
	skipRebuild bool
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	if query.UpdateCache {
		if err := i.UpdateCache(ctx); err != nil {
			return nil, fmt.Errorf("error at update cache: %w", err)
		}
	}
	i.m.RLock()
	defer i.m.RUnlock()
	return i.localCachedRep.FindKyous(ctx, query)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	i.m.RLock()
	defer i.m.RUnlock()
	return i.localCachedRep.GetKyou(ctx, id, updateTime)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	i.m.RLock()
	defer i.m.RUnlock()
	return i.localCachedRep.GetKyouHistories(ctx, id)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) GetPath(ctx context.Context, id string) (string, error) {
	i.m.RLock()
	defer i.m.RUnlock()
	return i.originalRep.GetPath(ctx, id)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) UpdateCache(ctx context.Context) error {
	i.m.Lock()
	defer i.m.Unlock()

	// 上位のキャッシュrepへ「元DBが変わったか」を伝えるための観測。
	// 基準が進むのは再構築が成功したときのCommitCacheRebuildだけなので、
	// 途中で失敗した回のぶんは次回も再構築される。
	i.cacheChange.refresh(i.originalDBFileName)
	i.skipRebuild = false

	// 元DBがローカルキャッシュと同じなら、閉じる・消す・コピーし直す・開き直すを全部飛ばす。
	// この判定を os.Remove のあとに置くと、消した直後なので必ず「要コピー」になり、
	// 変更のないrepまで LastUpdateCacheChanged() が true を返して
	// 上位のキャッシュrepが毎回フルリビルドする。
	if !localRepCacheNeedsCopy(i.originalDBFileName, i.localCacheDBFileName) {
		return nil
	}

	err := i.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at update cache: %w", err)
		return err
	}

	// ローカルキャッシュファイルを消せなくても致命的なエラーにはしない。
	// 別のRepositoryや別プロセスが同じファイルを開いていると
	// Windowsでは削除が共有違反で失敗する。
	// ここで error を返すと GetRepositories 全体が失敗し、
	// リポジトリがキャッシュに登録されないままリクエスト毎に再ロードされ続けるため、
	// この回の再取得だけ諦めて、閉じたハンドルを開き直して継続する。
	if err = os.Remove(i.localCacheDBFileName); err != nil && !os.IsNotExist(err) {
		slog.Log(ctx, gkill_log.Warn, "skip local rep cache refresh", "file", fmt.Sprintf("%q", i.localCacheDBFileName), "error", fmt.Sprintf("%q", err))
		i.skipRebuild = true
		reopenedRep, reopenErr := NewIDFDirRep(ctx, i.userID, i.contentDir, i.localCacheDBFileName, i.fullConnect, i.r, i.autoIDF, i.idfIgnore, i.repositoriesRef)
		if reopenErr != nil {
			return fmt.Errorf("error at reopen local cached rep %s: %w", i.localCacheDBFileName, reopenErr)
		}
		i.localCachedRep = reopenedRep
		return nil
	}

	localCacheDBFileName, err := localRepCacheDBFileName(i.userID, i.originalDBFileName)
	if err != nil {
		return err
	}
	localCacheDBParentDirName, _ := filepath.Split(localCacheDBFileName)

	err = os.MkdirAll(localCacheDBParentDirName, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at mk dir %s: %w", localCacheDBParentDirName, err)
		return err
	}

	// ここへ来るのは冒頭の判定でコピーが要ると分かったときだけ。
	if err := copyLocalRepCacheDB(i.originalDBFileName, localCacheDBFileName); err != nil {
		return err
	}

	newLocalCachedRep, err := NewIDFDirRep(ctx, i.userID, i.contentDir, localCacheDBFileName, i.fullConnect, i.r, i.autoIDF, i.idfIgnore, i.repositoriesRef)
	if err != nil {
		err = fmt.Errorf("error at new idf dir rep: %w", err)
		return err
	}
	i.localCachedRep = newLocalCachedRep
	return nil
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) LastUpdateCacheChanged() bool {
	// ローカルキャッシュを更新できなかった回は、古いコピーから作り直させない。
	if i.skipRebuild {
		return false
	}
	return i.cacheChange.lastChanged()
}

// CommitCacheRebuild は上位のキャッシュrepが再構築に成功したときに呼ばれます。
func (i *idfKyouRepositorySQLite3ImplLocalCached) CommitCacheRebuild() {
	i.cacheChange.commit()
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) GetRepName(ctx context.Context) (string, error) {
	return i.originalRep.GetRepName(ctx)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) Close(ctx context.Context) error {
	i.m.Lock()
	defer i.m.Unlock()
	err := i.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close: %w", err)
		return err
	}
	err = i.originalRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close: %w", err)
		return err
	}
	return nil
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) FindIDFKyou(ctx context.Context, query *find.FindQuery) ([]IDFKyou, error) {
	if query.UpdateCache {
		if err := i.UpdateCache(ctx); err != nil {
			return nil, fmt.Errorf("error at update cache: %w", err)
		}
	}
	i.m.RLock()
	defer i.m.RUnlock()
	return i.localCachedRep.FindIDFKyou(ctx, query)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) GetIDFKyou(ctx context.Context, id string, updateTime *time.Time) (*IDFKyou, error) {
	i.m.RLock()
	defer i.m.RUnlock()
	return i.localCachedRep.GetIDFKyou(ctx, id, updateTime)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) GetIDFKyouByTargetFile(ctx context.Context, targetFile string) (*IDFKyou, error) {
	i.m.RLock()
	defer i.m.RUnlock()
	return i.localCachedRep.GetIDFKyouByTargetFile(ctx, targetFile)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) GetIDFKyouHistories(ctx context.Context, id string) ([]IDFKyou, error) {
	i.m.RLock()
	defer i.m.RUnlock()
	return i.localCachedRep.GetIDFKyouHistories(ctx, id)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) IDF(ctx context.Context) error {
	err := func() error {
		i.m.Lock()
		defer i.m.Unlock()
		err := i.originalRep.IDF(ctx)
		if err != nil {
			return err
		}
		return nil
	}()

	if err != nil {
		return err
	}
	return i.UpdateCache(ctx)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) AddIDFKyouInfo(ctx context.Context, idfKyou IDFKyou) error {
	err := func() error {
		i.m.Lock()
		defer i.m.Unlock()
		err := i.originalRep.AddIDFKyouInfo(ctx, idfKyou)
		if err != nil {
			return err
		}
		return nil
	}()

	if err != nil {
		return err
	}
	return i.UpdateCache(ctx)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) HandleFileServe(w http.ResponseWriter, r *http.Request) {
	i.originalRep.HandleFileServe(w, r)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) GenerateThumbCache(ctx context.Context) error {
	return i.originalRep.GenerateThumbCache(ctx)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) ClearThumbCache(userID string) error {
	return i.originalRep.ClearThumbCache(userID)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) GenerateVideoCache(ctx context.Context) error {
	return i.originalRep.GenerateVideoCache(ctx)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) ClearVideoCache(userID string) error {
	return i.originalRep.ClearVideoCache(userID)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) ClearZipCache(userID string) error {
	return i.originalRep.ClearZipCache(userID)
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) UnWrapTyped() ([]IDFKyouRepository, error) {
	return []IDFKyouRepository{i.originalRep}, nil
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) UnWrap() ([]Repository, error) {
	return []Repository{i.originalRep}, nil
}

func (i *idfKyouRepositorySQLite3ImplLocalCached) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	addrs, err := i.localCachedRep.GetLatestDataRepositoryAddress(ctx, updateCache)
	if err != nil {
		return nil, err
	}
	repName, err := i.GetRepName(ctx)
	if err != nil {
		return nil, err
	}
	for idx := range addrs {
		addrs[idx].LatestDataRepositoryName = repName
	}
	return addrs, nil
}
