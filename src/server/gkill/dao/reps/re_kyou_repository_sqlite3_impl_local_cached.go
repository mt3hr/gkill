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

type reKyouRepositorySQLite3ImplLocalCached struct {
	userID               string
	originalDBFileName   string
	localCacheDBFileName string
	originalRep          ReKyouRepository
	localCachedRep       ReKyouRepository
	m                    sync.RWMutex

	fullConnect bool
	reps        *GkillRepositories
}

func NewReKyouRepositorySQLite3ImplLocalCached(ctx context.Context, userID string, filename string, fullConnect bool, reps *GkillRepositories) (ReKyouRepository, error) {
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

	originalRep, err := NewReKyouRepositorySQLite3Impl(ctx, filename, false, reps)
	if err != nil {
		err = fmt.Errorf("error at new rekyou rep: %w", err)
		return nil, err
	}

	localCachedRep, err := NewReKyouRepositorySQLite3Impl(ctx, localCacheDBFileName, fullConnect, reps)
	if err != nil {
		err = fmt.Errorf("error at new rekyou rep: %w", err)
		return nil, err
	}

	cachedRep := &reKyouRepositorySQLite3ImplLocalCached{
		userID:               userID,
		originalDBFileName:   filename,
		localCacheDBFileName: localCacheDBFileName,
		originalRep:          originalRep,
		localCachedRep:       localCachedRep,

		fullConnect: fullConnect,
		reps:        reps,

		m: sync.RWMutex{},
	}
	return cachedRep, nil
}

func (r *reKyouRepositorySQLite3ImplLocalCached) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	if query.UpdateCache {
		if err := r.UpdateCache(ctx); err != nil {
			return nil, fmt.Errorf("error at update cache: %w", err)
		}
	}
	r.m.RLock()
	defer r.m.RUnlock()
	return r.localCachedRep.FindKyous(ctx, query)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.localCachedRep.GetKyou(ctx, id, updateTime)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.localCachedRep.GetKyouHistories(ctx, id)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) GetPath(ctx context.Context, id string) (string, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.originalRep.GetPath(ctx, id)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) UpdateCache(ctx context.Context) error {
	r.m.Lock()
	defer r.m.Unlock()

	// 元DBがローカルキャッシュと同じなら、閉じる・消す・コピーし直す・開き直すを全部飛ばす。
	// 判定は os.Remove の前に置くこと。消したあとでは必ず「要コピー」になる。
	if !localRepCacheNeedsCopy(r.originalDBFileName, r.localCacheDBFileName) {
		return nil
	}

	err := r.localCachedRep.Close(ctx)
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
	if err = os.Remove(r.localCacheDBFileName); err != nil && !os.IsNotExist(err) {
		slog.Log(ctx, gkill_log.Warn, "skip local rep cache refresh", "file", fmt.Sprintf("%q", r.localCacheDBFileName), "error", fmt.Sprintf("%q", err))
		reopenedRep, reopenErr := NewReKyouRepositorySQLite3Impl(ctx, r.localCacheDBFileName, r.fullConnect, r.reps)
		if reopenErr != nil {
			return fmt.Errorf("error at reopen local cached rep %s: %w", r.localCacheDBFileName, reopenErr)
		}
		r.localCachedRep = reopenedRep
		return nil
	}

	localCacheDBFileName, err := localRepCacheDBFileName(r.userID, r.originalDBFileName)
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
	if err := copyLocalRepCacheDB(r.originalDBFileName, localCacheDBFileName); err != nil {
		return err
	}

	newLocalCachedRep, err := NewReKyouRepositorySQLite3Impl(ctx, localCacheDBFileName, r.fullConnect, r.reps)
	if err != nil {
		err = fmt.Errorf("error at new rekyou rep: %w", err)
		return err
	}
	r.localCachedRep = newLocalCachedRep
	return nil
}

// LastUpdateCacheChanged は常にtrueを返します。
//
// ReKyouのキャッシュ内容は自分のDBファイルだけでなく、
// 他repのLatestDataRepositoryAddress（ターゲット解決結果）にも依存します。
// GkillRepositories.UpdateCache はアドレス確定後にもう一度ここを更新するので、
// ファイルのmtimeで判定するとその2回目が丸ごと飛び、ターゲット未解決の中身が残ります。
// だから下層のsqlite3実装と同じく、常に「変更あり」を返します。
func (r *reKyouRepositorySQLite3ImplLocalCached) LastUpdateCacheChanged() bool {
	return true
}

func (r *reKyouRepositorySQLite3ImplLocalCached) GetRepName(ctx context.Context) (string, error) {
	return r.originalRep.GetRepName(ctx)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) Close(ctx context.Context) error {
	r.m.Lock()
	defer r.m.Unlock()
	err := r.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close: %w", err)
		return err
	}
	err = r.originalRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close: %w", err)
		return err
	}
	return nil
}

func (r *reKyouRepositorySQLite3ImplLocalCached) FindReKyou(ctx context.Context, query *find.FindQuery) ([]ReKyou, error) {
	if query.UpdateCache {
		if err := r.UpdateCache(ctx); err != nil {
			return nil, fmt.Errorf("error at update cache: %w", err)
		}
	}
	r.m.RLock()
	defer r.m.RUnlock()
	return r.localCachedRep.FindReKyou(ctx, query)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) GetReKyou(ctx context.Context, id string, updateTime *time.Time) (*ReKyou, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.localCachedRep.GetReKyou(ctx, id, updateTime)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) GetReKyouHistories(ctx context.Context, id string) ([]ReKyou, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.localCachedRep.GetReKyouHistories(ctx, id)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) AddReKyouInfo(ctx context.Context, rekyou ReKyou) error {
	err := func() error {
		r.m.Lock()
		defer r.m.Unlock()
		err := r.originalRep.AddReKyouInfo(ctx, rekyou)
		if err != nil {
			return err
		}
		return nil
	}()

	if err != nil {
		return err
	}
	return r.UpdateCache(ctx)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) GetReKyousAllLatest(ctx context.Context) ([]ReKyou, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.localCachedRep.GetReKyousAllLatest(ctx)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) GetRepositoriesWithoutReKyouRep(ctx context.Context) (*GkillRepositories, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.localCachedRep.GetRepositoriesWithoutReKyouRep(ctx)
}

func (r *reKyouRepositorySQLite3ImplLocalCached) UnWrapTyped() ([]ReKyouRepository, error) {
	return []ReKyouRepository{r.originalRep}, nil
}

func (r *reKyouRepositorySQLite3ImplLocalCached) UnWrap() ([]Repository, error) {
	return []Repository{r.originalRep}, nil
}

func (r *reKyouRepositorySQLite3ImplLocalCached) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	addrs, err := r.localCachedRep.GetLatestDataRepositoryAddress(ctx, updateCache)
	if err != nil {
		return nil, err
	}
	repName, err := r.GetRepName(ctx)
	if err != nil {
		return nil, err
	}
	for idx := range addrs {
		addrs[idx].LatestDataRepositoryName = repName
	}
	return addrs, nil
}
