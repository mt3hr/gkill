package reps

import (
	"context"
	"fmt"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

type kmemoRepositorySQLite3ImplLocalCached struct {
	userID               string
	originalDBFileName   string
	localCacheDBFileName string
	originalRep          KmemoRepository
	localCachedRep       KmemoRepository
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

func NewKmemoRepositorySQLite3ImplLocalCached(ctx context.Context, userID string, filename string, fullConnect bool) (KmemoRepository, error) {
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

	originalRep, err := NewKmemoRepositorySQLite3Impl(ctx, filename, false)
	if err != nil {
		err = fmt.Errorf("error at new kmemo rep: %w", err)
		return nil, err
	}

	localCachedRep, err := NewKmemoRepositorySQLite3Impl(ctx, localCacheDBFileName, fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new kmemo rep: %w", err)
		return nil, err
	}

	cachedRep := &kmemoRepositorySQLite3ImplLocalCached{
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

func (k *kmemoRepositorySQLite3ImplLocalCached) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	if query.UpdateCache {
		if err := k.UpdateCache(ctx); err != nil {
			return nil, fmt.Errorf("error at update cache: %w", err)
		}
	}
	k.m.RLock()
	defer k.m.RUnlock()
	return k.localCachedRep.FindKyous(ctx, query)
}

func (k *kmemoRepositorySQLite3ImplLocalCached) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	k.m.RLock()
	defer k.m.RUnlock()
	return k.localCachedRep.GetKyou(ctx, id, updateTime)
}

func (k *kmemoRepositorySQLite3ImplLocalCached) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	k.m.RLock()
	defer k.m.RUnlock()
	return k.localCachedRep.GetKyouHistories(ctx, id)
}

func (k *kmemoRepositorySQLite3ImplLocalCached) GetPath(ctx context.Context, id string) (string, error) {
	k.m.RLock()
	defer k.m.RUnlock()
	return k.originalRep.GetPath(ctx, id)
}

func (k *kmemoRepositorySQLite3ImplLocalCached) UpdateCache(ctx context.Context) error {
	k.m.Lock()
	defer k.m.Unlock()

	// 上位のキャッシュrepへ「元DBが変わったか」を伝えるための観測。
	// 基準が進むのは再構築が成功したときのCommitCacheRebuildだけなので、
	// 途中で失敗した回のぶんは次回も再構築される。
	k.cacheChange.refresh(k.originalDBFileName)
	k.skipRebuild = false

	// 元DBがローカルキャッシュと同じなら、閉じる・消す・コピーし直す・開き直すを全部飛ばす。
	// この判定を os.Remove のあとに置くと、消した直後なので必ず「要コピー」になり、
	// 変更のないrepまで LastUpdateCacheChanged() が true を返して
	// 上位のキャッシュrepが毎回フルリビルドする。
	if !localRepCacheNeedsCopy(k.originalDBFileName, k.localCacheDBFileName) {
		return nil
	}

	err := k.localCachedRep.Close(ctx)
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
	if err = os.Remove(k.localCacheDBFileName); err != nil && !os.IsNotExist(err) {
		slog.Log(ctx, gkill_log.Warn, "skip local rep cache refresh", "file", fmt.Sprintf("%q", k.localCacheDBFileName), "error", fmt.Sprintf("%q", err))
		k.skipRebuild = true
		reopenedRep, reopenErr := NewKmemoRepositorySQLite3Impl(ctx, k.localCacheDBFileName, k.fullConnect)
		if reopenErr != nil {
			return fmt.Errorf("error at reopen local cached rep %s: %w", k.localCacheDBFileName, reopenErr)
		}
		k.localCachedRep = reopenedRep
		return nil
	}

	localCacheDBFileName, err := localRepCacheDBFileName(k.userID, k.originalDBFileName)
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
	if err := copyLocalRepCacheDB(k.originalDBFileName, localCacheDBFileName); err != nil {
		return err
	}

	newLocalCachedRep, err := NewKmemoRepositorySQLite3Impl(ctx, localCacheDBFileName, k.fullConnect)
	if err != nil {
		err = fmt.Errorf("error at new kmemo rep: %w", err)
		return err
	}
	k.localCachedRep = newLocalCachedRep
	return nil
}

func (k *kmemoRepositorySQLite3ImplLocalCached) LastUpdateCacheChanged() bool {
	// ローカルキャッシュを更新できなかった回は、古いコピーから作り直させない。
	if k.skipRebuild {
		return false
	}
	return k.cacheChange.lastChanged()
}

// CommitCacheRebuild は上位のキャッシュrepが再構築に成功したときに呼ばれます。
func (k *kmemoRepositorySQLite3ImplLocalCached) CommitCacheRebuild() {
	k.cacheChange.commit()
}

func (k *kmemoRepositorySQLite3ImplLocalCached) GetRepName(ctx context.Context) (string, error) {
	return k.originalRep.GetRepName(ctx)
}

func (k *kmemoRepositorySQLite3ImplLocalCached) Close(ctx context.Context) error {
	k.m.Lock()
	defer k.m.Unlock()
	err := k.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close: %w", err)
		return err
	}
	err = k.originalRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close: %w", err)
		return err
	}
	return nil
}

func (k *kmemoRepositorySQLite3ImplLocalCached) FindKmemo(ctx context.Context, query *find.FindQuery) ([]Kmemo, error) {
	if query.UpdateCache {
		if err := k.UpdateCache(ctx); err != nil {
			return nil, fmt.Errorf("error at update cache: %w", err)
		}
	}
	k.m.RLock()
	defer k.m.RUnlock()
	return k.localCachedRep.FindKmemo(ctx, query)
}

func (k *kmemoRepositorySQLite3ImplLocalCached) GetKmemo(ctx context.Context, id string, updateTime *time.Time) (*Kmemo, error) {
	k.m.RLock()
	defer k.m.RUnlock()
	return k.localCachedRep.GetKmemo(ctx, id, updateTime)
}

func (k *kmemoRepositorySQLite3ImplLocalCached) GetKmemoHistories(ctx context.Context, id string) ([]Kmemo, error) {
	k.m.RLock()
	defer k.m.RUnlock()
	return k.localCachedRep.GetKmemoHistories(ctx, id)
}

func (k *kmemoRepositorySQLite3ImplLocalCached) AddKmemoInfo(ctx context.Context, kmemo Kmemo) error {
	err := func() error {
		k.m.Lock()
		defer k.m.Unlock()
		err := k.originalRep.AddKmemoInfo(ctx, kmemo)
		if err != nil {
			return err
		}
		return nil
	}()

	if err != nil {
		return err
	}
	return k.UpdateCache(ctx)
}

func (k *kmemoRepositorySQLite3ImplLocalCached) UnWrapTyped() ([]KmemoRepository, error) {
	return []KmemoRepository{k.originalRep}, nil
}

func (k *kmemoRepositorySQLite3ImplLocalCached) UnWrap() ([]Repository, error) {
	return []Repository{k.originalRep}, nil
}

func (k *kmemoRepositorySQLite3ImplLocalCached) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	addrs, err := k.localCachedRep.GetLatestDataRepositoryAddress(ctx, updateCache)
	if err != nil {
		return nil, err
	}
	repName, err := k.GetRepName(ctx)
	if err != nil {
		return nil, err
	}
	for idx := range addrs {
		addrs[idx].LatestDataRepositoryName = repName
	}
	return addrs, nil
}
