package reps

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

type miReKyouRepositorySQLite3ImplLocalCached struct {
	userID               string
	originalDBFileName   string
	localCacheDBFileName string
	originalRep          MiReKyouRepository
	localCachedRep       MiReKyouRepository
	m                    sync.RWMutex

	fullConnect bool
	reps        *GkillRepositories
}

func NewMiReKyouRepositorySQLite3ImplLocalCached(ctx context.Context, userID string, filename string, fullConnect bool, reps *GkillRepositories) (MiReKyouRepository, error) {
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

	originalRep, err := NewMiReKyouRepositorySQLite3Impl(ctx, filename, false, reps)
	if err != nil {
		err = fmt.Errorf("error at new mirekyou rep: %w", err)
		return nil, err
	}

	localCachedRep, err := NewMiReKyouRepositorySQLite3Impl(ctx, localCacheDBFileName, fullConnect, reps)
	if err != nil {
		err = fmt.Errorf("error at new mirekyou rep: %w", err)
		return nil, err
	}

	cachedRep := &miReKyouRepositorySQLite3ImplLocalCached{
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

func (m *miReKyouRepositorySQLite3ImplLocalCached) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	if query.UpdateCache {
		if err := m.UpdateCache(ctx); err != nil {
			return nil, fmt.Errorf("error at update cache: %w", err)
		}
	}
	m.m.RLock()
	defer m.m.RUnlock()
	return m.localCachedRep.FindKyous(ctx, query)
}

func (m *miReKyouRepositorySQLite3ImplLocalCached) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.localCachedRep.GetKyou(ctx, id, updateTime)
}

func (m *miReKyouRepositorySQLite3ImplLocalCached) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.localCachedRep.GetKyouHistories(ctx, id)
}

func (m *miReKyouRepositorySQLite3ImplLocalCached) GetPath(ctx context.Context, id string) (string, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.originalRep.GetPath(ctx, id)
}

func (m *miReKyouRepositorySQLite3ImplLocalCached) UpdateCache(ctx context.Context) error {
	m.m.Lock()
	defer m.m.Unlock()

	// 元DBがローカルキャッシュと同じなら、閉じる・消す・コピーし直す・開き直すを全部飛ばす。
	// 判定は os.Remove の前に置くこと。消したあとでは必ず「要コピー」になる。
	if !localRepCacheNeedsCopy(m.originalDBFileName, m.localCacheDBFileName) {
		return nil
	}

	err := m.localCachedRep.Close(ctx)
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
	if err = os.Remove(m.localCacheDBFileName); err != nil && !os.IsNotExist(err) {
		slog.Log(ctx, gkill_log.Warn, "skip local rep cache refresh", "file", fmt.Sprintf("%q", m.localCacheDBFileName), "error", fmt.Sprintf("%q", err))
		reopenedRep, reopenErr := NewMiReKyouRepositorySQLite3Impl(ctx, m.localCacheDBFileName, m.fullConnect, m.reps)
		if reopenErr != nil {
			return fmt.Errorf("error at reopen local cached rep %s: %w", m.localCacheDBFileName, reopenErr)
		}
		m.localCachedRep = reopenedRep
		return nil
	}

	localCacheDBFileName, err := localRepCacheDBFileName(m.userID, m.originalDBFileName)
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
	if err := copyLocalRepCacheDB(m.originalDBFileName, localCacheDBFileName); err != nil {
		return err
	}

	newLocalCachedRep, err := NewMiReKyouRepositorySQLite3Impl(ctx, localCacheDBFileName, m.fullConnect, m.reps)
	if err != nil {
		err = fmt.Errorf("error at new mirekyou rep: %w", err)
		return err
	}
	m.localCachedRep = newLocalCachedRep
	return nil
}

// LastUpdateCacheChanged は常にtrueを返します。
//
// MiReKyouのキャッシュ内容は自分のDBファイルだけでなく、
// 他repのLatestDataRepositoryAddress（ターゲット解決結果）にも依存します。
// GkillRepositories.UpdateCache はアドレス確定後にもう一度ここを更新するので、
// ファイルのmtimeで判定するとその2回目が丸ごと飛び、ターゲット未解決の中身が残ります。
// だから下層のsqlite3実装と同じく、常に「変更あり」を返します。
func (m *miReKyouRepositorySQLite3ImplLocalCached) LastUpdateCacheChanged() bool {
	return true
}

func (m *miReKyouRepositorySQLite3ImplLocalCached) GetRepName(ctx context.Context) (string, error) {
	return m.originalRep.GetRepName(ctx)
}

func (m *miReKyouRepositorySQLite3ImplLocalCached) Close(ctx context.Context) error {
	m.m.Lock()
	defer m.m.Unlock()
	err := m.localCachedRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close: %w", err)
		return err
	}
	err = m.originalRep.Close(ctx)
	if err != nil {
		err = fmt.Errorf("error at close: %w", err)
		return err
	}
	return nil
}

func (m *miReKyouRepositorySQLite3ImplLocalCached) FindMiReKyou(ctx context.Context, query *find.FindQuery) ([]MiReKyou, error) {
	if query.UpdateCache {
		if err := m.UpdateCache(ctx); err != nil {
			return nil, fmt.Errorf("error at update cache: %w", err)
		}
	}
	m.m.RLock()
	defer m.m.RUnlock()
	return m.localCachedRep.FindMiReKyou(ctx, query)
}

func (m *miReKyouRepositorySQLite3ImplLocalCached) GetMiReKyou(ctx context.Context, id string, updateTime *time.Time) (*MiReKyou, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.localCachedRep.GetMiReKyou(ctx, id, updateTime)
}

func (m *miReKyouRepositorySQLite3ImplLocalCached) GetMiReKyouHistories(ctx context.Context, id string) ([]MiReKyou, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.localCachedRep.GetMiReKyouHistories(ctx, id)
}

func (m *miReKyouRepositorySQLite3ImplLocalCached) AddMiReKyouInfo(ctx context.Context, mirekyou MiReKyou) error {
	err := func() error {
		m.m.Lock()
		defer m.m.Unlock()
		return m.originalRep.AddMiReKyouInfo(ctx, mirekyou)
	}()

	if err != nil {
		return err
	}
	return m.UpdateCache(ctx)
}

func (m *miReKyouRepositorySQLite3ImplLocalCached) GetMiReKyousAllLatest(ctx context.Context) ([]MiReKyou, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.localCachedRep.GetMiReKyousAllLatest(ctx)
}

func (m *miReKyouRepositorySQLite3ImplLocalCached) GetBoardNames(ctx context.Context) ([]string, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.localCachedRep.GetBoardNames(ctx)
}

func (m *miReKyouRepositorySQLite3ImplLocalCached) GetRepositoriesWithoutMiReKyouRep(ctx context.Context) (*GkillRepositories, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.localCachedRep.GetRepositoriesWithoutMiReKyouRep(ctx)
}

func (m *miReKyouRepositorySQLite3ImplLocalCached) UnWrapTyped() ([]MiReKyouRepository, error) {
	return []MiReKyouRepository{m.originalRep}, nil
}

func (m *miReKyouRepositorySQLite3ImplLocalCached) UnWrap() ([]Repository, error) {
	return []Repository{m.originalRep}, nil
}

func (m *miReKyouRepositorySQLite3ImplLocalCached) GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	addrs, err := m.localCachedRep.GetLatestDataRepositoryAddress(ctx, updateCache)
	if err != nil {
		return nil, err
	}
	repName, err := m.GetRepName(ctx)
	if err != nil {
		return nil, err
	}
	for idx := range addrs {
		addrs[idx].LatestDataRepositoryName = repName
	}
	return addrs, nil
}
