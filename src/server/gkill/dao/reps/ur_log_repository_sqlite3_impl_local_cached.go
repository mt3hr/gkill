package reps

import (
	"context"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

type urlogRepositorySQLite3ImplLocalCached struct {
	userID               string
	originalDBFileName   string
	localCacheDBFileName string
	originalRep          URLogRepository
	localCachedRep       URLogRepository
	m                    sync.RWMutex

	fullConnect bool

	// cacheChange は元DBファイルが前回のキャッシュ再構築から変わったかを見ます。
	// 上位のキャッシュrepはこれが false のときフルリビルドを飛ばします。
	cacheChange *dbFileChangeDetector

	// skipRebuild はローカルキャッシュを更新できなかった回だけ立ちます。
	// 古いコピーのまま上位に再構築させると、そのあと基準が進んで
	// 新しい内容が二度と取り込まれなくなるため、その回だけ再構築を見送らせます。
	skipRebuild bool
}

func NewURLogRepositorySQLite3ImplLocalCached(ctx context.Context, userID string, filename string, fullConnect bool) (URLogRepository, error) {
	localCacheDBFileName, err := localRepCacheDBFileName(userID, filename)
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
	originalStat, originalStatErr := os.Stat(filename)
	updateCache := originalStatErr != nil || cacheStatErr != nil || !originalStat.ModTime().Equal(cacheStat.ModTime()) || originalStat.Size() != cacheStat.Size()
	if updateCache {
		originalDBFile, err := os.Open(filename)
		if err != nil {
			err = fmt.Errorf("error at open file %s: %w", filename, err)
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
			err = fmt.Errorf("error at copy local cache db %s to %s: %w", filename, localCacheDBFileName, err)
			return nil, err
		}
		os.Chtimes(localCacheDBFileName, originalStat.ModTime(), originalStat.ModTime())
	}

	originalRep, err := NewURLogRepositorySQLite3Impl(ctx, filename, false)
	if err != nil {
		err = fmt.Errorf("error at new urlog rep: %w", err)
		return nil, err
	}

	localCachedRep, err := NewURLogRepositorySQLite3Impl(ctx, localCacheDBFileName, fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new urlog rep: %w", err)
		return nil, err
	}

	cachedRep := &urlogRepositorySQLite3ImplLocalCached{
		userID:               userID,
		originalDBFileName:   filename,
		localCacheDBFileName: localCacheDBFileName,
		originalRep:          originalRep,
		localCachedRep:       localCachedRep,

		fullConnect: fullConnect,

		cacheChange: &dbFileChangeDetector{},

		m: sync.RWMutex{},
	}
	return cachedRep, nil
}

func (u *urlogRepositorySQLite3ImplLocalCached) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	if query.UpdateCache {
		if err := u.UpdateCache(ctx); err != nil {
			return nil, fmt.Errorf("error at update cache: %w", err)
		}
	}
	u.m.RLock()
	defer u.m.RUnlock()
	return u.localCachedRep.FindKyous(ctx, query)
}

func (u *urlogRepositorySQLite3ImplLocalCached) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	return u.localCachedRep.GetKyou(ctx, id, updateTime)
}

func (u *urlogRepositorySQLite3ImplLocalCached) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	return u.localCachedRep.GetKyouHistories(ctx, id)
}

func (u *urlogRepositorySQLite3ImplLocalCached) GetPath(ctx context.Context, id string) (string, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	return u.originalRep.GetPath(ctx, id)
}

func (u *urlogRepositorySQLite3ImplLocalCached) UpdateCache(ctx context.Context) error {
	u.m.Lock()
	defer u.m.Unlock()

	// 上位のキャッシュrepへ「元DBが変わったか」を伝えるための観測。
	// 基準が進むのは再構築が成功したときのCommitCacheRebuildだけなので、
	// 途中で失敗した回のぶんは次回も再構築される。
	u.cacheChange.refresh(u.originalDBFileName)
	u.skipRebuild = false

	// 元DBがローカルキャッシュと同じなら、閉じる・消す・コピーし直す・開き直すを全部飛ばす。
	// この判定を os.Remove のあとに置くと、消した直後なので必ず「要コピー」になり、
	// 変更のないrepまで LastUpdateCacheChanged() が true を返して
	// 上位のキャッシュrepが毎回フルリビルドする。
	if !localRepCacheNeedsCopy(u.originalDBFileName, u.localCacheDBFileName) {
		return nil
	}

	err := u.localCachedRep.Close(ctx)
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
	if err = os.Remove(u.localCacheDBFileName); err != nil && !os.IsNotExist(err) {
		slog.Log(ctx, gkill_log.Warn, "skip local rep cache refresh", "file", fmt.Sprintf("%q", u.localCacheDBFileName), "error", fmt.Sprintf("%q", err))
		u.skipRebuild = true
		reopenedRep, reopenErr := NewURLogRepositorySQLite3Impl(ctx, u.localCacheDBFileName, u.fullConnect)
		if reopenErr != nil {
			return fmt.Errorf("error at reopen local cached rep %s: %w", u.localCacheDBFileName, reopenErr)
		}
		u.localCachedRep = reopenedRep
		return nil
	}

	localCacheDBFileName, err := localRepCacheDBFileName(u.userID, u.originalDBFileName)
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
	if err := copyLocalRepCacheDB(u.originalDBFileName, localCacheDBFileName); err != nil {
		return err
	}

	newLocalCachedRep, err := NewURLogRepositorySQLite3Impl(ctx, localCacheDBFileName, u.fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new urlog rep: %w", err)
		return err
	}
	u.localCachedRep = newLocalCachedRep
	return nil
}

func (u *urlogRepositorySQLite3ImplLocalCached) LastUpdateCacheChanged() bool {
	// ローカルキャッシュを更新できなかった回は、古いコピーから作り直させない。
	if u.skipRebuild {
		return false
	}
	return u.cacheChange.lastChanged()
}

// CommitCacheRebuild は上位のキャッシュrepが再構築に成功したときに呼ばれます。
func (u *urlogRepositorySQLite3ImplLocalCached) CommitCacheRebuild() {
	u.cacheChange.commit()
}

func (u *urlogRepositorySQLite3ImplLocalCached) GetRepName(ctx context.Context) (string, error) {
	return u.originalRep.GetRepName(ctx)
}

func (u *urlogRepositorySQLite3ImplLocalCached) Close(ctx context.Context) error {
	u.m.Lock()
	defer u.m.Unlock()
	err := u.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close: %w", err)
		return err
	}
	err = u.originalRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close: %w", err)
		return err
	}
	return nil
}

func (u *urlogRepositorySQLite3ImplLocalCached) FindURLog(ctx context.Context, query *find.FindQuery) ([]URLog, error) {
	if query.UpdateCache {
		if err := u.UpdateCache(ctx); err != nil {
			return nil, fmt.Errorf("error at update cache: %w", err)
		}
	}
	u.m.RLock()
	defer u.m.RUnlock()
	return u.localCachedRep.FindURLog(ctx, query)
}

func (u *urlogRepositorySQLite3ImplLocalCached) GetURLog(ctx context.Context, id string, updateTime *time.Time) (*URLog, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	return u.localCachedRep.GetURLog(ctx, id, updateTime)
}

func (u *urlogRepositorySQLite3ImplLocalCached) GetURLogHistories(ctx context.Context, id string) ([]URLog, error) {
	u.m.RLock()
	defer u.m.RUnlock()
	return u.localCachedRep.GetURLogHistories(ctx, id)
}

func (u *urlogRepositorySQLite3ImplLocalCached) AddURLogInfo(ctx context.Context, urlog URLog) error {
	err := func() error {
		u.m.Lock()
		defer u.m.Unlock()
		err := u.originalRep.AddURLogInfo(ctx, urlog)
		if err != nil {
			return err
		}
		return nil
	}()

	if err != nil {
		return err
	}
	return u.UpdateCache(ctx)
}

func (u *urlogRepositorySQLite3ImplLocalCached) UnWrapTyped() ([]URLogRepository, error) {
	return []URLogRepository{u.originalRep}, nil
}

func (u *urlogRepositorySQLite3ImplLocalCached) UnWrap() ([]Repository, error) {
	return []Repository{u.originalRep}, nil
}

func (u *urlogRepositorySQLite3ImplLocalCached) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	addrs, err := u.localCachedRep.GetLatestDataRepositoryAddress(ctx, updateCache)
	if err != nil {
		return nil, err
	}
	repName, err := u.GetRepName(ctx)
	if err != nil {
		return nil, err
	}
	for idx := range addrs {
		addrs[idx].LatestDataRepositoryName = repName
	}
	return addrs, nil
}
